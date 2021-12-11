package cosmos

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/cosmos/btcutil/bech32"
	"github.com/wallera-computer/wallera/crypto"
	"golang.org/x/crypto/ripemd160"
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

//go:generate stringer -type signPayloadDescr
type signPayloadDescr byte

const (
	signInit signPayloadDescr = 0
	signAdd  signPayloadDescr = 1
	signLast signPayloadDescr = 2
)

var (
	commandCodeOK         = [2]byte{0x90, 0x00}
	commandErrEmptyBuffer = [2]byte{0x69, 0x82}
	commandErrWrongLength = [2]byte{0x67, 0x82}
)

// Cosmos handles Cosmos SDK commands.
type Cosmos struct {
	Token                   crypto.Token
	currentSignatureSession *signatureSession
}

// Name implements the apps.App interface
func (c *Cosmos) Name() string {
	return appName
}

// ID implements the apps.App interface
func (c *Cosmos) ID() byte {
	return appID
}

// Commands implements the apps.App interface
func (c *Cosmos) Commands() (commandIDs []byte) {
	ret := []byte{
		byte(claGetVersion),
		byte(claSignSecp256K1),
		byte(claGetAddrSecp256K1),
	}

	return ret
}

// Handle implements the apps.App interface
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

type signatureSession struct {
	derivationPath crypto.DerivationPath
	data           *bytes.Buffer
}

func (c *Cosmos) handleSignSecp256K1(data []byte) (response []byte, code [2]byte, err error) {
	payloadDescription := signPayloadDescr(data[2])
	log.Println("sign payload description:", payloadDescription.String())

	if c.currentSignatureSession == nil && payloadDescription != signInit {
		return nil, commandErrEmptyBuffer, fmt.Errorf("wrong signature description with no session initialized, %v", payloadDescription.String())
	}

	if payloadDescription == signInit {
		c.currentSignatureSession = &signatureSession{
			data: &bytes.Buffer{},
		}
	}

	data = data[5:]
	switch payloadDescription {
	case signInit:
		c.currentSignatureSession.derivationPath = crypto.NewDerivationPathFromBytes(
			data[0:4],
			data[4:8],
			data[8:12],
			data[12:16],
			data[16:20],
		)

		log.Println("read derivation path in sign init:", c.currentSignatureSession.derivationPath.String())
	case signAdd, signLast:
		log.Println("writing data to sigsession:", len(data))
		c.currentSignatureSession.data.Write(data)
	}

	if payloadDescription != signLast {
		log.Println("not continuing with signature since we're not in signLast")
		return nil, commandCodeOK, nil
	}

	defer func(c *Cosmos) {
		c.currentSignatureSession = nil
	}(c)

	// we need to clone the Token instance because the derivation path passed as argument in signInit
	// might differ from the one we used to initialize the token before.
	sessionToken := c.Token.Clone()
	if err := sessionToken.Initialize(c.currentSignatureSession.derivationPath); err != nil {
		return nil, commandErrEmptyBuffer, err
	}

	// len(sigBytes) will be always 10 bytes less than the session data as a whole,
	// because we're trimming the APDU header for signAdd and signLast.
	sigBytes := c.currentSignatureSession.data.Bytes()
	log.Println("complete signature payload:", sigBytes)
	log.Println("sigBytes len:", len(sigBytes))
	log.Println("sigBytes str:", string(sigBytes))

	sbHash := sha256.Sum256(sigBytes)
	resp, err := sessionToken.Sign(sbHash[:], crypto.AlgoSecp256K1)
	if err != nil {
		return nil, commandErrWrongLength, err
	}

	log.Println("length signature:", len(resp))
	return resp, commandCodeOK, nil
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

	return crypto.NewDerivationPathFromBytes(
		base[0:4],
		base[4:8],
		base[8:12],
		base[12:16],
		base[16:20],
	)
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

	dp := derivationPathFromGetAddressRequest(req, data)
	log.Println("derivation path:", dp.String())

	sessionToken := c.Token.Clone()

	if err := sessionToken.Initialize(dp); err != nil {
		return nil, commandErrWrongLength, err
	}

	pubkey, err := sessionToken.PublicKey()
	if err != nil {
		return nil, commandErrWrongLength, err
	}

	address, err := addressFromPubkey(pubkey, hrp)
	if err != nil {
		return nil, commandErrWrongLength, err
	}

	log.Println("generated address:", address)

	// TODO: handle derivation path
	return buildGetAddressResponse(
		pubkey,
		address,
	), commandCodeOK, nil
}

func addressFromPubkey(pubkey []byte, hrp string) (string, error) {
	sha := sha256.Sum256(pubkey)
	s := sha[:]
	r := ripemd160.New()
	_, err := r.Write(s)
	if err != nil {
		return "", err
	}
	pub := r.Sum(nil)

	converted, err := bech32.ConvertBits(pub, 8, 5, true)
	if err != nil {
		return "", err
	}

	addr, err := bech32.Encode(hrp, converted)
	if err != nil {
		return "", err
	}

	return addr, nil
}
