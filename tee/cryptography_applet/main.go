// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"
	_ "unsafe"

	"github.com/f-secure-foundry/GoTEE/applet"
	_ "github.com/f-secure-foundry/GoTEE/applet"
	"github.com/f-secure-foundry/GoTEE/syscall"
	"github.com/wallera-computer/wallera/log"
	"github.com/wallera-computer/wallera/tee/cryptography_applet/info"
	"github.com/wallera-computer/wallera/tee/cryptography_applet/token"
	"github.com/wallera-computer/wallera/tee/mem"
	"github.com/wallera-computer/wallera/tee/trusted_os/tz/client"
	"go.uber.org/zap"
)

//go:linkname ramStart runtime.ramStart
var ramStart uint32 = mem.AppletStart

//go:linkname ramSize runtime.ramSize
var ramSize uint32 = mem.AppletSize

//go:linkname ramStackOffset runtime.ramStackOffset
var ramStackOffset uint32 = 0x100

var l *zap.SugaredLogger

func init() {
	l = log.Development().Sugar()
}

func testRNG(n int) {
	buf := make([]byte, n)
	syscall.GetRandom(buf, uint(n))
	l.Debugw("PL0 obtained %d random bytes from PL1: %x", n, buf)
}

func main() {
	defer handlePanic()

	l.Infof("PL0 %s/%s (%s) • TEE user applet (Secure World)", runtime.GOOS, runtime.GOARCH, runtime.Version())

	mail, err := client.SecureRPC{}.RetrieveMail(info.AppletID)
	if err != nil {
		panic(err)
	}

	t := token.NewToken()

	resp, err := token.Dispatch(mail.Payload, t)
	if err != nil {
		l.Fatalw("cannot dispatch:", "error", err)
	}

	mail.Payload = resp

	err = client.SecureRPC{}.WriteResponse(mail)
	if err != nil {
		panic(err)
	}

	l.Info("written response from trusted applet")

	applet.Exit()
}

func handlePanic() {
	if r := recover(); r != nil {
		client.ExitWithError(fmt.Errorf("%v", r))
	}
}
