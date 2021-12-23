package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime/debug"

	"github.com/f-secure-foundry/GoTEE/syscall"
	"github.com/wallera-computer/wallera/tee/trusted_os/nonsecuresyscall"
	tztypes "github.com/wallera-computer/wallera/tee/trusted_os/tz/types"
)

type rpcCallFunc func(serviceMethod string, args interface{}, reply interface{}) error

func callRPC(callFunc rpcCallFunc, funcName string, arg, dest interface{}) error {
	if err := callFunc(funcName, arg, &dest); err != nil {
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			return err
		}
	}

	return nil
}

type SecureRPC struct{}

func (s SecureRPC) RetrieveMail(appID uint) (tztypes.Mail, error) {
	var mail tztypes.Mail
	return mail, callRPC(
		syscall.Call,
		"SecureRPC.RetrieveMail",
		appID,
		&mail,
	)
}

func (s SecureRPC) WriteResponse(mail tztypes.Mail) error {
	return callRPC(
		syscall.Call,
		"SecureRPC.WriteResponse",
		mail,
		nil,
	)
}

type NonsecureRPC struct{}

func (ns NonsecureRPC) SendMail(mail tztypes.Mail) error {
	return callRPC(
		nonsecuresyscall.Call,
		"NonsecureRPC.SendMail",
		mail,
		nil,
	)
}

func (ns NonsecureRPC) RetrieveResult(appID uint) (tztypes.Mail, error) {
	var mail tztypes.Mail
	return mail, callRPC(
		nonsecuresyscall.Call,
		"NonsecureRPC.RetrieveResult",
		appID,
		&mail,
	)
}

type ClientPanic struct {
	Msg        string
	Stacktrace string
}

func (cp ClientPanic) Error() string {
	return fmt.Sprintf("%s\n\n%s", cp.Msg, cp.Stacktrace)
}

func ExitWithError(err error) {
	trace := debug.Stack()

	p := ClientPanic{
		Msg:        err.Error(),
		Stacktrace: string(trace),
	}

	m, me := json.Marshal(p)
	if me != nil {
		panic(me)
	}

	syscall.Write(666, m, uint(len(m)))
}
