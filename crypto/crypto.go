package crypto

import (
	"encoding/binary"
	"fmt"
)

type Algorithm uint

const (
	AlgoSecp256K1 Algorithm = iota
)

var (
	// I am not ashamed:
	// dGVuZyBlIHNvcmQKdGVuZyBlIHNvcmQKdGVuZyBlIHNvcmQgbyB2ZXIKZmFjaXRtIHN0YSBxdWlldAptIG1hZ24gbWlsbCdldXIgbyBqdW9ybg==
	diversifier = []byte{
		116, 101, 110, 103,
		32, 101, 32, 115,
		111, 114, 100, 10,
		116, 101, 110, 103,
		32, 101, 32, 115,
		111, 114, 100, 10,
		116, 101, 110, 103,
		32, 101, 32, 115,
		111, 114, 100, 32,
		111, 32, 118, 101,
		114, 10, 102, 97,
		99, 105, 116, 109,
		32, 115, 116, 97,
		32, 113, 117, 105,
		101, 116, 10, 109,
		32, 109, 97, 103,
		110, 32, 109, 105,
		108, 108, 39, 101,
		117, 114, 32, 111,
		32, 106, 117, 111,
		114, 110,
	}
)

// Diversifier returns the bytes used to do secure derivation on the Token's memory.
// Secure derivation algorithm is vendor-specific.
func Diversifier() []byte {
	return diversifier
}

// Token is a component which is in charge of executing cryptographic operation involving secrets, key derivation
// and signature execution.
type Token interface {
	RandomBytes(amount uint64) ([]byte, error)
	DeriveSecret() ([32]byte, error)
	Initialize(path DerivationPath) error
	Sign(data []byte, algorithm Algorithm) ([]byte, error)
	PublicKey() ([]byte, error)
	Mnemonic() ([]string, error)
	SupportedSignAlgorithms() []Algorithm
}

type DerivationPath struct {
	Purpose      uint32
	CoinType     uint32
	Account      uint32
	Change       uint32
	AddressIndex uint32
}

func NewDerivationPathFromBytes(purpose, coinType, account, change, addressIndex []byte) DerivationPath {
	return DerivationPath{
		Purpose:      0x80000000 ^ (binary.LittleEndian.Uint32(purpose)),
		CoinType:     0x80000000 ^ (binary.LittleEndian.Uint32(coinType)),
		Account:      0x80000000 ^ (binary.LittleEndian.Uint32(account)),
		Change:       (binary.LittleEndian.Uint32(change)),
		AddressIndex: (binary.LittleEndian.Uint32(addressIndex)),
	}
}

// m / purpose' / coin_type' / account' / change / address_index
func (d DerivationPath) String() string {
	return fmt.Sprintf("m/%v'/%v'/%v'/%v/%v",
		d.Purpose,
		d.CoinType,
		d.Account,
		d.Change,
		d.AddressIndex,
	)
}
