package cosmos

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
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

	// TODO: remove this
	testAddrBytes = "994bc0e6262f6d129aab9911074d836e60c3a8f8"
)

func testBytes() []byte {
	d, err := hex.DecodeString(testAddrBytes)
	if err != nil {
		panic(err)
	}

	return d
}

var (
	commandCodeOK         = [2]byte{0x90, 0x00}
	commandErrEmptyBuffer = [2]byte{0x69, 0x82}
	commandErrWrongLength = [2]byte{0x67, 0x82}
)

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
		return c.handleGetVersion()
	case byte(claSignSecp256K1):
		return c.handleSignSecp256K1(data)
	case byte(claGetAddrSecp256K1):
		return c.handleGetAddrSecp256K1(data)
	default:
		// TODO: handle this
		return nil, [2]byte{}, fmt.Errorf("command not found")
	}
}

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

func (c *Cosmos) handleGetVersion() (response []byte, code [2]byte, err error) {
	resp, err := getVersionResponse{
		TestMode: 0,
		Version: version{
			Major: 2,
			Minor: 0,
			Patch: 0,
		},
		DeviceLocked: 0,
	}.Marshal()

	return resp, commandCodeOK, err
}

func (c *Cosmos) handleSignSecp256K1(data []byte) (response []byte, code [2]byte, err error) {
	return nil, [2]byte{}, nil
}

type getAddressResponse struct {
	PublicKey [33]byte
	Address   [65]byte
}

func (g getAddressResponse) Marshal() ([]byte, error) {
	ret := &bytes.Buffer{}

	err := binary.Write(ret, binary.BigEndian, g)

	return ret.Bytes(), err
}

type getAddressRequest struct {
	_             [2]byte // space at the beginning, CLA and INS are not interesting here
	P1            uint8
	P2            uint8
	PayloadLength uint8
	HRPLength     uint8
}

func (c *Cosmos) handleGetAddrSecp256K1(data []byte) (response []byte, code [2]byte, err error) {
	req := getAddressRequest{}
	if err := binary.Read(bytes.NewReader(data), binary.BigEndian, &req); err != nil {
		return nil, commandErrEmptyBuffer, err
	}

	if req.PayloadLength == 0 {
		return nil, commandCodeOK, fmt.Errorf("command packet is empty")
	}

	displayAddrOnDevice := false
	if req.P1 == 0x01 {
		displayAddrOnDevice = true
	}

	log.Println("display on device:", displayAddrOnDevice)

	data = data[6:req.PayloadLength]

	log.Println("data:", data)
	hrp := data[0:req.HRPLength]

	log.Println("requested hrp:", string(hrp))

	/*addr, err := bech32.Encode("cosmos", testBytes())
	if err != nil {
		return nil, [2]byte{0x64, 0x00}, err
	}*/

	gar := getAddressResponse{}

	d, _ := hex.DecodeString("02963020258b9fae259da3ba669b29d06a165e319eba845f8857859a140426614e")

	copy(gar.Address[:], []byte("cosmos1n99upe3x9ak39x4tnygswnvrdesv828cnrmm3v"))
	copy(gar.PublicKey[:], d)

	r := &bytes.Buffer{}
	r.Write(d)
	r.WriteString("cosmos1n99upe3x9ak39x4tnygswnvrdesv828cnrmm3v")
	resp, err := gar.Marshal()
	if err != nil {
		return nil, [2]byte{0x64, 0x00}, err
	}

	_ = resp

	// TODO: handle derivation path
	return r.Bytes(), commandCodeOK, nil
}
