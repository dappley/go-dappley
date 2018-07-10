package core

import (
	"testing"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/core/pb"
)

func TestTXOutput_Proto(t *testing.T) {
	vout := TXOutput{
		1,
		[]byte("PubKeyHash"),
	}

	pb := vout.ToProto()
	mpb,err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &corepb.TXOutput{}
	err = proto.Unmarshal(mpb,newpb)
	assert.Nil(t, err)

	vout2 := TXOutput{}
	vout2.FromProto(newpb)

	assert.Equal(t,vout,vout2)
}
