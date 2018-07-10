package core

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/gogo/protobuf/proto"
	"github.com/dappley/go-dappley/core/pb"
)

func TestBlockHeader_Proto(t *testing.T) {
	bh1 := BlockHeader{
		[]byte("hash"),
		[]byte("hash"),
		1,
		2,

	}

	pb := bh1.ToProto()
	mpb,err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &corepb.BlockHeader{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	bh2 := BlockHeader{}
	bh2.FromProto(newpb)

	assert.Equal(t,bh1,bh2)
}

func TestBlock_Proto(t *testing.T) {

	b1 := GenerateMockBlock()

	pb := b1.ToProto()
	mpb,err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &corepb.Block{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	b2 := &Block{}
	b2.FromProto(newpb)

	assert.Equal(t,*b1,*b2)
}