//go:build tee_enabled

package main

import (
	"github.com/f-secure-foundry/GoTEE/syscall"
	"github.com/f-secure-foundry/tamago/soc/imx6"
	"github.com/wallera-computer/wallera/tee/mem"
	_ "unsafe"
)

const (
	SYS_WRITE = syscall.SYS_WRITE
	SYS_EXIT  = syscall.SYS_EXIT
)

// defined in tee_asm.s
func printSecure(byte)
func exit()

//go:linkname ramStart runtime.ramStart
var ramStart uint32 = mem.NonSecureStart

//go:linkname ramSize runtime.ramSize
var ramSize uint32 = mem.NonSecureSize

//go:linkname hwinit runtime.hwinit
func hwinit() {
	imx6.Init()
	imx6.UART2.Init()
}

//go:linkname printk runtime.printk
func printk(c byte) {

	imx6.UART2.Tx(c)

}

func loadDebugAccessory() {
}

func resetBoard() {
	imx6.Reset()
}
