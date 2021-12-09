package main

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wallera-computer/wallera/crypto"
)

const (
	standardSecret = "10cbc65c342e07b730ffb0f48fe2c41e9776363c73ade5e699c4cccb1257188d"
	standardPubkey = "03c04b40d19f65ff09f60beba0f38b711ef21e12e18b88fadced3fb285d1431bea"
)

func secretBytes(t *testing.T) [32]byte {
	t.Helper()
	h, err := hex.DecodeString(standardSecret)
	require.NoError(t, err)
	ret := [32]byte{}
	copy(ret[:], h)
	return ret
}

func pubKeyBytes(t *testing.T) []byte {
	t.Helper()
	h, err := hex.DecodeString(standardPubkey)
	require.NoError(t, err)
	return h
}

func Test_dumbToken_DeriveSecretReturnsExpectedSecret(t *testing.T) {
	dt := &dumbToken{}
	s, err := dt.DeriveSecret()
	require.NoError(t, err)
	require.NotNil(t, s)
	require.Equal(t, secretBytes(t), s)
}

func Test_dumbToken_DeriveSecretIsIdempotent(t *testing.T) {
	dt := &dumbToken{}
	s1, err := dt.DeriveSecret()
	require.NoError(t, err)
	require.NotNil(t, s1)

	s2, err := dt.DeriveSecret()
	require.NoError(t, err)
	require.NotNil(t, s2)

	require.Equal(t, s1, s2)
}

func Test_dumbToken_PublicKey(t *testing.T) {
	dt := &dumbToken{}
	require.NoError(t, dt.Initialize(crypto.DerivationPath{
		Purpose:      44,
		CoinType:     118,
		Account:      0,
		Change:       0,
		AddressIndex: 0,
	}))

	pk, err := dt.PublicKey()

	require.NoError(t, err)
	require.NotNil(t, pk)
	require.Equal(t, pubKeyBytes(t), pk)
}

func Test_dumbToken_RandomBytes(t *testing.T) {
	tests := []struct {
		name              string
		amount            uint64
		expectedResultLen int
		errAssertion      require.ErrorAssertionFunc
		dataAssertion     require.ValueAssertionFunc
		lenAssertion      require.ComparisonAssertionFunc
	}{
		{
			"non-zero bytes request gets fulfilled",
			42,
			42,
			require.NoError,
			require.NotEmpty,
			require.Equal,
		},
		{
			"zero bytes request doesn't fulfilled",
			0,
			0,
			require.Error,
			require.Empty,
			require.Equal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := &dumbToken{}
			got, err := dt.RandomBytes(tt.amount)
			tt.errAssertion(t, err)
			tt.dataAssertion(t, got)
			tt.lenAssertion(t, len(got), tt.expectedResultLen)
		})
	}
}
