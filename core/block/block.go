package block

import (
	"time"

	"github.com/dappley/go-dappley/common/hash"
	blockpb "github.com/dappley/go-dappley/core/block/pb"
	"github.com/dappley/go-dappley/core/transaction"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

type Block struct {
	header       *BlockHeader
	transactions []*transaction.Transaction
}

func NewBlock(txs []*transaction.Transaction, parent *Block, producer string) *Block {
	return NewBlockWithTimestamp(txs, parent, time.Now().Unix(), producer)
}

func NewBlockWithRawInfo(hash hash.Hash, prevHash hash.Hash, nonce int64, timeStamp int64, height uint64, txs []*transaction.Transaction) *Block {
	return &Block{
		NewBlockHeader(
			hash,
			prevHash,
			nonce,
			timeStamp,
			height),
		txs,
	}
}

func NewBlockWithTimestamp(txs []*transaction.Transaction, parent *Block, timeStamp int64, producer string) *Block {

	var prevHash []byte
	var height uint64
	height = 1
	if parent != nil {
		prevHash = parent.GetHash()
		height = parent.GetHeight() + 1
	}

	if txs == nil {
		txs = []*transaction.Transaction{}
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

func NewBlockByHash(preHash hash.Hash, producer string) *Block  {
	return &Block{
		header: &BlockHeader{
			[]byte{},
			preHash,
			0,
			0,
			[]byte{},
			0,
			producer,
		},
		transactions: []*transaction.Transaction{},
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

func (b *Block) GetTransactions() []*transaction.Transaction {
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

func (b *Block) SetHeight(height uint64) {
	b.header.height = height
}

func (b *Block) SetTimestamp(timestamp int64) {
	b.header.timestamp = timestamp
}

func (b *Block) SetTransactions(txs []*transaction.Transaction) {
	b.transactions = txs
}

func (b *Block) IsSigned() bool {
	return b.header != nil && b.header.signature != nil
}

func (b *Block) ToProto() proto.Message {

	var txArray []*transactionpb.Transaction
	for _, tx := range b.transactions {
		txArray = append(txArray, tx.ToProto().(*transactionpb.Transaction))
	}

	return &blockpb.Block{
		Header:       b.header.ToProto().(*blockpb.BlockHeader),
		Transactions: txArray,
	}
}

func (b *Block) FromProto(pb proto.Message) {

	bh := BlockHeader{}
	bh.FromProto(pb.(*blockpb.Block).GetHeader())
	b.header = &bh

	var txs []*transaction.Transaction

	for _, txpb := range pb.(*blockpb.Block).GetTransactions() {
		tx := &transaction.Transaction{}
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
	return rawBytes
}

func Deserialize(d []byte) *Block {
	pb := &blockpb.Block{}
	err := proto.Unmarshal(d, pb)
	if err != nil {
		logger.WithError(err).Panic("Block: Cannot deserialize block!")
	}
	block := &Block{}
	block.FromProto(pb)
	return block
}

func (b *Block) GetCoinbaseTransaction() *transaction.Transaction {
	//the coinbase transaction is usually placed at the end of all transactions
	for i := len(b.transactions) - 1; i >= 0; i-- {
		adaptedTx := transaction.NewTxAdapter(b.transactions[i])
		if adaptedTx.IsCoinbase() {
			return adaptedTx.Transaction
		}
	}
	return nil
}
