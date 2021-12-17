package main

import (
	"github.com/wallera-computer/wallera/apps"
	"github.com/wallera-computer/wallera/usb"
	"go.uber.org/zap"
)

type hidHandler struct {
	ah *apps.Handler

	outboundChan chan []byte
	session      *usb.Session
	l            *zap.SugaredLogger
}

func newHidHandler(l *zap.SugaredLogger, apps *apps.Handler) *hidHandler {
	return &hidHandler{
		ah:           apps,
		outboundChan: make(chan []byte),
		l:            l,
	}
}

func (hh *hidHandler) Tx(buf []byte, lastErr error) (res []byte, err error) {
	res = <-hh.outboundChan

	return
}

func (hh *hidHandler) Rx(buf []byte, lastErr error) (res []byte, err error) {
	hh.l.Debugw("handling rx", "input bytes", buf, "length", len(buf))

	if hh.session == nil {
		s, err := usb.NewSession(buf, hh.l)
		notErr(err, hh.l)
		hh.session = &s
	} else {
		err := hh.session.ReadData(buf)
		if err != nil {
			hh.l.Errorw("cannot read input data", "error", err)

			for _, chunk := range hh.session.FormatResponse(
				apps.PackageResponse(nil, apps.APDUCommandNotAllowed),
			) {
				hh.outboundChan <- chunk
			}
		}

		hh.l.Debugw("read new data for active session", "should read more", hh.session.ShouldReadMore)

		if hh.session.ShouldReadMore {
			return nil, err
		}
	}

	hh.l.Debugw("handling session", "data", hh.session)

	if hh.session.ShouldReadMore {
		hh.l.Debug("should still read more data, continuing")
		return nil, nil
	}

	resp, err := hh.ah.Handle(hh.session.Data())
	if err != nil {
		hh.l.Errorw("cannot handle session data", "error", err)
	}

	if resp == nil {
		hh.session = nil
		return nil, nil
	}

	chunks := hh.session.FormatResponse(resp)

	if chunks != nil {
		hh.l.Debugw("sending chunks", "chunks", chunks, "channel nil", hh.outboundChan == nil)
		for _, chunk := range chunks {
			hh.outboundChan <- chunk
		}
		hh.l.Debug("chunks sent")
		hh.session = nil
	}

	hh.l.Debug("exiting from rx")
	return nil, nil
}
