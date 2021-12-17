//go:build !debug

package main

import (
	"github.com/wallera-computer/wallera/log"
	"go.uber.org/zap"
)

func init() {

}

func logger() *zap.SugaredLogger {
	return log.Production().Sugar()
}
