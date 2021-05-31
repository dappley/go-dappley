package account

import (
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
	expected := []byte{0x0, 0x1, 0x51, 0xbe, 0x94, 0x9e, 0xbd, 0x47, 0xf8, 0x7e, 0x77, 0x4a, 0x12, 0x36, 0x28, 0x6d, 0x7, 0x49, 0xe1, 0x77, 0x56, 0x35, 0x3b, 0x48, 0x9a, 0x44}
	assert.Equal(t, expected, addr.decode())
}

func TestAddress_getAddressCheckSum(t *testing.T) {
	addr := NewAddress(addressString)
	expected := []byte{0x3b, 0x48, 0x9a, 0x44}

	assert.Equal(t, expected, addr.getAddressCheckSum())
}