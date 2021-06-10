package account

import (
	"testing"

	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/stretchr/testify/assert"
)

var addressString = "dVaFsQL9He4Xn4CEUh1TCNtfEhHNHKX3hs"

func TestAddress_ToProto(t *testing.T) {
	addr := &Address{address: addressString}
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
	addr := &Address{address: addressString}
	expected := []byte{0x5a, 0xb1, 0x34, 0x4c, 0x17, 0x67, 0x4c, 0x18, 0xd1, 0xa2, 0xdc, 0xea, 0x9f, 0x17, 0x16, 0xe0, 0x49, 0xf4, 0xa0, 0x5e, 0x6c, 0x8d, 0xc6, 0x1e, 0x9a}
	assert.Equal(t, expected, addr.decode())
}

func TestAddress_getAddressCheckSum(t *testing.T) {
	addr := &Address{address: addressString}
	expected := []byte{0x8d, 0xc6, 0x1e, 0x9a}

	assert.Equal(t, expected, addr.getAddressCheckSum())
}