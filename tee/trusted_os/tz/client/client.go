package client

import (
	"errors"
	"io"

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

func ExitWithError(err error) {
	b := make([]byte, len(err.Error()))

	copy(b, []byte(err.Error()))
	syscall.Write(666, b, uint(len(b)))
}
