package main

import (
	"runtime"
	"runtime/debug"
	"time"

	"github.com/f-secure-foundry/tamago/soc/imx6"
	"github.com/wallera-computer/wallera/apps"
	"github.com/wallera-computer/wallera/apps/cosmos"
	"github.com/wallera-computer/wallera/crypto"
	"go.uber.org/zap"
)

var (
	// Build is a string which contains build user, host and date.
	Build string

	// Revision contains the git revision (last hash and/or tag).
	Revision string
)

func init() {
	l := logger()

	l.Infow("wallera started", "GOOS", runtime.GOOS, "GOARCH", runtime.GOARCH, "GOVERSION", runtime.Version(), "revision", Revision, "build", Build)

	if !imx6.Native {
		l.Fatal("running wallera on emulated hardware is not supported")
	}

	loadDebugAccessory()

	if err := imx6.SetARMFreq(imx6.FreqLow); err != nil {
		l.Warnf("WARNING: error setting ARM frequency: %v", err)
	}
}

func main() {
	defer catchPanic()

	l := logger()

	t := crypto.NewDumbToken()

	ah := apps.NewHandler()
	ah.Register(&cosmos.Cosmos{
		Token: t,
	})

	hh := newHidHandler(l, ah)

	if err := startUSB(hh); err != nil {
		l.Panic(err)
	}

	resetBoard()
}

// catchPanic catches every panic(), sets the LEDs into error mode and prints the stacktrace.
func catchPanic() {
	l := logger()
	if r := recover(); r != nil {
		l.Errorf("panic: %v\n\n", r)
		l.Error(string(debug.Stack()))
		l.Warn("rebooting in 1 second...")

		time.Sleep(1 * time.Second)
		resetBoard()
	}
}

// since we're in a critical configuration phase, panic on error.
func notErr(e error, l *zap.SugaredLogger) {
	if e != nil {
		l.Panic(e)
	}
}
