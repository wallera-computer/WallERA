package crypto

import (
	"crypto/sha256"

	"github.com/wallera-computer/wallera/crypto"
	"github.com/wallera-computer/wallera/tee/cryptography_applet/info"
	teetoken "github.com/wallera-computer/wallera/tee/cryptography_applet/token"
	"github.com/wallera-computer/wallera/tee/trusted_os/tz/client"
	tztypes "github.com/wallera-computer/wallera/tee/trusted_os/tz/types"
)

// Compile-time check which fails if TEEToken doesn't comply with
// crypto.Token interface.
var _ crypto.Token = (*TEEToken)(nil)

type TEEToken struct {
	path crypto.DerivationPath
}

func (tt *TEEToken) RandomBytes(amount uint64) ([]byte, error) {
	req := teetoken.RandomBytesRequest{
		Request: teetoken.Request{
			ID: teetoken.RequestRandomBytes,
		},
		Amount: amount,
	}

	resp := teetoken.RandomBytesResponse{}

	if err := doRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (tt *TEEToken) DeriveSecret() ([32]byte, error) {
	return [32]byte{}, nil // TODO: implement secret derivation (do we need it?)
}

func (tt *TEEToken) Initialize(path crypto.DerivationPath) error {
	tt.path = path
	return nil
}

func (tt *TEEToken) Sign(data []byte, algorithm crypto.Algorithm) ([]byte, error) {
	hs := sha256.Sum256(data)

	req := teetoken.SignRequest{
		Request: teetoken.Request{
			ID: teetoken.RequestSign,
		},
		Data:           hs[:],
		DerivationPath: tt.path,
		Algorithm:      crypto.AlgoSecp256K1,
	}

	resp := teetoken.SignResponse{}

	if err := doRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (tt *TEEToken) PublicKey() ([]byte, error) {
	req := teetoken.PublicKeyRequest{
		Request: teetoken.Request{
			ID: teetoken.RequestPublicKey,
		},
		DerivationPath: tt.path,
	}

	resp := teetoken.PublicKeyResponse{}

	if err := doRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (tt *TEEToken) Mnemonic() ([]string, error) {
	req := teetoken.MnemonicRequest{
		Request: teetoken.Request{
			ID: teetoken.RequestMnemonic,
		},
		DerivationPath: tt.path,
	}

	resp := teetoken.MnemonicResponse{}

	if err := doRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp.Words, nil
}

func (tt *TEEToken) SupportedSignAlgorithms() []crypto.Algorithm {
	return []crypto.Algorithm{
		crypto.AlgoSecp256K1,
	}
}

func (tt *TEEToken) Clone() crypto.Token {
	cl := *tt
	return &cl
}

func doRequest(input interface{}, output interface{}) error {
	reqBytes, err := teetoken.PackageRequest(input)
	if err != nil {
		return err
	}

	m := tztypes.Mail{
		AppID:   info.AppletID,
		Payload: reqBytes,
	}

	err = client.NonsecureRPC{}.SendMail(m)
	if err != nil {
		return err
	}

	res, err := client.NonsecureRPC{}.RetrieveResult(m.AppID)
	if err != nil {
		return err
	}

	if err := teetoken.UnpackResponse(res.Payload, output); err != nil {
		return err
	}

	return nil
}
