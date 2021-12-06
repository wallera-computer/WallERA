package cosmos

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/wallera-computer/sacco.go"
	"github.com/wallera-computer/wallera/apps/crypto"
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

func buildGetAddressResponse(pubkey []byte, address string) []byte {
	r := &bytes.Buffer{}
	r.Write(pubkey)
	r.WriteString(address)
	return r.Bytes()
}

type getAddressRequest struct {
	_             [2]byte // space at the beginning, CLA and INS are not interesting here
	P1            uint8
	P2            uint8
	PayloadLength uint8
	HRPLength     uint8
}

func (g getAddressRequest) validate() error {
	if g.P1 > 1 {
		return fmt.Errorf("first parameter cannot be greater than 1")
	}

	if g.PayloadLength > 255 {
		return fmt.Errorf("total APDU payload length cannot exceed 255 bytes")
	}

	if g.PayloadLength == 0 {
		return fmt.Errorf("no payload specified but should be present")
	}

	if g.HRPLength < 1 || g.HRPLength > 83 {
		return fmt.Errorf("hrp length cannot be less than 1 or exceed 83, found %v", g.HRPLength)
	}

	return nil
}

func hrpFromGetAddressRequest(r getAddressRequest, data []byte) string {
	return string(data[6 : 6+r.HRPLength])
}

func derivationPathFromGetAddressRequest(r getAddressRequest, data []byte) crypto.DerivationPath {
	offset := 6 + r.HRPLength
	base := data[offset : offset+20]

	return crypto.DerivationPath{
		Purpose:      0x80000000 ^ (binary.LittleEndian.Uint32(base[0:4])),
		CoinType:     0x80000000 ^ (binary.LittleEndian.Uint32(base[4:8])),
		Account:      0x80000000 ^ (binary.LittleEndian.Uint32(base[8:12])),
		Change:       (binary.LittleEndian.Uint32(base[12:16])),
		AddressIndex: (binary.LittleEndian.Uint32(base[16:20])),
	}
}

func displayAddrOnDevice(r getAddressRequest) bool {
	return r.P1 == 0x01
}

func (c *Cosmos) handleGetAddrSecp256K1(data []byte) (response []byte, code [2]byte, err error) {
	req := getAddressRequest{}
	if err := binary.Read(bytes.NewReader(data), binary.BigEndian, &req); err != nil {
		return nil, commandErrEmptyBuffer, err // TODO: correct error here
	}

	if err := req.validate(); err != nil {
		return nil, commandErrWrongLength, err
	}

	log.Println("display on device:", displayAddrOnDevice(req))

	hrp := hrpFromGetAddressRequest(req, data)
	log.Println("requested hrp:", string(hrp))

	// TODO: we're generating a random address + pubkey on each call for demo purposes
	// please someone build a better design, thanks!

	/*addr, err := bech32.Encode("cosmos", testBytes())
	if err != nil {
		return nil, [2]byte{0x64, 0x00}, err
	}*/

	dp := derivationPathFromGetAddressRequest(req, data)
	log.Println("derivation path:", dp.String())

	mnm, err := sacco.GenerateMnemonic()
	if err != nil {
		return nil, [2]byte{0x64, 0x00}, err
	}

	wl, err := sacco.FromMnemonic(hrp, mnm, dp.String())
	if err != nil {
		return nil, [2]byte{0x64, 0x00}, err
	}

	pk, err := wl.PublicKeyRaw.ECPubKey()
	if err != nil {
		return nil, [2]byte{0x64, 0x00}, err
	}

	// TODO: handle derivation path
	return buildGetAddressResponse(
		pk.SerializeCompressed(),
		wl.Address,
	), commandCodeOK, nil
}
