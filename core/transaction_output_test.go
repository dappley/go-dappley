package core

import (
	"testing"

	"github.com/dappley/go-dappley/core/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestTXOutput_Proto(t *testing.T) {
	vout := TXOutput{
		1,
		[]byte("PubKeyHash"),
	}

	pb := vout.ToProto()
	var i interface{} = pb
	_, correct := i.(proto.Message)
	assert.Equal(t, true, correct)
	mpb, err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &corepb.TXOutput{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	vout2 := TXOutput{}
	vout2.FromProto(newpb)

	assert.Equal(t, vout, vout2)
}
