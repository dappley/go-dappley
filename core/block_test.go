// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package core

import (
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dappley/go-dappley/common"
	corepb "github.com/dappley/go-dappley/core/pb"
	storage2 "github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
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

var expect = []byte{0x42, 0xff, 0x81, 0x3, 0x1, 0x1, 0xb, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x1, 0xff, 0x82, 0x0, 0x1, 0x3, 0x1, 0x6, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x1, 0xff, 0x84, 0x0, 0x1, 0xc, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x1, 0xff, 0x90, 0x0, 0x1, 0x6, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74, 0x1, 0x6, 0x0, 0x0, 0x0, 0x4d, 0xff, 0x83, 0x3, 0x1, 0x1, 0x11, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x1, 0xff, 0x84, 0x0, 0x1, 0x4, 0x1, 0x4, 0x48, 0x61, 0x73, 0x68, 0x1, 0xa, 0x0, 0x1, 0x8, 0x50, 0x72, 0x65, 0x76, 0x48, 0x61, 0x73, 0x68, 0x1, 0xa, 0x0, 0x1, 0x5, 0x4e, 0x6f, 0x6e, 0x63, 0x65, 0x1, 0x4, 0x0, 0x1, 0x9, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x1, 0x4, 0x0, 0x0, 0x0, 0x22, 0xff, 0x8f, 0x2, 0x1, 0x1, 0x13, 0x5b, 0x5d, 0x2a, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x1, 0xff, 0x90, 0x0, 0x1, 0xff, 0x86, 0x0, 0x0, 0x2e, 0xff, 0x85, 0x3, 0x1, 0x2, 0xff, 0x86, 0x0, 0x1, 0x4, 0x1, 0x2, 0x49, 0x44, 0x1, 0xa, 0x0, 0x1, 0x3, 0x56, 0x69, 0x6e, 0x1, 0xff, 0x8a, 0x0, 0x1, 0x4, 0x56, 0x6f, 0x75, 0x74, 0x1, 0xff, 0x8e, 0x0, 0x1, 0x3, 0x54, 0x69, 0x70, 0x1, 0x4, 0x0, 0x0, 0x0, 0x1d, 0xff, 0x89, 0x2, 0x1, 0x1, 0xe, 0x5b, 0x5d, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x54, 0x58, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x1, 0xff, 0x8a, 0x0, 0x1, 0xff, 0x88, 0x0, 0x0, 0x40, 0xff, 0x87, 0x3, 0x1, 0x1, 0x7, 0x54, 0x58, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x1, 0xff, 0x88, 0x0, 0x1, 0x4, 0x1, 0x4, 0x54, 0x78, 0x69, 0x64, 0x1, 0xa, 0x0, 0x1, 0x4, 0x56, 0x6f, 0x75, 0x74, 0x1, 0x4, 0x0, 0x1, 0x9, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x1, 0xa, 0x0, 0x1, 0x6, 0x50, 0x75, 0x62, 0x4b, 0x65, 0x79, 0x1, 0xa, 0x0, 0x0, 0x0, 0x1e, 0xff, 0x8d, 0x2, 0x1, 0x1, 0xf, 0x5b, 0x5d, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x54, 0x58, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x1, 0xff, 0x8e, 0x0, 0x1, 0xff, 0x8c, 0x0, 0x0, 0x2f, 0xff, 0x8b, 0x3, 0x1, 0x1, 0x8, 0x54, 0x58, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x1, 0xff, 0x8c, 0x0, 0x1, 0x2, 0x1, 0x5, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x1, 0x4, 0x0, 0x1, 0xa, 0x50, 0x75, 0x62, 0x4b, 0x65, 0x79, 0x48, 0x61, 0x73, 0x68, 0x1, 0xa, 0x0, 0x0, 0x0, 0x13, 0xff, 0x82, 0x1, 0x2, 0x1, 0x61, 0x2, 0xfc, 0xb6, 0xb2, 0x24, 0x6a, 0x0, 0x1, 0x1, 0x0, 0x1, 0x1, 0x0}
var expectHash = []uint8([]byte{0x5d, 0xf6, 0xe0, 0xe2, 0x76, 0x13, 0x59, 0xd3, 0xa, 0x82, 0x75, 0x5, 0x8e, 0x29, 0x9f, 0xcc, 0x3, 0x81, 0x53, 0x45, 0x45, 0xf5, 0x5c, 0xf4, 0x3e, 0x41, 0x98, 0x3f, 0x5d, 0x4c, 0x94, 0x56})
var header2 = &BlockHeader{
	hash:      []byte{'a'},
	prevHash:  []byte{'e', 'c'},
	nonce:     0,
	timestamp: time.Now().Unix(),
}
var blk2 = &Block{
	header: header2,
}

var header3 = &BlockHeader{
	hash:      []byte{'a'},
	prevHash:  []byte{'e', 'c'},
	nonce:     0,
	timestamp: 0,
}
var blk3 = &Block{
	header: header3,
}

func TestHashTransactions(t *testing.T) {
	block := NewBlock([]*Transaction{{}}, blk2)
	hash := block.HashTransactions()
	assert.Equal(t, expectHash, hash)
}

func TestNewBlock(t *testing.T) {
	var emptyTx = []*Transaction([]*Transaction{})
	var emptyHash = Hash(Hash{})
	var expectBlock3Hash = Hash{0x61}
	block1 := NewBlock(nil, nil)
	assert.Nil(t, block1.header.prevHash)
	assert.Equal(t, emptyTx, block1.transactions)

	block2 := NewBlock(nil, blk)
	assert.Equal(t, emptyHash, block2.header.prevHash)
	assert.Equal(t, Hash(Hash{}), block2.header.prevHash)
	assert.Equal(t, emptyTx, block2.transactions)

	block3 := NewBlock(nil, blk2)
	assert.Equal(t, expectBlock3Hash, block3.header.prevHash)
	assert.Equal(t, Hash(Hash{'a'}), block3.header.prevHash)
	assert.Equal(t, []byte{'a'}[0], block3.header.prevHash[0])
	assert.Equal(t, uint64(1), block3.header.height)
	assert.Equal(t, emptyTx, block3.transactions)

	block4 := NewBlock([]*Transaction{}, nil)
	assert.Nil(t, block4.header.prevHash)
	assert.Equal(t, emptyTx, block4.transactions)
	assert.Equal(t, Hash(nil), block4.header.prevHash)

	block5 := NewBlock([]*Transaction{{}}, nil)
	assert.Nil(t, block5.header.prevHash)
	assert.Equal(t, []*Transaction{{}}, block5.transactions)
	assert.Equal(t, &Transaction{}, block5.transactions[0])
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
	}

	pb := bh1.ToProto()
	var i interface{} = pb
	_, correct := i.(proto.Message)
	assert.Equal(t, true, correct)
	mpb, err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &corepb.BlockHeader{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	bh2 := BlockHeader{}
	bh2.FromProto(newpb)

	assert.Equal(t, bh1, bh2)
}

func TestBlock_Proto(t *testing.T) {

	b1 := GenerateMockBlock()

	pb := b1.ToProto()
	var i interface{} = pb
	_, correct := i.(proto.Message)
	assert.Equal(t, true, correct)
	mpb, err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &corepb.Block{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	b2 := &Block{}
	b2.FromProto(newpb)

	assert.Equal(t, *b1, *b2)
}

func TestBlock_VerifyHash(t *testing.T) {
	b1 := GenerateMockBlock()

	//The mocked block does not have correct hash Value
	assert.False(t, b1.VerifyHash())

	//calculate correct hash Value
	hash := b1.CalculateHash()
	b1.SetHash(hash)
	assert.True(t, b1.VerifyHash())

	//calculate a hash Value with a different nonce
	hash = b1.CalculateHashWithNonce(b1.GetNonce() + 1)
	b1.SetHash(hash)
	assert.False(t, b1.VerifyHash())

	hash = b1.CalculateHashWithoutNonce()
	b1.SetHash(hash)
	assert.False(t, b1.VerifyHash())
}

func TestBlock_Rollback(t *testing.T) {
	b := GenerateMockBlock()
	tx := MockTransaction()
	b.transactions = []*Transaction{tx}
	txPool := NewTransactionPool(128)
	b.Rollback(txPool)
	assert.ElementsMatch(t, tx.ID, txPool.GetTransactions()[0].ID)
}

func TestBlock_FindTransaction(t *testing.T) {
	b := GenerateMockBlock()
	tx := MockTransaction()
	b.transactions = []*Transaction{tx}

	assert.Equal(t, tx.ID, b.FindTransactionById(tx.ID).ID)
}

func TestBlock_FindTransactionNilInput(t *testing.T) {
	b := GenerateMockBlock()
	assert.Nil(t, b.FindTransactionById(nil))
}

func TestBlock_FindTransactionEmptyBlock(t *testing.T) {
	b := GenerateMockBlock()
	tx := MockTransaction()
	assert.Nil(t, b.FindTransactionById(tx.ID))
}

func TestIsParentBlockHash(t *testing.T) {
	parentBlock := NewBlock([]*Transaction{{}}, blk2)
	childBlock := NewBlock([]*Transaction{{}}, parentBlock)

	assert.True(t, IsParentBlockHash(parentBlock, childBlock))
	assert.False(t, IsParentBlockHash(parentBlock, nil))
	assert.False(t, IsParentBlockHash(nil, childBlock))
	assert.False(t, IsParentBlockHash(childBlock, parentBlock))
}

func TestIsParentBlockHeight(t *testing.T) {
	parentBlock := NewBlock([]*Transaction{{}}, blk2)
	childBlock := NewBlock([]*Transaction{{}}, parentBlock)

	assert.True(t, IsParentBlockHeight(parentBlock, childBlock))
	assert.False(t, IsParentBlockHeight(parentBlock, nil))
	assert.False(t, IsParentBlockHeight(nil, childBlock))
	assert.False(t, IsParentBlockHeight(childBlock, parentBlock))
}
func TestCalculateHashWithNonce(t *testing.T) {
	block := NewBlock([]*Transaction{{}}, blk3)
	block.header.timestamp = 0
	expectHash1 := Hash{0x3f, 0x2f, 0xec, 0xb4, 0x33, 0xf0, 0xd1, 0x1a, 0xa6, 0xf4, 0xf, 0xb8, 0x7f, 0x8f, 0x99, 0x11, 0xae, 0xe7, 0x42, 0xf4, 0x69, 0x7d, 0xf1, 0xaa, 0xc8, 0xd0, 0xfc, 0x40, 0xa2, 0xd8, 0xb1, 0xa5}
	assert.Equal(t, Hash(expectHash1), block.CalculateHashWithNonce(1))
	expectHash2 := Hash{0xe7, 0x57, 0x13, 0xc6, 0x8a, 0x98, 0x58, 0xb3, 0x5, 0x70, 0x6e, 0x33, 0xf0, 0x95, 0xd8, 0x1a, 0xbc, 0x76, 0xef, 0x30, 0x14, 0x59, 0x88, 0x11, 0x3c, 0x11, 0x59, 0x92, 0x65, 0xd5, 0xd3, 0x4c}
	assert.Equal(t, Hash(expectHash2), block.CalculateHashWithNonce(2))
}

func TestBlock_VerifyTransactions(t *testing.T) {
	// Prepare test data
	normalCoinbaseTX := NewCoinbaseTX(address1Hash.GenerateAddress(), "", 1, common.NewAmount(0))
	rewardTX := NewRewardTx(1, map[string]string{address1Hash.GenerateAddress().String(): "10"})
	userPubKey := NewKeyPair().PublicKey
	userPubKeyHash, _ := NewUserPubKeyHash(userPubKey)
	userAddr := userPubKeyHash.GenerateAddress()
	contractPubKeyHash := NewContractPubKeyHash()
	contractAddr := contractPubKeyHash.GenerateAddress()

	txIdStr := "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	generatedTxId, err := hex.DecodeString(txIdStr)
	assert.Nil(t, err)
	fmt.Println(hex.EncodeToString(generatedTxId))
	generatedTX := &Transaction{
		generatedTxId,
		[]TXInput{
			{[]byte("prevtxid"), 0, []byte("txid"), []byte(contractPubKeyHash)},
			{[]byte("prevtxid"), 1, []byte("txid"), []byte(contractPubKeyHash)},
		},
		[]TXOutput{
			*NewTxOut(common.NewAmount(23), userAddr, ""),
			*NewTxOut(common.NewAmount(10), contractAddr, ""),
		},
		common.NewAmount(7),
	}

	var prikey1 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa71"
	var pubkey1 = GetKeyPairByString(prikey1).PublicKey
	var pkHash1, _ = NewUserPubKeyHash(pubkey1)
	var prikey2 = "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa72"
	var pubkey2 = GetKeyPairByString(prikey2).PublicKey
	var pkHash2, _ = NewUserPubKeyHash(pubkey2)

	dependentTx1 := NewTransactionByVin(tx1.ID, 1, pubkey1, 10, pkHash2, 3)
	dependentTx2 := NewTransactionByVin(dependentTx1.ID, 0, pubkey2, 5, pkHash1, 5)
	dependentTx3 := NewTransactionByVin(dependentTx2.ID, 0, pubkey1, 1, pkHash2, 4)

	tx2Utxo1 := UTXO{dependentTx2.Vout[0], dependentTx2.ID, 0, UtxoNormal}

	tx1Utxos := map[string][]*UTXO{
		hex.EncodeToString(pkHash2): {&UTXO{dependentTx1.Vout[0], dependentTx1.ID, 0, UtxoNormal}},
	}
	dependentTx2.Sign(GetKeyPairByString(prikey2).PrivateKey, tx1Utxos[hex.EncodeToString(pkHash2)])
	dependentTx3.Sign(GetKeyPairByString(prikey1).PrivateKey, []*UTXO{&tx2Utxo1})

	tests := []struct {
		name   string
		txs    []*Transaction
		utxos  map[string][]*UTXO
		txPool *TransactionPool
		ok     bool
	}{
		{
			"normal txs",
			[]*Transaction{&normalCoinbaseTX},
			map[string][]*UTXO{},
			nil,
			true,
		},
		{"no txs", []*Transaction{}, make(map[string][]*UTXO), nil, true},
		{
			"invalid normal txs",
			[]*Transaction{{
				ID: []byte("txid"),
				Vin: []TXInput{{
					[]byte("tx1"),
					0,
					util.GenerateRandomAoB(2),
					address1Bytes,
				}},
				Vout: MockUtxoOutputsWithInputs(),
				Tip:  common.NewAmount(5),
			}},
			map[string][]*UTXO{},
			nil,
			false,
		},
		{
			"normal dependent txs",
			[]*Transaction{&dependentTx2, &dependentTx3},
			tx1Utxos,
			nil,
			true,
		},
		{
			"invalid dependent txs",
			[]*Transaction{&dependentTx3, &dependentTx2},
			tx1Utxos,
			nil,
			false,
		},
		{
			"reward tx",
			[]*Transaction{&rewardTX},
			map[string][]*UTXO{
				hex.EncodeToString(contractPubKeyHash): {
					{*NewTXOutput(common.NewAmount(0), contractAddr), []byte("prevtxid"), 0, UtxoNormal},
				},
				hex.EncodeToString(userPubKeyHash): {
					{*NewTXOutput(common.NewAmount(1), userAddr), []byte("txinid"), 0, UtxoNormal},
				},
			},
			nil,
			false,
		},
		{
			"generated tx",
			[]*Transaction{generatedTX},
			map[string][]*UTXO{
				hex.EncodeToString(contractPubKeyHash): {
					{*NewTXOutput(common.NewAmount(20), contractAddr), []byte("prevtxid"), 0, UtxoNormal},
					{*NewTXOutput(common.NewAmount(20), contractAddr), []byte("prevtxid"), 1, UtxoNormal},
				},
				hex.EncodeToString(userPubKeyHash): {
					{*NewTXOutput(common.NewAmount(1), userAddr), []byte("txinid"), 0, UtxoNormal},
				},
			},
			&TransactionPool{txs: map[string]*TransactionNode{"contractGenerated": {Value: generatedTX}}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := storage2.NewRamStorage()
			index := make(map[string]*UTXOTx)

			for key, addrUtxos := range tt.utxos {
				utxoTx := NewUTXOTx()
				for _, addrUtxo := range addrUtxos {
					utxoTx.PutUtxo(addrUtxo)
				}
				index[key] = &utxoTx
			}

			utxoIndex := UTXOIndex{index, NewUTXOCache(db), &sync.RWMutex{}}
			scState := NewScState()
			block := NewBlock(tt.txs, blk)
			assert.Equal(t, tt.ok, block.VerifyTransactions(&utxoIndex, scState, nil, block))
		})
	}
}
