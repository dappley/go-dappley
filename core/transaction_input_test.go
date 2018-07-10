package core

import (
	"testing"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core/pb"
)

func TestTXInput_Proto(t *testing.T) {
	vin := TXInput{
		[]byte("txid"),
		1,
		[]byte("signature"),
		[]byte("PubKey"),
	}

	pb := vin.ToProto()
	mpb,err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &corepb.TXInput{}
	err = proto.Unmarshal(mpb,newpb)
	assert.Nil(t, err)

	vin2 := TXInput{}
	vin2.FromProto(newpb)

	assert.Equal(t,vin,vin2)
}
