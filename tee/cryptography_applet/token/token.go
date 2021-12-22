package token

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/cosmos/go-bip39"
	"github.com/wallera-computer/wallera/crypto"
)

// Compile-time check which fails if Token doesn't comply with
// crypto.Token interface.
var _ crypto.Token = (*Token)(nil)

var defaultEntropy = []byte{
	118, 252, 209, 103,
	94, 240, 60, 245,
	18, 224, 156, 240,
	11, 232, 52, 25,
	31, 134, 125, 135,
	192, 2, 31, 206,
	216, 100, 159, 234,
	150, 9, 236, 57,
}

type Token struct {
	privKey *hdkeychain.ExtendedKey
}

// NewToken returns a new instance of Token.
// Callers should Clone() this instance and then call Initialize().
func NewToken() crypto.Token {
	return &Token{}
}

func (dt *Token) RandomBytes(amount uint64) ([]byte, error) {
	if amount == 0 {
		return nil, fmt.Errorf("requested bytes amount is zero")
	}

	b := make([]byte, amount)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (dt *Token) DeriveSecret() ([32]byte, error) {
	h := hmac.New(sha256.New, crypto.Diversifier())
	if _, err := h.Write(defaultEntropy); err != nil {
		return [32]byte{}, fmt.Errorf("cannot generate secret, %w", err)
	}

	res := h.Sum(nil)
	if res == nil {
		panic(fmt.Errorf("hmac.SHA256 sum result empty"))
	}

	ret := [32]byte{}
	copy(ret[:], res)

	return ret, nil
}

func (dt *Token) Initialize(path crypto.DerivationPath) error {
	secret, err := dt.DeriveSecret()
	if err != nil {
		return err
	}

	params := chaincfg.MainNetParams
	params.HDCoinType = path.CoinType

	sb, err := hdkeychain.NewMaster(secret[:], &params)
	if err != nil {
		return err
	}

	dt.privKey, err = crypto.KeyFromPath(sb, path)
	if err != nil {
		return err
	}

	return nil
}

func (dt *Token) Sign(data []byte, algorithm crypto.Algorithm) ([]byte, error) {
	pk, err := dt.privKey.ECPrivKey()
	if err != nil {
		return nil, err
	}

	signature, err := pk.Sign(data)
	if err != nil {
		return nil, err
	}

	return signature.Serialize(), nil
}

func (dt *Token) PublicKey() ([]byte, error) {
	epubk, err := dt.privKey.Neuter()
	if err != nil {
		return nil, err
	}

	pp, err := epubk.ECPubKey()
	if err != nil {
		return nil, err
	}

	return pp.SerializeCompressed(), nil
}

func (dt *Token) Mnemonic() ([]string, error) {
	secret, err := dt.DeriveSecret()
	if err != nil {
		return nil, err
	}

	mnemonic, err := bip39.NewMnemonic(secret[:])
	if err != nil {
		return nil, err
	}

	return strings.Split(mnemonic, " "), nil
}

func (dt *Token) SupportedSignAlgorithms() []crypto.Algorithm {
	return []crypto.Algorithm{
		crypto.AlgoSecp256K1,
	}
}

func (dt *Token) Clone() crypto.Token {
	cl := *dt
	return &cl
}
