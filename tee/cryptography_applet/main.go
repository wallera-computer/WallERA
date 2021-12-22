// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

import (
	"log"
	"os"
	"runtime"
	_ "unsafe"

	"github.com/f-secure-foundry/GoTEE/applet"
	"github.com/f-secure-foundry/GoTEE/syscall"

	"github.com/wallera-computer/wallera/tee/mem"
	"github.com/wallera-computer/wallera/tee/cryptography_applet/info"
	"github.com/wallera-computer/wallera/tee/cryptography_applet/token"
	"github.com/wallera-computer/wallera/tee/trusted_os/tz/client"
)

//go:linkname ramStart runtime.ramStart
var ramStart uint32 = mem.AppletStart

//go:linkname ramSize runtime.ramSize
var ramSize uint32 = mem.AppletSize

//go:linkname ramStackOffset runtime.ramStackOffset
var ramStackOffset uint32 = 0x100

func init() {
	log.SetFlags(log.Ltime)
	log.SetOutput(os.Stdout)
}

func testRNG(n int) {
	buf := make([]byte, n)
	syscall.GetRandom(buf, uint(n))
	log.Printf("PL0 obtained %d random bytes from PL1: %x", n, buf)
}

func testRPC() {
	res := ""
	req := "hello"

	log.Printf("PL0 requests echo via RPC: %s", req)
	err := syscall.Call("RPC.Echo", req, &res)

	if err != nil {
		log.Printf("PL0 received RPC error: %v", err)
	} else {
		log.Printf("PL0 received echo via RPC: %s", res)
	}
}

func main() {
	defer applet.Exit()
	log.Printf("PL0 %s/%s (%s) • TEE user applet (Secure World)", runtime.GOOS, runtime.GOARCH, runtime.Version())

	mail, err := client.SecureRPC{}.RetrieveMail(info.AppletID)
	if err != nil {
		log.Fatal(err)
	}

	t := token.NewToken()

	resp, err := token.Dispatch(mail.Payload, t)
	if err != nil {
		log.Fatal("cannot dispatch:", err)
	}

	mail.Payload = resp

	err = client.SecureRPC{}.WriteResponse(mail)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("written response from trusted applet")
}
