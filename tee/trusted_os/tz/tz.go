// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package tz

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/f-secure-foundry/GoTEE/monitor"
	"github.com/f-secure-foundry/GoTEE/syscall"
	"github.com/f-secure-foundry/armory-boot/exec"
	"github.com/f-secure-foundry/tamago/arm"
	"github.com/f-secure-foundry/tamago/soc/imx6"
	"github.com/f-secure-foundry/tamago/soc/imx6/csu"
	"github.com/f-secure-foundry/tamago/soc/imx6/tzasc"

	"github.com/wallera-computer/wallera/tee/mem"
	"github.com/wallera-computer/wallera/tee/trusted_os/tz/types"
)

var ErrAppNotFound = errors.New("app not found")
var ErrNoMail = errors.New("no mail")
var ErrNoResult = errors.New("no result")
var ErrMailboxFull = errors.New("mailbox full")
var ErrResultBoxFull = errors.New("result box full")
var ErrTAExit = errors.New("ta exit")
var ErrNonsecureExit = errors.New("nonsecure exit")

type SecureRPC struct {
	ctx *Context
}

func (srpc *SecureRPC) RetrieveMail(appID uint, out *types.Mail) error {
	mail, err := srpc.ctx.RetrieveMail(appID)
	if err != nil {
		return err
	}

	out.CopyFrom(mail)

	return nil
}

func (srpc *SecureRPC) WriteResponse(response types.Mail, _ *struct{}) error {
	err := srpc.ctx.WriteResponse(response)
	if err != nil {
		return err
	}

	return nil
}

type NonsecureRPC struct {
	ctx *Context
}

func (nsrpc *NonsecureRPC) SendMail(mail types.Mail, res *[]byte) error {
	return nsrpc.ctx.SendMail(mail)
}

func (nsrpc *NonsecureRPC) RetrieveResult(appID uint, out *types.Mail) error {
	mail, err := nsrpc.ctx.ReadResponse(appID)
	if err != nil {
		return err
	}

	out.CopyFrom(mail)

	return nil
}

type Context struct {
	Apps           map[uint]*exec.ELFImage
	NonsecureWorld *monitor.ExecCtx

	mailbox   sync.Map
	resultBox sync.Map
}

func NewContext() *Context {
	return &Context{
		Apps:           map[uint]*exec.ELFImage{},
		NonsecureWorld: &monitor.ExecCtx{},
		mailbox:        sync.Map{},
	}
}

func (c *Context) RetrieveMail(appID uint) (types.Mail, error) {
	mail, found := c.mailbox.LoadAndDelete(appID)
	if !found {
		return types.Mail{}, ErrNoMail
	}

	mailb, ok := mail.([]byte)
	if !ok {
		return types.Mail{}, fmt.Errorf("could not read mailbox content as byte slice")
	}

	out := types.Mail{}
	out.AppID = appID
	out.Payload = mailb

	return out, nil
}

func (c *Context) WriteResponse(result types.Mail) error {
	_, found := c.resultBox.LoadAndDelete(result.AppID)
	if found {
		return ErrResultBoxFull
	}

	c.resultBox.Store(result.AppID, result.Payload)
	return nil
}

func (c *Context) ReadResponse(appID uint) (types.Mail, error) {
	mail, found := c.resultBox.LoadAndDelete(appID)
	if !found {
		return types.Mail{}, ErrNoResult
	}

	mailb, ok := mail.([]byte)
	if !ok {
		return types.Mail{}, fmt.Errorf("could not read mailbox content as byte slice")
	}

	out := types.Mail{}
	out.AppID = appID
	out.Payload = mailb

	return out, nil
}

func (c *Context) SendMail(mail types.Mail) error {
	_, found := c.mailbox.Load(mail.AppID)
	if found {
		return fmt.Errorf("cannot deliver mail for app %v, %w", mail.AppID, ErrMailboxFull)
	}

	c.mailbox.Store(mail.AppID, mail.Payload)
	return nil
}

func (c *Context) RegisterApp(appContent []byte, appID uint) error {
	image := &exec.ELFImage{
		Region: mem.AppletRegion,
		ELF:    appContent,
	}

	if err := image.Load(); err != nil {
		return err
	}

	c.Apps[appID] = image

	return nil
}

