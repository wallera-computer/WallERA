package main

import (
	"fmt"
	"unsafe"

	wallerausb "github.com/wallera-computer/wallera/usb"
)

// #cgo LDFLAGS: -lusbgx
// #include "gadget-hid.h"
// #include <stdlib.h>
import "C"

func configureHidg(configfsPath string) error {
	reportDescC := (*C.char)(unsafe.Pointer(&wallerausb.LedgerNanoXReport[0]))

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
		C.ulong(len(wallerausb.LedgerNanoXReport)),
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
