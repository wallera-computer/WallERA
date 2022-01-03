//go:build !tee_enabled

package main

import (
	"time"

	usbarmory "github.com/f-secure-foundry/tamago/board/f-secure/usbarmory/mark-two"
	"github.com/wallera-computer/wallera/crypto"
)

// this file defines functions needed when the TEE is disabled

func loadDebugAccessory() {
	debugConsole, _ := usbarmory.DetectDebugAccessory(250 * time.Millisecond)
	<-debugConsole
}

func resetBoard() {
	usbarmory.Reset()
}

func tokenImpl() crypto.Token {
	return crypto.NewDumbToken()
}