func (c *Context) loadTA(appID uint) (*monitor.ExecCtx, error) {
	image, found := c.Apps[appID]
	if !found {
		return nil, fmt.Errorf("cannot load app %v, %w", appID, ErrAppNotFound)
	}

	if err := image.Load(); err != nil {
		return nil, err
	}

	ta, err := monitor.Load(image.Entry(), image.Region, true)
	if err != nil {
		return nil, fmt.Errorf("cannot load app, %w", err)
	}

	// register example RPC receiver
	ta.Server.Register(&SecureRPC{ctx: c})

	// set stack pointer to the end of applet memory
	ta.R13 = mem.AppletStart + mem.AppletSize

	// override default handler to improve logging
	ta.Handler = logHandler
	ta.Debug = true

	return ta, nil
}

func (c *Context) LoadNonsecureWorld(appContent []byte) error {
	image := &exec.ELFImage{
		Region: mem.NonSecureRegion,
		ELF:    appContent,
	}

	if err := image.Load(); err != nil {
		return err
	}

	os, err := monitor.Load(image.Entry(), image.Region, false)
	if err != nil {
		return fmt.Errorf("cannot load nonsecure world, %w", err)
	}

	log.Printf("PL1 loaded kernel addr:%#x size:%d entry:%#x", os.Memory.Start, len(appContent), os.R15)

	lock := true
	if !imx6.Native {
		lock = false
	}

	if err = configureTrustZone(lock); err != nil {
		return fmt.Errorf("could not configure trustzone, %w", err)
	}

	os.Server.Register(&NonsecureRPC{ctx: c})

	// override default handler to improve logging
	os.Handler = c.nonsecureWorldHandler
	os.Debug = true

	c.NonsecureWorld = os

	return nil
}

func (c *Context) RunNonsecureWorld() {
	run(c.NonsecureWorld)
}

func (c *Context) dispatchAppsCalls() {
	c.mailbox.Range(func(keyRaw, _ interface{}) bool {
		key, ok := keyRaw.(uint)
		if !ok {
			panic("somehow a mailbox key isn't uint")
		}

		ta, err := c.loadTA(key)
		if err != nil {
			panic(fmt.Errorf("cannot load ta %v, %w", key, err))
		}

		run(ta)

		return true
	})
}

func (c *Context) nonsecureWorldHandler(ctx *monitor.ExecCtx) error {
	if !(ctx.R0 == syscall.SYS_RPC_REQ || ctx.R0 == syscall.SYS_RPC_RES) {
		return logHandler(ctx)
	}

	if err := monitor.SecureHandler(ctx); err != nil {
		return err
	}

	c.dispatchAppsCalls()

	return nil
}

func configureTrustZone(lock bool) (err error) {
	// grant NonSecure access to CP10 and CP11
	imx6.ARM.NonSecureAccessControl(1<<11 | 1<<10)

	if !imx6.Native {
		return
	}

	csu.Init()

	// grant NonSecure access to all peripherals
	for i := csu.CSL_MIN; i < csu.CSL_MAX; i++ {
		if err = csu.SetSecurityLevel(i, 0, csu.SEC_LEVEL_0, false); err != nil {
			return
		}

		if err = csu.SetSecurityLevel(i, 1, csu.SEC_LEVEL_0, false); err != nil {
			return
		}
	}

	// set default TZASC region (entire memory space) to NonSecure access
	if err = tzasc.EnableRegion(0, 0, 0, (1<<tzasc.SP_NW_RD)|(1<<tzasc.SP_NW_WR)); err != nil {
		return
	}

	if lock {
		// restrict Secure World memory
		if err = tzasc.EnableRegion(1, mem.SecureStart, mem.SecureSize+mem.SecureDMASize, (1<<tzasc.SP_SW_RD)|(1<<tzasc.SP_SW_WR)); err != nil {
			return
		}

		// restrict Secure World applet region
		if err = tzasc.EnableRegion(2, mem.AppletStart, mem.AppletSize, (1<<tzasc.SP_SW_RD)|(1<<tzasc.SP_SW_WR)); err != nil {
			return
		}
	} else {
		return
	}

	// set all controllers to NonSecure
	for i := csu.SA_MIN; i < csu.SA_MAX; i++ {
		if err = csu.SetAccess(i, false, false); err != nil {
			return
		}
	}

	// restrict access to GPIO4 (used by LEDs)
	if err = csu.SetSecurityLevel(2, 1, csu.SEC_LEVEL_4, false); err != nil {
		return
	}

	// restrict access to IOMUXC (used by LEDs)
	if err = csu.SetSecurityLevel(6, 1, csu.SEC_LEVEL_4, false); err != nil {
		return
	}

	// restrict access to USB
	if err = csu.SetSecurityLevel(8, 0, csu.SEC_LEVEL_4, false); err != nil {
		return
	}

	// set USB controller as Secure
	if err = csu.SetAccess(4, true, false); err != nil {
		return
	}

	// restrict access to ROMCP
	if err = csu.SetSecurityLevel(13, 0, csu.SEC_LEVEL_4, false); err != nil {
		return
	}

	// restrict access to TZASC
	if err = csu.SetSecurityLevel(16, 1, csu.SEC_LEVEL_4, false); err != nil {
		return
	}

	// restrict access to DCP
	if err = csu.SetSecurityLevel(34, 0, csu.SEC_LEVEL_4, false); err != nil {
		return
	}

	// set DCP as Secure
	if err = csu.SetAccess(14, true, false); err != nil {
		return
	}

	return
}

