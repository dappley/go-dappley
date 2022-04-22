package block

import (
	"testing"
	"time"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	blockpb "github.com/dappley/go-dappley/core/block/pb"
	"github.com/dappley/go-dappley/core/transaction"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/core/transactionbase"

	"github.com/dappley/go-dappley/common/hash"
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

func TestNewBlockWithTimestamp(t *testing.T) {
	block1 := NewBlockWithTimestamp(nil, nil, 123456789, "")
	assert.Equal(t, int64(123456789), block1.header.timestamp)

	block2 := NewBlockWithTimestamp(nil, nil, 0, "")
	assert.Equal(t, int64(0), block2.header.timestamp)

	block3 := NewBlockWithTimestamp(nil, nil, -10, "")
	assert.Equal(t, int64(-10), block3.header.timestamp)
}

func TestNewBlockWithRawInfo(t *testing.T) {
	var emptyTx = []*transaction.Transaction([]*transaction.Transaction{})
	var emptyHash = hash.Hash(hash.Hash{})

	block1 := NewBlockWithRawInfo(nil, nil, 15, 16, 17, emptyTx)
	assert.Nil(t, block1.header.hash)
	assert.Nil(t, block1.header.prevHash)
	assert.Equal(t, int64(15), block1.header.nonce)
	assert.Equal(t, int64(16), block1.header.timestamp)
	assert.Equal(t, uint64(17), block1.header.height)
	assert.Equal(t, emptyTx, block1.transactions)

	block2 := NewBlockWithRawInfo(emptyHash, emptyHash, 18, 19, 20, emptyTx)
	assert.Equal(t, emptyHash, block2.header.hash)
	assert.Equal(t, emptyHash, block2.header.prevHash)
	assert.Equal(t, int64(18), block2.header.nonce)
	assert.Equal(t, int64(19), block2.header.timestamp)
	assert.Equal(t, uint64(20), block2.header.height)
	assert.Equal(t, emptyTx, block2.transactions)
}

func TestNewBlockByHash(t *testing.T) {
	var emptyTx = []*transaction.Transaction([]*transaction.Transaction{})
	var emptyHash = hash.Hash(hash.Hash{})
	var nonEmptyHash = hash.Hash(hash.Hash{0x61})

	block1 := NewBlockByHash(nil, "")
	assert.Equal(t, emptyHash, block1.header.hash)
	assert.Nil(t, block1.header.prevHash)
	assert.Equal(t, int64(0), block1.header.nonce)
	assert.Equal(t, int64(0), block1.header.timestamp)
	assert.Equal(t, emptyHash, block1.header.signature)
	assert.Equal(t, uint64(0), block1.header.height)
	assert.Equal(t, "", block1.header.producer)
	assert.Equal(t, emptyTx, block1.transactions)

	block2 := NewBlockByHash(emptyHash, "producer")
	assert.Equal(t, emptyHash, block2.header.hash)
	assert.Equal(t, emptyHash, block2.header.prevHash)
	assert.Equal(t, int64(0), block2.header.nonce)
	assert.Equal(t, int64(0), block2.header.timestamp)
	assert.Equal(t, "producer", block2.header.producer)
	assert.Equal(t, uint64(0), block2.header.height)
	assert.Equal(t, emptyHash, block2.header.signature)
	assert.Equal(t, emptyTx, block2.transactions)

	block3 := NewBlockByHash(nonEmptyHash, "")
	assert.Equal(t, emptyHash, block3.header.hash)
	assert.Equal(t, nonEmptyHash, block3.header.prevHash)
	assert.Equal(t, int64(0), block3.header.nonce)
	assert.Equal(t, int64(0), block3.header.timestamp)
	assert.Equal(t, "", block3.header.producer)
	assert.Equal(t, uint64(0), block3.header.height)
	assert.Equal(t, emptyHash, block3.header.signature)
	assert.Equal(t, emptyTx, block3.transactions)
}

func TestNewBlockHeader(t *testing.T) {
	var emptyHash = hash.Hash(hash.Hash{})

	header1 := NewBlockHeader(nil, nil, 15, 16, 17)
	assert.Nil(t, header1.hash)
	assert.Nil(t, header1.prevHash)
	assert.Equal(t, int64(15), header1.nonce)
	assert.Equal(t, int64(16), header1.timestamp)
	assert.Equal(t, uint64(17), header1.height)

	header2 := NewBlockHeader(emptyHash, emptyHash, 18, 19, 20)
	assert.Equal(t, emptyHash, header2.hash)
	assert.Equal(t, emptyHash, header2.prevHash)
	assert.Equal(t, int64(18), header2.nonce)
	assert.Equal(t, int64(19), header2.timestamp)
	assert.Equal(t, uint64(20), header2.height)
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
			Hash:         b1.header.hash,
			PreviousHash: b1.header.prevHash,
			Nonce:        b1.header.nonce,
			Timestamp:    b1.header.timestamp,
			Signature:    b1.header.signature,
			Height:       b1.header.height,
			Producer:     b1.header.producer,
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
			Hash:         expected.header.hash,
			PreviousHash: expected.header.prevHash,
			Nonce:        expected.header.nonce,
			Timestamp:    expected.header.timestamp,
			Signature:    expected.header.signature,
			Height:       expected.header.height,
			Producer:     expected.header.producer,
		},
		Transactions: txArray,
	}

	b1 := &Block{}
	b1.FromProto(blockProto)
	assert.Equal(t, expected, b1)
}

