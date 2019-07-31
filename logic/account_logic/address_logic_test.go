package account_logic

import (
	"testing"

	"github.com/dappley/go-dappley/core/client"
	"github.com/stretchr/testify/assert"
)

func TestAddress_IsContract(t *testing.T) {
	tests := []struct {
		name        string
		address     client.Address
		expectedRes bool
		expectedErr error
	}{
		{
			name:        "ContractAddress",
			address:     client.NewAddress("cfSr89kUCpKxxaH8qgqYcnp7BqbMyND9Po"),
			expectedRes: true,
			expectedErr: nil,
		},
		{
			name:        "UserAddress",
			address:     client.NewAddress("dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"),
			expectedRes: false,
			expectedErr: nil,
		},
		{
			name:        "InvalidAddress",
			address:     client.NewAddress("1Xnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"),
			expectedRes: false,
			expectedErr: client.ErrInvalidAddress,
		},
		{
			name:        "EmptyAddress",
			address:     client.NewAddress(""),
			expectedRes: false,
			expectedErr: client.ErrInvalidAddress,
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