func grantPeripheralAccess() (err error) {
	// allow access to GPIO4 (used by LEDs)
	if err = csu.SetSecurityLevel(2, 1, csu.SEC_LEVEL_0, false); err != nil {
		return
	}

	// allow access to IOMUXC (used by LEDs)
	if err = csu.SetSecurityLevel(6, 1, csu.SEC_LEVEL_0, false); err != nil {
		return
	}

	// allow access to USB
	if err = csu.SetSecurityLevel(8, 0, csu.SEC_LEVEL_0, false); err != nil {
		return
	}

	// set USB controller as NonSecure
	if err = csu.SetAccess(4, false, false); err != nil {
		return
	}

	// set USDHC1 (microSD) controller as NonSecure
	if err = csu.SetAccess(10, false, false); err != nil {
		return
	}

	// set USDHC2 (eMMC) controller as NonSecure
	if err = csu.SetAccess(11, false, false); err != nil {
		return
	}

	return
}

func run(ctx *monitor.ExecCtx) {
	mode := arm.ModeName(int(ctx.SPSR) & 0x1f)
	ns := ctx.NonSecure()

	log.Printf("PL1 starting mode:%s ns:%v sp:%#.8x pc:%#.8x", mode, ns, ctx.R13, ctx.R15)

	err := ctx.Run()

	log.Printf("PL1 stopped mode:%s ns:%v sp:%#.8x lr:%#.8x pc:%#.8x err:%v", mode, ns, ctx.R13, ctx.R14, ctx.R15, err)
	errTemplate := ErrTAExit
	if ctx.NonSecure() {
		errTemplate = ErrNonsecureExit
	}

	if err != nil && !errors.Is(err, errTemplate) {
		panic(err)
	}
}

// logHandler allows to override the GoTEE default handler and avoid
// interleaved logs, as the supervisor and applet contexts are logging
// simultaneously.
func logHandler(ctx *monitor.ExecCtx) (err error) {
	defaultHandler := monitor.SecureHandler

	switch {
	case ctx.R0 == syscall.SYS_WRITE:
		bufferedStdoutLog(byte(ctx.R1), ctx.NonSecure())
	case ctx.NonSecure() && ctx.R0 == syscall.SYS_EXIT:
		if ctx.Debug {
			ctx.Print()
		}

		return ErrNonsecureExit
	case ctx.R0 == syscall.SYS_EXIT:
		return ErrTAExit

	// TODO: clean this
	case ctx.R0 == 666: // syscall.SYS_EXIT_ERROR:
		off := int(ctx.R1 - ctx.Memory.Start)
		buf := make([]byte, ctx.R2)

		if !(off >= 0 && off < (ctx.Memory.Size-len(buf))) {
			return errors.New("invalid read offset")
		}

		if n, err := rand.Read(buf); err != nil || n != int(ctx.R2) {
			return errors.New("internal error")
		}

		ctx.Memory.Read(ctx.Memory.Start, off, buf)

		return fmt.Errorf(string(buf))
	default:
		err = defaultHandler(ctx)
	}

	return
}

func logHandlerCopy(ctx *monitor.ExecCtx) (err error) {
	defaultHandler := monitor.SecureHandler

	if ctx.NonSecure() {
		defaultHandler = monitor.NonSecureHandler
	}

	switch {
	case ctx.R0 == syscall.SYS_WRITE:
		bufferedStdoutLog(byte(ctx.R1), ctx.NonSecure())
	case ctx.NonSecure() && ctx.R0 == syscall.SYS_EXIT:
		if ctx.Debug {
			ctx.Print()
		}

		return errors.New("exit")
	default:
		err = defaultHandler(ctx)
	}

	return
}
