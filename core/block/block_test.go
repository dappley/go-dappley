package block

import (
	"github.com/dappley/go-dappley/core/transaction"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"testing"
	"time"

	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core/block/pb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

var header = &BlockHeader{
	hash:      []byte{},
	prevHash:  []byte{},
	nonce:     0,
	timestamp: time.Now().Unix(),
}
var blk = &Block{
	header: header,
}

var header2 = &BlockHeader{
	hash:      []byte{'a'},
	prevHash:  []byte{'e', 'c'},
	nonce:     0,
	timestamp: time.Now().Unix(),
}
var blk2 = &Block{
	header: header2,
}

func TestNewBlock(t *testing.T) {
	var emptyTx = []*transaction.Transaction([]*transaction.Transaction{})
	var emptyHash = hash.Hash(hash.Hash{})
	var expectBlock3Hash = hash.Hash{0x61}

	block1 := NewBlock(nil, nil, "")
	assert.Nil(t, block1.header.prevHash)
	assert.Equal(t, emptyTx, block1.transactions)

	block2 := NewBlock(nil, blk, "")
	assert.Equal(t, emptyHash, block2.header.prevHash)
	assert.Equal(t, hash.Hash(hash.Hash{}), block2.header.prevHash)
	assert.Equal(t, emptyTx, block2.transactions)

	block3 := NewBlock(nil, blk2, "")
	assert.Equal(t, expectBlock3Hash, block3.header.prevHash)
	assert.Equal(t, hash.Hash(hash.Hash{'a'}), block3.header.prevHash)
	assert.Equal(t, []byte{'a'}[0], block3.header.prevHash[0])
	assert.Equal(t, uint64(1), block3.header.height)
	assert.Equal(t, emptyTx, block3.transactions)

	block4 := NewBlock([]*transaction.Transaction{}, nil, "")
	assert.Nil(t, block4.header.prevHash)
	assert.Equal(t, emptyTx, block4.transactions)
	assert.Equal(t, hash.Hash(nil), block4.header.prevHash)

	block5 := NewBlock([]*transaction.Transaction{{}}, nil, "")
	assert.Nil(t, block5.header.prevHash)
	assert.Equal(t, []*transaction.Transaction{{}}, block5.transactions)
	assert.Equal(t, &transaction.Transaction{}, block5.transactions[0])
	assert.NotNil(t, block5.transactions)
}

func TestBlockHeader_Proto(t *testing.T) {
	bh1 := BlockHeader{
		[]byte("hash"),
		[]byte("hash"),
		1,
		2,
		nil,
		0,
		"",
	}

	pb := bh1.ToProto()
	var i interface{} = pb
	_, correct := i.(proto.Message)
	assert.Equal(t, true, correct)
	mpb, err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &blockpb.BlockHeader{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	bh2 := BlockHeader{}
	bh2.FromProto(newpb)

	assert.Equal(t, bh1, bh2)
}

func TestBlock_ToProto(t *testing.T) {
	b1 := GenerateMockBlock()

	var txArray []*transactionpb.Transaction
	for _, tx := range b1.transactions {
		txArray = append(txArray, tx.ToProto().(*transactionpb.Transaction))
	}
	expected := &blockpb.Block{
		Header: &blockpb.BlockHeader{
			Hash: b1.header.hash,
			PreviousHash: b1.header.prevHash,
			Nonce: b1.header.nonce,
			Timestamp: b1.header.timestamp,
			Signature: b1.header.signature,
			Height: b1.header.height,
			Producer: b1.header.producer,
		},
		Transactions: txArray,
	}

	assert.Equal(t, expected, b1.ToProto())
}

func TestBlock_FromProto(t *testing.T) {
	expected := GenerateMockBlock()

	var txArray []*transactionpb.Transaction
	for _, tx := range expected.transactions {
		txArray = append(txArray, tx.ToProto().(*transactionpb.Transaction))
	}
	blockProto := &blockpb.Block{
		Header: &blockpb.BlockHeader{
			Hash: expected.header.hash,
			PreviousHash: expected.header.prevHash,
			Nonce: expected.header.nonce,
			Timestamp: expected.header.timestamp,
			Signature: expected.header.signature,
			Height: expected.header.height,
			Producer: expected.header.producer,
		},
		Transactions: txArray,
	}

	b1 := &Block{}
	b1.FromProto(blockProto)
	assert.Equal(t, expected, b1)
}

func TestBlock_IsSigned(t *testing.T) {
	block := NewBlock(nil, nil, "")
	assert.False(t, block.IsSigned())

	block.SetSignature(hash.Hash{0x88})
	assert.True(t, block.IsSigned())
}

func TestBlock_Serialize(t *testing.T) {
	block := GenerateMockBlock()
	serializedBytes, _ := proto.Marshal(block.ToProto())
	assert.Equal(t, serializedBytes, block.Serialize())
}

func TestDeserialize(t *testing.T) {
	rawBytes := []byte{10, 10, 32, 2, 48, 1, 58, 4, 116, 101, 115, 116}
	b1 := Deserialize(rawBytes)

	expectedBlock := NewBlockWithTimestamp(nil, nil, 2, "test")

	assert.Equal(t, expectedBlock.header, b1.header)
	assert.Equal(t, expectedBlock.transactions, b1.transactions)
}

func TestBlock_GetCoinbaseTransaction(t *testing.T) {
	b1 := NewBlock(nil, nil, "")
	assert.Nil(t, b1.GetCoinbaseTransaction())

	b2 := GenerateMockBlock()
	b2.transactions[1].Type = transaction.TxTypeCoinbase
	assert.Equal(t, b2.transactions[1], b2.GetCoinbaseTransaction())
}
