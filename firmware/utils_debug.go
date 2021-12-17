//go:build debug

package main

import (
	"runtime"
	"time"

	usbarmory "github.com/f-secure-foundry/tamago/board/f-secure/usbarmory/mark-two"
	"github.com/f-secure-foundry/tamago/soc/imx6"
	"github.com/wallera-computer/wallera/log"
	"go.uber.org/zap"
)

func init() {
	go rebootWatcher()

}

func rebootWatcher() {
	buf := make([]byte, 1)

	l := logger()

	for {
		runtime.Gosched()
		imx6.UART2.Read(buf)
		if buf[0] == 0 {
			continue
		}

		if buf[0] == 'r' {
			l.Info("rebooting...")
			time.Sleep(500 * time.Millisecond)
			usbarmory.Reset()
		}

		buf[0] = 0
	}
}

func logger() *zap.SugaredLogger {
	return log.Development().Sugar()
}
