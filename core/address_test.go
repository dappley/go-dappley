package core

import (
	"github.com/dappley/go-dappley/core/pb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddress_IsContract(t *testing.T) {
	tests := []struct {
		name        string
		address     Address
		expectedRes bool
		expectedErr error
	}{
		{
			name:        "ContractAddress",
			address:     Address{"cfSr89kUCpKxxaH8qgqYcnp7BqbMyND9Po"},
			expectedRes: true,
			expectedErr: nil,
		},
		{
			name:        "UserAddress",
			address:     Address{"dXnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"},
			expectedRes: false,
			expectedErr: nil,
		},
		{
			name:        "InvalidAddress",
			address:     Address{"1Xnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"},
			expectedRes: false,
			expectedErr: ErrInvalidAddress,
		},
		{
			name:        "EmptyAddress",
			address:     Address{""},
			expectedRes: false,
			expectedErr: ErrInvalidAddress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.address.IsContract()
			assert.Equal(t, tt.expectedRes, res)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestAddress_Proto(t *testing.T) {
	addr := NewAddress("1Xnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB")
	rawBytes, err := proto.Marshal(addr.ToProto())
	assert.Nil(t, err)
	addrProto := &corepb.Address{}
	err = proto.Unmarshal(rawBytes, addrProto)
	assert.Nil(t, err)
	addr1 := Address{}
	addr1.FromProto(addrProto)
	assert.Equal(t, addr, addr1)
}
