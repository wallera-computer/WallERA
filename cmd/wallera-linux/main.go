package main

import (
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wallera-computer/wallera/apps"
	"github.com/wallera-computer/wallera/apps/cosmos"
	"github.com/wallera-computer/wallera/log"
	"github.com/wallera-computer/wallera/usb"
	"go.uber.org/zap"
)

type args struct {
	hidg          string
	configfsPath  string
	mustClean     bool
	mustSetupHidg bool
}

func cliArgs() args {
	a := args{}

	flag.StringVar(&a.hidg, "hidg", "/dev/hidg0", "/dev/hidgX file descriptor path")
	flag.StringVar(&a.configfsPath, "configfs-path", "/sys/kernel/config", "configfs path")
	flag.BoolVar(&a.mustClean, "clean", false, "clean existing hidg descriptors and exit")
	flag.BoolVar(&a.mustSetupHidg, "setup", false, "sets up dummy_hcd device and exits")
	flag.Parse()

	return a
}

func hidgExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func main() {
	a := cliArgs()
	l := log.Development().Sugar()

	if a.mustClean {
		if err := cleanupHidg(a.configfsPath); err != nil {
			l.Panic(err)
		}

		return
	}

	shouldSetupHidg := !hidgExists(a.hidg) || a.mustClean

	if shouldSetupHidg {
		l.Info("configuring hidg")
		if err := configureHidg(a.configfsPath); err != nil {
			l.Panic(err)
		}
	} else {
		l.Info("hidg already configured, using pre-existing one")
	}

	if a.mustSetupHidg {
		// we exit here
		return
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	hidRx, err := os.OpenFile(a.hidg, os.O_RDWR, 0666)
	notErr(err, l)

	l.Info("done, polling...")

	// add 50ms delay in both rx and tx
	// we don't wanna burn laptop cpus :^)

	t := NewDumbToken()

	ah := apps.NewHandler()
	ah.Register(&cosmos.Cosmos{
		Token: t,
	})

	ha := hidHandler{
		ah:           ah,
		outboundChan: make(chan [][]byte),
	}

	// rx
	go func() {
		for {
			time.Sleep(50 * time.Millisecond)
			buf := make([]byte, 64)
			_, err := hidRx.Read(buf)
			notErr(err, l)

			_, err = ha.Rx(buf, l)
			if err != nil {
				l.Error("rx error: ", err)
				continue
			}
		}
	}()

	go func() {
		for {
			time.Sleep(50 * time.Millisecond)
			data, err := ha.Tx()
			if err != nil {
				l.Error("tx error: ", err)
				continue
			}

			if data == nil {
				continue
			}

			for i, chunk := range data {
				_, err = hidRx.Write(chunk)
				notErr(err, l)
				l.Infow("written chunk", "index", i, "chunk", chunk)
			}
		}
	}()

	l.Info("running...")

	<-sigs

	l.Info("exiting, call this binary with the '-clean' flag to clean hidg entries")
}

type hidHandler struct {
	ah *apps.Handler

	outboundChan chan [][]byte
	session      *usb.Session
}

// writeOutbound will block until h.outboundChan gets read on the other
// side.
func (h *hidHandler) writeOutbound(data [][]byte) {
	h.outboundChan <- data
}

// readOutbound will block until h.outboundChan gets written on the other
// side.
func (h *hidHandler) readOutbound() [][]byte {
	return <-h.outboundChan
}

func (h *hidHandler) Tx() ([][]byte, error) {
	return h.readOutbound(), nil
}

func (h *hidHandler) Rx(input []byte, l *zap.SugaredLogger) ([]byte, error) {
	l.Debugw("handling rx", "input bytes", input, "length", len(input))

	if h.session == nil {
		s, err := usb.NewSession(input, l)
		notErr(err, l)
		h.session = &s
	} else {
		err := h.session.ReadData(input)
		l.Errorw("cannot read input data", "error", err)
		h.writeOutbound(
			h.session.FormatResponse(
				apps.PackageResponse(nil, apps.APDUCommandNotAllowed),
			),
		)
		return nil, nil
	}

	l.Debugw("handling session", "data", h.session)

	if h.session.ShouldReadMore {
		l.Debug("should still read more data, continuing")
		return nil, nil
	}

	resp, err := h.ah.Handle(h.session.Data())
	if err != nil {
		l.Errorw("cannot handle session data", "error", err)
	}

	if resp == nil {
		h.session = nil
		return nil, nil
	}

	chunks := h.session.FormatResponse(resp)

	if chunks != nil {
		h.writeOutbound(chunks)
		h.session = nil
	}

	return nil, nil
}

// since we're in a critical configuration phase, panic on error.
func notErr(e error, l *zap.SugaredLogger) {
	if e != nil {
		l.Panic(e)
	}
}
