package core

import (
	"testing"

	"github.com/dappley/go-dappley/core/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestTXInput_Proto(t *testing.T) {
	vin := TXInput{
		[]byte("txid"),
		1,
		[]byte("signature"),
		[]byte("PubKey"),
	}

	pb := vin.ToProto()
	var i interface{} = pb
	_, correct := i.(proto.Message)
	assert.Equal(t, true, correct)
	mpb, err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &corepb.TXInput{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	vin2 := TXInput{}
	vin2.FromProto(newpb)

	assert.Equal(t, vin, vin2)
}
