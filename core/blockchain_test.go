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

	db := new(mocks.Storage)

	// Storage will allow blockchain creation to succeed
	db.On("Put", mock.Anything, mock.Anything).Return(nil)
	db.On("Get", []byte("utxo")).Return(serializedUTXOIndex, nil)
	db.On("EnableBatch").Return()
	db.On("DisableBatch").Return()
	// Flush invoked in AddBlockToTail twice
	db.On("Flush").Return(nil).Twice()

	// Create a blockchain for testing
	addr := NewAddress("dGDrVKjCG3sdXtDUgWZ7Fp3Q97tLhqWivf")
	bc := &Blockchain{Hash{}, db, nil, nil, nil, nil}

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

	//Expect 2 mock txs to be rejected when minting
	assert.Equal(t, int64(1), MetricsTxDoubleSpend.Count())
	// Expect the coinbase tx to go through
	assert.Equal(t, nil , err)
	// Expect that the block added is the blockchain tail
	assert.Equal(t, blk.GetHash(), Hash(bc.tailBlockHash))
}
