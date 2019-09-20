package block

import (
	"github.com/dappley/go-dappley/common/hash"
	blockpb "github.com/dappley/go-dappley/core/block/pb"
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
	return &blockpb.BlockHeader{
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
	bh.hash = pb.(*blockpb.BlockHeader).GetHash()
	bh.prevHash = pb.(*blockpb.BlockHeader).GetPreviousHash()
	bh.nonce = pb.(*blockpb.BlockHeader).GetNonce()
	bh.timestamp = pb.(*blockpb.BlockHeader).GetTimestamp()
	bh.signature = pb.(*blockpb.BlockHeader).GetSignature()
	bh.height = pb.(*blockpb.BlockHeader).GetHeight()
	bh.producer = pb.(*blockpb.BlockHeader).GetProducer()
}
