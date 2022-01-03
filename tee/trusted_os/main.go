// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"time"
	_ "unsafe"

	usbarmory "github.com/f-secure-foundry/tamago/board/f-secure/usbarmory/mark-two"
	"github.com/f-secure-foundry/tamago/dma"
	"github.com/f-secure-foundry/tamago/soc/imx6"
	"github.com/f-secure-foundry/tamago/soc/imx6/dcp"
	"go.uber.org/zap"

	"github.com/wallera-computer/wallera/log"
	"github.com/wallera-computer/wallera/tee/cryptography_applet/info"
	"github.com/wallera-computer/wallera/tee/mem"
	"github.com/wallera-computer/wallera/tee/trusted_os/angel"
	"github.com/wallera-computer/wallera/tee/trusted_os/tz"
)

const (
	sshPort   = 22
	deviceIP  = "10.0.0.1"
	deviceMAC = "1a:55:89:a2:69:41"
	hostMAC   = "1a:55:89:a2:69:42"
)

//go:linkname ramStart runtime.ramStart
var ramStart uint32 = mem.SecureStart

//go:linkname ramSize runtime.ramSize
var ramSize uint32 = mem.SecureSize

var l *zap.SugaredLogger

func init() {
	l = log.Development().Sugar()

	if imx6.Native {
		if err := imx6.SetARMFreq(900); err != nil {
			panic(fmt.Sprintf("WARNING: error setting ARM frequency: %v", err))
		}

		debugConsole, _ := usbarmory.DetectDebugAccessory(250 * time.Millisecond)
		<-debugConsole
	}

	// Move DMA region to prevent NonSecure access, alternatively
	// iRAM/OCRAM (default DMA region) can be locked down on its own (as it
	// is outside TZASC control).
	dma.Init(mem.SecureDMAStart, mem.SecureDMASize)
	dcp.DeriveKeyMemory = dma.Default()

	l.Info("trusted os loaded")
}

func main() {

	defer panicHandler()
	tzCtx := tz.NewContext()

	if err := tzCtx.RegisterApp(taELF, info.AppletID); err != nil {
		panic(err)
	}

	l.Info("loaded ta")

	if err := tzCtx.LoadNonsecureWorld(osELF); err != nil {
		panic(err)
	}

	l.Info("running nonsecure world")
	tzCtx.RunNonsecureWorld()

	if !imx6.Native {
		angel.SemihostingShutdown()
	}

	time.Sleep(500 * time.Millisecond)
	imx6.Reset()
}

func panicHandler() {
	if r := recover(); r != nil {
		l.Errorf("panic: %v", r)

		if !imx6.Native {
			angel.SemihostingShutdown()
		}

		time.Sleep(500 * time.Millisecond)
		imx6.Reset()
	}
}
