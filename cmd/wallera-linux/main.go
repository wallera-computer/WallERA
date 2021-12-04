package main

import (
	"bytes"
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

			_, err = hidRx.Write(data)
			notErr(err)
		}
	}()

	log.Println("running...")

	<-sigs

	log.Println("exiting, call this binary with the '-clean' flag to clean hidg entries")
}

type hidHandler struct {
	ah *apps.Handler

	outboundMutex  sync.Mutex
	outboundBuffer []byte
}

func (h *hidHandler) writeOutbound(data []byte) {
	h.outboundMutex.Lock()
	defer h.outboundMutex.Unlock()

	h.outboundBuffer = data
}

func (h *hidHandler) readOutbound() []byte {
	h.outboundMutex.Lock()
	defer h.outboundMutex.Unlock()

	ret := &bytes.Buffer{}
	ret.Write(h.outboundBuffer)

	h.outboundBuffer = []byte{}

	return ret.Bytes()
}

func (h *hidHandler) Tx() ([]byte, error) {
	return h.readOutbound(), nil
}

func (h *hidHandler) Rx(input []byte) ([]byte, error) {
	log.Println("input bytes:", input, "length:", len(input))

	frame, err := usb.ParseHIDFrame(input)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("HID frame: %+v\n", frame)

	session, err := usb.NewSession(frame)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("session: %+v\n", session)

	packet, err := session.CAPDU()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("apdu packet: %+v\n", packet)

	resp, err := h.ah.Handle(packet)

	resp = session.FormatResponse(resp)

	log.Println("response:", resp, err)

	if resp != nil {
		h.writeOutbound(resp)
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
