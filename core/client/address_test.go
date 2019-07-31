package client

import (
	"testing"

	accountpb "github.com/dappley/go-dappley/core/client/pb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestAddress_Proto(t *testing.T) {
	addr := NewAddress("1Xnq2R6SzRNUt7ZANAqyZc2P9ziF6vYekB")
	rawBytes, err := proto.Marshal(addr.ToProto())
	assert.Nil(t, err)
	addrProto := &accountpb.Address{}
	err = proto.Unmarshal(rawBytes, addrProto)
	assert.Nil(t, err)
	addr1 := Address{}
	addr1.FromProto(addrProto)
	assert.Equal(t, addr, addr1)
}
