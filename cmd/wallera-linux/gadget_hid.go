package main

// #cgo LDFLAGS: -lusbgx
// #include "gadget-hid.h"
// #include <stdlib.h>
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/wallera-computer/wallera/u2fhid"
)

var ledgerNanoXReport = []byte{
	0x06,
	0xA0,
	0xFF,
	0x09,
	0x01,
	0xA1,
	0x01,
	0x09,
	0x03,
	0x15,
	0x00,
	0x26,
	0xFF,
	0x00,
	0x75,
	0x08,
	0x95,
	0x40,
	0x81,
	0x08,
	0x09,
	0x04,
	0x15,
	0x00,
	0x26,
	0xFF,
	0x00,
	0x75,
	0x08,
	0x95,
	0x40,
	0x91,
	0x08,
	0xC0,
}

func configureHidg(configfsPath string) error {
	reportDescC := (*C.char)(unsafe.Pointer(&ledgerNanoXReport[0]))

	serial := C.CString("0001")
	manufacturer := C.CString("Ledger")
	product := C.CString("Nano X")
	cfp := C.CString(configfsPath)

	defer func() {
		C.free(unsafe.Pointer(serial))
		C.free(unsafe.Pointer(manufacturer))
		C.free(unsafe.Pointer(product))
		C.free(unsafe.Pointer(cfp))
	}()

	res := C.configure_hidg(
		serial,
		manufacturer,
		product,
		cfp,
		reportDescC,
		C.ulong(len(u2fhid.DefaultReport)),
	)

	if res != C.USBG_SUCCESS {
		rres := C.usbg_error(res)
		errName := C.GoString(C.usbg_error_name(rres))
		stdErr := C.GoString(C.usbg_strerror(rres))

		return fmt.Errorf("libusbgx failure, %s: %s", errName, stdErr)
	}

	return nil
}

func cleanupHidg(configfsPath string) error {
	cfp := C.CString(configfsPath)

	defer func() {
		C.free(unsafe.Pointer(cfp))
	}()

	C.cleanup_usbg(cfp)
	return nil

}