func TestBlock_IsSigned(t *testing.T) {
	block := &Block{
		header: &BlockHeader{
			hash:      []byte{},
			prevHash:  nil,
			nonce:     0,
			timestamp: 0,
			signature: nil,
			height:    0,
			producer:  "",
		},
		transactions: []*transaction.Transaction{},
	}
	assert.False(t, block.IsSigned())

	block.header.signature = hash.Hash{0x88}
	assert.True(t, block.IsSigned())

	block.header = nil
	assert.False(t, block.IsSigned())
}

func TestBlock_Serialize(t *testing.T) {
	block := &Block{
		header: &BlockHeader{
			hash:      hash.Hash{0x68, 0x61, 0x73, 0x68},
			prevHash:  hash.Hash{0x70, 0x72, 0x65, 0x76, 0x68, 0x61, 0x73, 0x68},
			nonce:     1,
			timestamp: 1623087951,
			signature: hash.Hash{0x58},
			height:    0,
			producer:  "producer",
		},
		transactions: []*transaction.Transaction{
			{
				ID: []byte{0x67},
				Vin: []transactionbase.TXInput{
					{Txid: []byte{0x52, 0xfd}, Vout: 10, Signature: []byte{0xfc, 0x7}, PubKey: []byte{0x21, 0x82}},
					{Txid: []byte{0x65, 0x4f}, Vout: 5, Signature: []byte{0x16, 0x3f}, PubKey: []byte{0x5f, 0xf}},
				},
				Vout: []transactionbase.TXOutput{
					{Value: common.NewAmount(1), PubKeyHash: account.PubKeyHash{0x9a, 0x62}, Contract: ""},
					{Value: common.NewAmount(2), PubKeyHash: account.PubKeyHash{0x1d, 0x72}, Contract: ""}},
				Tip:      common.NewAmount(1),
				GasLimit: common.NewAmount(3),
				GasPrice: common.NewAmount(2),
			},
		},
	}
	expected := []byte{0xa, 0x25, 0xa, 0x4, 0x68, 0x61, 0x73, 0x68, 0x12, 0x8, 0x70, 0x72, 0x65, 0x76, 0x68, 0x61, 0x73, 0x68, 0x18, 0x1, 0x20, 0xcf, 0xb6, 0xf9, 0x85, 0x6, 0x2a, 0x1, 0x58, 0x3a, 0x8, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x65, 0x72, 0x12, 0x3e, 0xa, 0x1, 0x67, 0x12, 0xe, 0xa, 0x2, 0x52, 0xfd, 0x10, 0xa, 0x1a, 0x2, 0xfc, 0x7, 0x22, 0x2, 0x21, 0x82, 0x12, 0xe, 0xa, 0x2, 0x65, 0x4f, 0x10, 0x5, 0x1a, 0x2, 0x16, 0x3f, 0x22, 0x2, 0x5f, 0xf, 0x1a, 0x7, 0xa, 0x1, 0x1, 0x12, 0x2, 0x9a, 0x62, 0x1a, 0x7, 0xa, 0x1, 0x2, 0x12, 0x2, 0x1d, 0x72, 0x22, 0x1, 0x1, 0x2a, 0x1, 0x3, 0x32, 0x1, 0x2}
	assert.Equal(t, expected, block.Serialize())
}

