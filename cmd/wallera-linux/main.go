package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hsanjuan/go-nfctype4/apdu"
)

var (
	// X.509 attestation certificate, sent along in registration requests
	attestationCertificate []byte

	// ECDSA private key, used to sign registration requests
	attestationPrivkey []byte
)

// CosmosCLA ...
//go:generate stringer -type CosmosCLA
type CosmosCLA byte

const (
	claGetVersion       CosmosCLA = 0x00
	claSignSecp256K1    CosmosCLA = 0x02
	claGetAddrSecp256K1 CosmosCLA = 0x04
)

// HIDFrame ...
type HIDFrame struct {
	ChannelID   uint16
	Tag         uint8
	PacketIndex uint16
	DataLength  uint16
	Data        [57]byte
}

func readMessage(in []byte) (HIDFrame, error) {
	ret := HIDFrame{}
	return ret, binary.Read(bytes.NewReader(in), binary.BigEndian, &ret)
}

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

	ha := hidHandler{}
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

	/*go func() {
		for {
			time.Sleep(50 * time.Millisecond)
			data, err := hid.Tx(nil, nil)
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
	}()*/

	log.Println("running...")

	<-sigs

	log.Println("exiting, call this binary with the '-clean' flag to clean hidg entries")
}

type hidHandler struct{}

func (hidHandler) Rx(input []byte) ([]byte, error) {
	log.Println("input bytes:", input, "length:", len(input))
	msg, err := readMessage(input)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("HID frame: %+v\n", msg)

	packet := apdu.CAPDU{}
	_, err = packet.Unmarshal(msg.Data[:])
	if err != nil {
		log.Println("apdu decode error,", err)
		return nil, nil
	}

	log.Printf("apdu packet: %+v\n", packet)

	log.Println("requested command:", CosmosCLA(packet.INS).String())
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
