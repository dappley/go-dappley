package account

import (
	"github.com/btcsuite/btcutil/base58"
	"testing"

	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/stretchr/testify/assert"
)

var addressString = "1Xnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB"

func TestAddress_ToProto(t *testing.T) {
	addr := NewAddress(addressString)
	expected := &accountpb.Address{Address: addressString}
	assert.Equal(t, expected, addr.ToProto())
}

func TestAddress_FromProto(t *testing.T) {
	addr := &Address{}
	addrProto := &accountpb.Address{Address: addressString}
	addr.FromProto(addrProto)

	expected := &Address{address: addressString}
	assert.Equal(t, expected, addr)
}

func TestAddress_decode(t *testing.T) {
	addr := NewAddress(addressString)
	expected := base58.Decode(addressString)
	assert.Equal(t, expected, addr.decode())
}

func TestAddress_getAddressCheckSum(t *testing.T) {
	addr := NewAddress(addressString)
	addrHash := base58.Decode(addressString)
	expected := addrHash[len(addrHash)-addressChecksumLen:]

	assert.Equal(t, expected, addr.getAddressCheckSum())
}