func TestDeserialize(t *testing.T) {
	rawBytes := []byte{0xa, 0x25, 0xa, 0x4, 0x68, 0x61, 0x73, 0x68, 0x12, 0x8, 0x70, 0x72, 0x65, 0x76, 0x68, 0x61, 0x73, 0x68, 0x18, 0x1, 0x20, 0xcf, 0xb6, 0xf9, 0x85, 0x6, 0x2a, 0x1, 0x58, 0x3a, 0x8, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x65, 0x72, 0x12, 0x3e, 0xa, 0x1, 0x67, 0x12, 0xe, 0xa, 0x2, 0x52, 0xfd, 0x10, 0xa, 0x1a, 0x2, 0xfc, 0x7, 0x22, 0x2, 0x21, 0x82, 0x12, 0xe, 0xa, 0x2, 0x65, 0x4f, 0x10, 0x5, 0x1a, 0x2, 0x16, 0x3f, 0x22, 0x2, 0x5f, 0xf, 0x1a, 0x7, 0xa, 0x1, 0x1, 0x12, 0x2, 0x9a, 0x62, 0x1a, 0x7, 0xa, 0x1, 0x2, 0x12, 0x2, 0x1d, 0x72, 0x22, 0x1, 0x1, 0x2a, 0x1, 0x3, 0x32, 0x1, 0x2}
	block := Deserialize(rawBytes)

	expectedBlock := &Block{
		header: &BlockHeader{
			hash:      hash.Hash{0x68, 0x61, 0x73, 0x68},
			prevHash:  hash.Hash{0x70, 0x72, 0x65, 0x76, 0x68, 0x61, 0x73, 0x68},
			nonce:     1,
			timestamp: 1623087951,
			signature: hash.Hash{0x58},
			height:    0,
			producer:  "producer",
		},
		transactions: []*transaction.Transaction{
			{
				ID: []byte{0x67},
				Vin: []transactionbase.TXInput{
					{Txid: []byte{0x52, 0xfd}, Vout: 10, Signature: []byte{0xfc, 0x7}, PubKey: []byte{0x21, 0x82}},
					{Txid: []byte{0x65, 0x4f}, Vout: 5, Signature: []byte{0x16, 0x3f}, PubKey: []byte{0x5f, 0xf}},
				},
				Vout: []transactionbase.TXOutput{
					{Value: common.NewAmount(1), PubKeyHash: account.PubKeyHash{0x9a, 0x62}, Contract: ""},
					{Value: common.NewAmount(2), PubKeyHash: account.PubKeyHash{0x1d, 0x72}, Contract: ""}},
				Tip:      common.NewAmount(1),
				GasLimit: common.NewAmount(3),
				GasPrice: common.NewAmount(2),
			},
		},
	}

	assert.Equal(t, expectedBlock, block)
}

func TestBlock_GetCoinbaseTransaction(t *testing.T) {
	b1 := &Block{
		header: &BlockHeader{
			hash:      []byte{},
			prevHash:  nil,
			nonce:     0,
			timestamp: 0,
			signature: nil,
			height:    0,
			producer:  "",
		},
		transactions: []*transaction.Transaction{},
	}
	assert.Nil(t, b1.GetCoinbaseTransaction())

	b2 := GenerateMockBlock()
	assert.Nil(t, b2.GetCoinbaseTransaction())

	b2.transactions[1].Type = transaction.TxTypeCoinbase
	assert.Equal(t, b2.transactions[1], b2.GetCoinbaseTransaction())
}
