package cosmos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

//go:generate stringer -type command
type command byte

const (
	appName                     = "COSMOS"
	appID               byte    = 85
	claGetVersion       command = 0x00
	claSignSecp256K1    command = 0x02
	claGetAddrSecp256K1 command = 0x04
)

var commandCodeOK = [2]byte{0x90, 0x00}

type Cosmos struct {
}

func (c *Cosmos) Name() string {
	return appName
}

func (c *Cosmos) ID() byte {
	return appID
}

func (c *Cosmos) Commands() (commandIDs []byte) {
	ret := []byte{
		byte(claGetVersion),
		byte(claSignSecp256K1),
		byte(claGetAddrSecp256K1),
	}

	return ret
}

func (c *Cosmos) Handle(cmd byte, data []byte) (response []byte, code [2]byte, err error) {
	log.Println("handling command", command(cmd).String())
	switch cmd {
	case byte(claGetVersion):
		return c.handleGetVersion(data)
	case byte(claSignSecp256K1):
		return c.handleSignSecp256K1(data)
	case byte(claGetAddrSecp256K1):
		return c.handleGetAddrSecp256K1(data)
	default:
		// TODO: handle this
		return nil, [2]byte{}, fmt.Errorf("command not found")
	}
}

/*
   test_mode: response[0] !== 0,
   version: "" + response[1] + "." + response[2] + "." + response[3],
   device_locked: response[4] === 1,
   major: response[1],
*/

type version struct {
	Major uint8
	Minor uint8
	Patch uint8
}
type getVersionResponse struct {
	TestMode     uint8
	Version      version
	DeviceLocked uint8
}

func (g getVersionResponse) Marshal() ([]byte, error) {
	ret := &bytes.Buffer{}

	err := binary.Write(ret, binary.BigEndian, g)

	return ret.Bytes(), err
}

func (c *Cosmos) handleGetVersion(_ []byte) (response []byte, code [2]byte, err error) {
	resp, err := getVersionResponse{
		TestMode: 1,
		Version: version{
			Major: 42,
			Minor: 42,
			Patch: 42,
		},
		DeviceLocked: 0,
	}.Marshal()

	return resp, commandCodeOK, err
}

func (c *Cosmos) handleSignSecp256K1(data []byte) (response []byte, code [2]byte, err error) {
	return nil, [2]byte{}, nil
}

func (c *Cosmos) handleGetAddrSecp256K1(data []byte) (response []byte, code [2]byte, err error) {
	return nil, [2]byte{}, nil
}
