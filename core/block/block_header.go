package block

import (
	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/golang/protobuf/proto"
)

type BlockHeader struct {
	hash      hash.Hash
	prevHash  hash.Hash
	nonce     int64
	timestamp int64
	signature hash.Hash
	height    uint64
	producer  string
}

func NewBlockHeader(hash hash.Hash, prevHash hash.Hash, nonce int64, timeStamp int64, height uint64) *BlockHeader {
	return &BlockHeader{
		hash:      hash,
		prevHash:  prevHash,
		nonce:     nonce,
		timestamp: timeStamp,
		height:    height,
	}
}

func (bh *BlockHeader) ToProto() proto.Message {
	return &corepb.BlockHeader{
		Hash:         bh.hash,
		PreviousHash: bh.prevHash,
		Nonce:        bh.nonce,
		Timestamp:    bh.timestamp,
		Signature:    bh.signature,
		Height:       bh.height,
		Producer:     bh.producer,
	}
}

func (bh *BlockHeader) FromProto(pb proto.Message) {
	if pb == nil {
		return
	}
	bh.hash = pb.(*corepb.BlockHeader).GetHash()
	bh.prevHash = pb.(*corepb.BlockHeader).GetPreviousHash()
	bh.nonce = pb.(*corepb.BlockHeader).GetNonce()
	bh.timestamp = pb.(*corepb.BlockHeader).GetTimestamp()
	bh.signature = pb.(*corepb.BlockHeader).GetSignature()
	bh.height = pb.(*corepb.BlockHeader).GetHeight()
	bh.producer = pb.(*corepb.BlockHeader).GetProducer()
}
