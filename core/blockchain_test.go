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
	"errors"
	"os"
	"testing"

	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/storage/mocks"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	logger.SetLevel(logger.WarnLevel)
	retCode := m.Run()
	os.Exit(retCode)
}

func TestCreateBlockchain(t *testing.T) {
	//create a new block chain
	s := storage.NewRamStorage()
	defer s.Close()

	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc := CreateBlockchain(addr, s, nil, 128, nil)

	//find next block. This block should be the genesis block and its prev hash should be empty
	blk, err := bc.Next()
	assert.Nil(t, err)
	assert.Empty(t, blk.GetPrevHash())
}

func TestBlockchain_HigherThanBlockchainTestHigher(t *testing.T) {
	//create a new block chain
	s := storage.NewRamStorage()
	defer s.Close()

	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc := CreateBlockchain(addr, s, nil, 128, nil)
	blk := GenerateMockBlock()
	blk.header.height = 1
	assert.True(t, bc.IsHigherThanBlockchain(blk))
}

func TestBlockchain_HigherThanBlockchainTestLower(t *testing.T) {
	//create a new block chain
	s := storage.NewRamStorage()
	defer s.Close()

	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc := CreateBlockchain(addr, s, nil, 128, nil)
	tailblk, _ := bc.GetTailBlock()
	blk := GenerateBlockWithCbtx(addr, tailblk)
	blk.header.height = 1
	bc.AddBlockToTail(blk)

	assert.False(t, bc.IsHigherThanBlockchain(blk))

}

func TestBlockchain_IsInBlockchain(t *testing.T) {
	//create a new block chain
	s := storage.NewRamStorage()
	defer s.Close()

	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc := CreateBlockchain(addr, s, nil, 128, nil)

	blk := GenerateUtxoMockBlockWithoutInputs()
	bc.AddBlockToTail(blk)

	isFound := bc.IsInBlockchain([]byte("hash"))
	assert.True(t, isFound)

	isFound = bc.IsInBlockchain([]byte("hash2"))
	assert.False(t, isFound)
}

func TestBlockchain_RollbackToABlock(t *testing.T) {
	//create a mock blockchain with max height of 5
	bc := GenerateMockBlockchainWithCoinbaseTxOnly(5)
	defer bc.db.Close()

	blk, err := bc.GetTailBlock()
	assert.Nil(t, err)

	//find the hash at height 3 (5-2)
	for i := 0; i < 2; i++ {
		blk, err = bc.GetBlockByHash(blk.GetPrevHash())
		assert.Nil(t, err)
	}

	//rollback to height 3
	bc.Rollback(blk.GetHash())

	//the height 3 block should be the new tail block
	newTailBlk, err := bc.GetTailBlock()
	assert.Nil(t, err)
	assert.Equal(t, blk.GetHash(), newTailBlk.GetHash())

}

func TestBlockchain_AddBlockToTail(t *testing.T) {

	// Serialized data of an empty UTXOIndex (generated using `hex.EncodeToString(UTXOIndex{}.serialize())`)
	serializedUTXOIndex, _ := hex.DecodeString(`0fff89040102ff8a00010c01ff8800000dff87020102ff880001ff8200003cff81030102ff82000104010556616c756501ff8400010a5075624b65794861736801ff8600010454786964010a0001075478496e64657801040000000aff83050102ff8c0000000fff8d05010103496e7401ff8e00000027ff850301010a5075624b65794861736801ff86000101010a5075624b657948617368010a00000004ff8a0000`)
	serializedBlk, _ := hex.DecodeString(`37ff810301010b426c6f636b53747265616d01ff82000102010648656164657201ff8400010c5472616e73616374696f6e7301ff9400000061ff8303010111426c6f636b48656164657253747265616d01ff84000106010448617368010a0001085072657648617368010a0001054e6f6e6365010400010954696d657374616d7001040001045369676e010a000106486569676874010600000022ff93020101135b5d2a636f72652e5472616e73616374696f6e01ff940001ff8600002fff85030102ff8600010401024944010a00010356696e01ff8a000104566f757401ff9200010354697001ff8e0000001dff890201010e5b5d636f72652e5458496e70757401ff8a0001ff88000040ff87030101075458496e70757401ff88000104010454786964010a000104566f757401040001095369676e6174757265010a0001065075624b6579010a0000001eff910201010f5b5d636f72652e54584f757470757401ff920001ff8c00003eff8b0301010854584f757470757401ff8c000103010556616c756501ff8e00010a5075624b65794861736801ff90000108436f6e7472616374010c0000000aff8d050102ff960000000fff9705010103496e7401ff9800000027ff8f0301010a5075624b65794861736801ff90000101010a5075624b657948617368010a000000ff89ff82010120f443b42e715541a9a280e81d09caa46beb59f5ff7850307247d929c0246babb003fcb6acebc00001010120642e4c103a7cffc99eb0c9eee0b24ebf666696b63294e807d5d0507a1520047301010201020b48656c6c6f20776f726c640001010104029896800101155a1ebed35a7fd3b7e5877702ee26f0e46604d238e700000101020000`)
	db := new(mocks.Storage)

	// Storage will allow blockchain creation to succeed
	db.On("Put", mock.Anything, mock.Anything).Return(nil)
	db.On("Get", []byte("utxo")).Return(serializedUTXOIndex, nil)
	db.On("Get", mock.Anything).Return(serializedBlk, nil)
	db.On("EnableBatch").Return()
	db.On("DisableBatch").Return()
	// Flush invoked in AddBlockToTail twice
	db.On("Flush").Return(nil).Twice()

	// Create a blockchain for testing
	addr := NewAddress("dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf")
	bc := &Blockchain{Hash{}, db, nil, nil, nil}

	// Add genesis block
	genesis := NewGenesisBlock(addr)
	err := bc.AddBlockToTail(genesis)

	// Expect batch write was used
	db.AssertCalled(t, "EnableBatch")
	db.AssertCalled(t, "Flush")
	db.AssertCalled(t, "DisableBatch")

	// Expect no error when adding genesis block
	assert.Nil(t, err)
	// Expect that blockchain tail is genesis block
	assert.Equal(t, genesis.GetHash(), Hash(bc.tailBlockHash))

	// Simulate a failure when flushing new block to storage
	simulatedFailure := errors.New("simulated storage failure")
	db.On("Flush").Return(simulatedFailure)

	// Add new block
	blk := GenerateMockBlock()
	blk.SetHash([]byte("hash1"))
	blk.header.height = 1
	err = bc.AddBlockToTail(blk)

	// Expect the coinbase tx to go through
	assert.Equal(t, nil, err)
	// Expect that the block added is the blockchain tail
	assert.Equal(t, blk.GetHash(), Hash(bc.tailBlockHash))
}
