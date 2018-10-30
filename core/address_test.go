package core

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddress_IsContract(t *testing.T) {
	tests := []struct {
		name     	string
		address 	Address
		expectedRes bool
		expectedErr error
	}{
		{
			name:     		"ContractAddress",
			address: 		Address{"cfSr89kUCpKxxaH8qgqYcnp7BqbMyND9Po"},
			expectedRes:    true,
			expectedErr: 	nil,
		},
		{
			name:     		"UserAddress",
			address: 		Address{"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"},
			expectedRes:    false,
			expectedErr: 	nil,
		},
		{
			name:     		"InvalidAddress",
			address: 		Address{"1Xnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"},
			expectedRes:    false,
			expectedErr: 	ErrInvalidPubKeyHashVersion,
		},
		{
			name:     		"EmptyAddress",
			address: 		Address{""},
			expectedRes:    false,
			expectedErr: 	ErrInvalidPubKeyHashLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res,err := tt.address.IsContract()
			assert.Equal(t, res, tt.expectedRes)
			assert.Equal(t, err, tt.expectedErr)
		})
	}
}