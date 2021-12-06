package crypto

import "fmt"

type DerivationPath struct {
	Purpose      uint32
	CoinType     uint32
	Account      uint32
	Change       uint32
	AddressIndex uint32
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
