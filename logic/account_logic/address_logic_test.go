package account_logic

import (
	"testing"

	"github.com/dappley/go-dappley/core/account"
	"github.com/stretchr/testify/assert"
)

func TestAddress_IsContract(t *testing.T) {
	tests := []struct {
		name        string
		address     account.Address
		expectedRes bool
		expectedErr error
	}{
		{
			name:        "ContractAddress",
			address:     account.NewAddress("cfSr89kUCpKxxaH8qgqYcnp7BqbMyND9Po"),
			expectedRes: true,
			expectedErr: nil,
		},
		{
			name:        "UserAddress",
			address:     account.NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"),
			expectedRes: false,
			expectedErr: nil,
		},
		{
			name:        "InvalidAddress",
			address:     account.NewAddress("1Xnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"),
			expectedRes: false,
			expectedErr: account.ErrInvalidAddress,
		},
		{
			name:        "EmptyAddress",
			address:     account.NewAddress(""),
			expectedRes: false,
			expectedErr: account.ErrInvalidAddress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := IsContract(tt.address)
			assert.Equal(t, tt.expectedRes, res)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
