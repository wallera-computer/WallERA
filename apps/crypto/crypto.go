package crypto

import (
	"encoding/binary"
	"fmt"
)

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
