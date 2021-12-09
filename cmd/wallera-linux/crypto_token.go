package main

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

// Compile-time check which fails if dumbToken doesn't comply with
// crypto.Token interface.
var _ crypto.Token = (*dumbToken)(nil)

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

type dumbToken struct {
	privKey *hdkeychain.ExtendedKey
}

func (dt *dumbToken) RandomBytes(amount uint64) ([]byte, error) {
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

func (dt *dumbToken) DeriveSecret() ([32]byte, error) {
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

func (dt *dumbToken) Initialize(path crypto.DerivationPath) error {
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

	dt.privKey = sb

	return nil
}

func (dt *dumbToken) Sign(data []byte, algorithm crypto.Algorithm) ([]byte, error) {
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

func (dt *dumbToken) PublicKey() ([]byte, error) {
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

func (dt *dumbToken) Mnemonic() ([]string, error) {
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

func (dt *dumbToken) SupportedSignAlgorithms() []crypto.Algorithm {
	return []crypto.Algorithm{
		crypto.AlgoSecp256K1,
	}
}
