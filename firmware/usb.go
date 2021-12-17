package main

import (
	"github.com/f-secure-foundry/tamago/soc/imx6/usb"
	"github.com/wallera-computer/wallera"
)

func baseConfiguration(device *usb.Device) error {
	// Supported Language Code Zero: English
	device.SetLanguageCodes([]uint16{0x0409})

	// device descriptor
	device.Descriptor = &usb.DeviceDescriptor{}
	device.Descriptor.SetDefaults()

	// HID devices sets those in the Interface descriptor.
	device.Descriptor.DeviceClass = 0x0
	device.Descriptor.DeviceSubClass = 0x0
	device.Descriptor.DeviceProtocol = 0x0

	// Ledger Nano X USB IDs
	device.Descriptor.VendorId = 0x2c97
	device.Descriptor.ProductId = 0x4011

	device.Descriptor.Device = 0x0001

	iManufacturer, err := device.AddString(`Ledger`)
	if err != nil {
		return err
	}

	device.Descriptor.Manufacturer = iManufacturer

	iProduct, err := device.AddString(`Nano X`)
	if err != nil {
		return err
	}

	device.Descriptor.Product = iProduct

	iSerial, err := device.AddString(`0001`)
	if err != nil {
		return err
	}

	device.Descriptor.SerialNumber = iSerial

	return nil
}

func startUSB(handler wallera.HIDHandler) error {
	device := &usb.Device{}

	cd := usb.ConfigurationDescriptor{}
	cd.SetDefaults()
	cd.Attributes = 160

	baseConfiguration(device)

	err := device.AddConfiguration(&cd)
	if err != nil {
		return err
	}

	if err := wallera.ConfigureUSB(&cd, device, handler); err != nil {
		return err
	}

	usb.USB1.Init()
	usb.USB1.DeviceMode()
	usb.USB1.Reset()

	// never returns
	usb.USB1.Start(device)

	return nil
}
