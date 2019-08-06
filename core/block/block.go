package block

import (
	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
	"time"
)

type Block struct {
	header       *BlockHeader
	transactions []*core.Transaction
}

func NewBlock(txs []*core.Transaction, parent *Block, producer string) *Block {
	return NewBlockWithTimestamp(txs, parent, time.Now().Unix(), producer)
}

func NewBlockFromHeader(blkHeader *BlockHeader, txs []*core.Transaction) *Block {
	return &Block{blkHeader, txs}
}

func NewBlockWithTimestamp(txs []*core.Transaction, parent *Block, timeStamp int64, producer string) *Block {

	var prevHash []byte
	var height uint64
	height = 1
	if parent != nil {
		prevHash = parent.GetHash()
		height = parent.GetHeight() + 1
	}

	if txs == nil {
		txs = []*core.Transaction{}
	}
	return &Block{
		header: &BlockHeader{
			hash:      []byte{},
			prevHash:  prevHash,
			nonce:     0,
			timestamp: timeStamp,
			signature: nil,
			height:    height,
			producer:  producer,
		},
		transactions: txs,
	}
}

func (b *Block) GetHeader() *BlockHeader {
	return b.header
}

func (b *Block) GetHash() hash.Hash {
	return b.header.hash
}

func (b *Block) GetSign() hash.Hash {
	return b.header.signature
}

func (b *Block) GetHeight() uint64 {
	return b.header.height
}

func (b *Block) GetPrevHash() hash.Hash {
	return b.header.prevHash
}

func (b *Block) GetNonce() int64 {
	return b.header.nonce
}

func (b *Block) GetTimestamp() int64 {
	return b.header.timestamp
}

func (b *Block) GetProducer() string {
	return b.header.producer
}

func (b *Block) GetTransactions() []*core.Transaction {
	return b.transactions
}

func (b *Block) SetHash(hash hash.Hash) {
	b.header.hash = hash
}

func (b *Block) SetNonce(nonce int64) {
	b.header.nonce = nonce
}

func (b *Block) SetSignature(sig hash.Hash) {
	b.header.signature = sig
}

func (b *Block) IsSigned() bool {
	return b.header != nil && b.header.signature != nil
}

func (b *Block) ToProto() proto.Message {

	var txArray []*corepb.Transaction
	for _, tx := range b.transactions {
		txArray = append(txArray, tx.ToProto().(*corepb.Transaction))
	}

	return &corepb.Block{
		Header:       b.header.ToProto().(*corepb.BlockHeader),
		Transactions: txArray,
	}
}

func (b *Block) FromProto(pb proto.Message) {

	bh := BlockHeader{}
	bh.FromProto(pb.(*corepb.Block).GetHeader())
	b.header = &bh

	var txs []*core.Transaction

	for _, txpb := range pb.(*corepb.Block).GetTransactions() {
		tx := &core.Transaction{}
		tx.FromProto(txpb)
		txs = append(txs, tx)
	}
	b.transactions = txs
}

func (b *Block) Serialize() []byte {
	rawBytes, err := proto.Marshal(b.ToProto())
	if err != nil {
		logger.WithError(err).Panic("Block: Cannot serialize block!")
	}
	logger.WithFields(logger.Fields{
		"size": len(rawBytes),
	}).Info("Block: Serialize Block!")
	return rawBytes
}

func Deserialize(d []byte) *Block {
	pb := &corepb.Block{}
	err := proto.Unmarshal(d, pb)
	if err != nil {
		logger.WithError(err).Panic("Block: Cannot deserialize block!")
	}
	block := &Block{}
	block.FromProto(pb)
	return block
}
