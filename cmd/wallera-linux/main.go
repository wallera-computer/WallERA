package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/wallera-computer/wallera/apps"
	"github.com/wallera-computer/wallera/apps/cosmos"
	"github.com/wallera-computer/wallera/usb"
)

func cliArgs() (hidg, configfsPath string, mustClean bool) {
	flag.StringVar(&hidg, "hidg", "/dev/hidg0", "/dev/hidgX file descriptor path")
	flag.StringVar(&configfsPath, "configfs-path", "/sys/kernel/config", "configfs path")
	flag.BoolVar(&mustClean, "clean", false, "clean existing hidg descriptors and exit")
	flag.Parse()

	return
}

func hidgExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func main() {
	hidg, configfsPath, mustClean := cliArgs()

	if mustClean {
		if err := cleanupHidg(configfsPath); err != nil {
			panic(err)
		}

		return
	}

	if !hidgExists(hidg) {
		log.Println("configuring hidg")
		if err := configureHidg(configfsPath); err != nil {
			panic(err)
		}
	} else {
		log.Println("hidg already configured, using pre-existing one")
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	hidRx, err := os.OpenFile(hidg, os.O_RDWR, 0666)
	notErr(err)

	log.Println("done, polling...")

	// add 50ms delay in both rx and tx
	// we don't wanna burn laptop cpus :^)

	ah := apps.NewHandler()
	ah.Register(&cosmos.Cosmos{})

	ha := hidHandler{
		ah: ah,
	}
	// rx
	go func() {
		for {
			time.Sleep(50 * time.Millisecond)
			buf := make([]byte, 64)
			_, err := hidRx.Read(buf)
			notErr(err)

			_, err = ha.Rx(buf)
			if err != nil {
				log.Println("rx error:", err)
				continue
			}
		}
	}()

	go func() {
		for {
			time.Sleep(50 * time.Millisecond)
			data, err := ha.Tx()
			if err != nil {
				log.Println("tx error:", err)
				continue
			}

			if data == nil {
				continue
			}

			for _, chunk := range data {
				_, err = hidRx.Write(chunk)
				notErr(err)
				log.Println("written chunk", chunk)
			}
		}
	}()

	log.Println("running...")

	<-sigs

	log.Println("exiting, call this binary with the '-clean' flag to clean hidg entries")
}

type hidHandler struct {
	ah *apps.Handler

	outboundMutex  sync.Mutex
	outboundBuffer [][]byte
	session        *usb.Session
}

func (h *hidHandler) writeOutbound(data [][]byte) {
	h.outboundMutex.Lock()
	defer h.outboundMutex.Unlock()

	h.outboundBuffer = data
}

func (h *hidHandler) readOutbound() [][]byte {
	h.outboundMutex.Lock()
	defer h.outboundMutex.Unlock()

	m := make([][]byte, len(h.outboundBuffer))
	copy(m, h.outboundBuffer)
	if h.session != nil {
		if !h.session.ShouldReadMore {
			h.session = nil
		}
	}

	h.outboundBuffer = make([][]byte, 0)

	return m
}

func (h *hidHandler) Tx() ([][]byte, error) {
	return h.readOutbound(), nil
}

func (h *hidHandler) Rx(input []byte) ([]byte, error) {
	log.Println("input bytes:", input, "length:", len(input))

	if h.session == nil {
		s, err := usb.NewSession(input)
		if err != nil {
			log.Fatal(err)
		}
		h.session = &s
	} else {
		err := h.session.ReadData(input)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("session: %+v\n", h.session)

	if h.session.ShouldReadMore {
		log.Println("should still read more data, continuing")
		return nil, nil
	}

	resp, err := h.ah.Handle(h.session.Data())
	if err != nil {
		log.Fatal(err)
	}

	chunks := h.session.FormatResponse(resp)

	if chunks != nil {
		h.writeOutbound(chunks)
	}

	return nil, nil
}

func h(in []byte) string {
	return hex.EncodeToString(in)
}

// since we're in a critical configuration phase, panic on error.
func notErr(e error) {
	if e != nil {
		panic(e)
	}
}
