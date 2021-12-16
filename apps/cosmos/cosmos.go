package cosmos

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/cosmos/btcutil/bech32"
	"github.com/wallera-computer/wallera/apps"
	"github.com/wallera-computer/wallera/crypto"
	"github.com/wallera-computer/wallera/log"
	"go.uber.org/zap"

	//lint:ignore SA1019 RIPEMD160 is used for Cosmos addresses derivation
	"golang.org/x/crypto/ripemd160"
)

//go:generate stringer -type command
type command byte

const (
	appName      = "COSMOS"
	appID   byte = 85

	minDataLen = 5

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

// Cosmos handles Cosmos SDK commands.
type Cosmos struct {
	Token                   crypto.Token
	currentSignatureSession *signatureSession

	// TODO: figure out how to better handle logger instance
	l *zap.SugaredLogger
}

func (c *Cosmos) initLog() {
	if c.l != nil {
		return
	}

	c.l = log.Development(
		zap.Fields(zap.String("app_name", c.Name())),
	).Sugar()
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
func (c *Cosmos) Handle(cmd byte, data []byte) (response []byte, code apps.APDUCode, err error) {
	c.initLog()

	if len(data) < minDataLen {
		return nil, apps.APDUWrongLength, fmt.Errorf("data is too small to be processed")
	}

	c.l.Debugw("handling command", "name", command(cmd).String())
	switch cmd {
	case byte(claGetVersion):
		return c.handleGetVersion()
	case byte(claSignSecp256K1):
		return c.handleSignSecp256K1(data)
	case byte(claGetAddrSecp256K1):
		return c.handleGetAddrSecp256K1(data)
	default:
		return nil, apps.APDUINSNotSupported, fmt.Errorf("command not found")
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

func (c *Cosmos) handleGetVersion() (response []byte, code apps.APDUCode, err error) {
	resp, err := getVersionResponse{
		TestMode: 0,
		Version: version{
			Major: 2,
			Minor: 0,
			Patch: 0,
		},
		DeviceLocked: 0,
	}.Marshal()

	return resp, apps.APDUSuccess, err
}

type signatureSession struct {
	derivationPath crypto.DerivationPath
	data           *bytes.Buffer
}

func (c *Cosmos) handleSignSecp256K1(data []byte) (response []byte, code apps.APDUCode, err error) {
	// TODO: check validity of signature payload
	// https://github.com/LedgerHQ/app-cosmos/blob/master/docs/TXSPEC.md
	payloadDescription := signPayloadDescr(data[2])
	c.l.Debugw("sign payload", "description", payloadDescription.String())

	if c.currentSignatureSession == nil && payloadDescription != signInit {
		return nil, apps.APDUExecutionError, fmt.Errorf("wrong signature description with no session initialized, %v", payloadDescription.String())
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

		c.l.Debugw("read derivation path in sign init", "derivation path", c.currentSignatureSession.derivationPath.String())
	case signAdd, signLast:
		c.l.Debugw("writing data to session", "length", len(data))
		c.currentSignatureSession.data.Write(data)
	}

	if payloadDescription != signLast {
		c.l.Debugw("not continuing with signature since we're not in signLast")
		return nil, apps.APDUSuccess, nil
	}

	defer func(c *Cosmos) {
		c.currentSignatureSession = nil
	}(c)

	// we need to clone the Token instance because the derivation path passed as argument in signInit
	// might differ from the one we used to initialize the token before.
	sessionToken := c.Token.Clone()
	if err := sessionToken.Initialize(c.currentSignatureSession.derivationPath); err != nil {
		return nil, apps.APDUExecutionError, err
	}

	// len(sigBytes) will be always 10 bytes less than the session data as a whole,
	// because we're trimming the APDU header for signAdd and signLast.
	sigBytes := c.currentSignatureSession.data.Bytes()
	c.l.Debugw("complete signature payload", "payload", sigBytes, "length", len(sigBytes), "string representation", string(sigBytes))

	if err := json.Unmarshal(sigBytes, &json.RawMessage{}); err != nil {
		return nil, apps.APDUDataInvalid, fmt.Errorf("provided signature data isn't JSON")
	}

	sbHash := sha256.Sum256(sigBytes)
	resp, err := sessionToken.Sign(sbHash[:], crypto.AlgoSecp256K1)
	if err != nil {
		return nil, apps.APDUExecutionError, err
	}

	c.l.Debugw("signature length", "length", len(resp))
	return resp, apps.APDUSuccess, nil
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

func (c *Cosmos) handleGetAddrSecp256K1(data []byte) (response []byte, code apps.APDUCode, err error) {
	req := getAddressRequest{}
	if err := binary.Read(bytes.NewReader(data), binary.BigEndian, &req); err != nil {
		return nil, apps.APDUExecutionError, err // TODO: correct error here
	}

	if err := req.validate(); err != nil {
		return nil, apps.APDUExecutionError, err
	}

	c.l.Debugw("should display on device", "value", displayAddrOnDevice(req))

	hrp := hrpFromGetAddressRequest(req, data)
	c.l.Debugw("request hrp", "value", string(hrp))

	dp := derivationPathFromGetAddressRequest(req, data)
	c.l.Debugw("derivation path", "value", dp.String())

	sessionToken := c.Token.Clone()

	if err := sessionToken.Initialize(dp); err != nil {
		return nil, apps.APDUExecutionError, err
	}

	pubkey, err := sessionToken.PublicKey()
	if err != nil {
		return nil, apps.APDUExecutionError, err
	}

	address, err := addressFromPubkey(pubkey, hrp)
	if err != nil {
		return nil, apps.APDUExecutionError, err
	}

	c.l.Debugw("address generation complete", "address", address)

	return buildGetAddressResponse(
		pubkey,
		address,
	), apps.APDUSuccess, nil
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
