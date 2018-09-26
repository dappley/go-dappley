// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
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
	"github.com/dappley/go-dappley/common"
	"time"

	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
)

func GenerateMockBlock() *Block {
	bh1 := &BlockHeader{
		[]byte("hash"),
		[]byte("prevhash"),
		1,
		time.Now().Unix(),
		nil,
		0,
	}

	t1 := MockTransaction()
	t2 := MockTransaction()

	return &Block{
		header:       bh1,
		transactions: []*Transaction{t1, t2},
	}
}

func FakeNewBlockWithTimestamp(t int64, transactions []*Transaction, parent *Block) *Block {
	var prevHash []byte
	var height uint64
	height = 0
	if parent != nil {
		prevHash = parent.GetHash()
		height = parent.GetHeight() + 1
	}

	if transactions == nil {
		transactions = []*Transaction{}
	}
	return &Block{
		header: &BlockHeader{
			hash:      []byte{},
			prevHash:  prevHash,
			nonce:     0,
			timestamp: t,
			sign: nil,
			height:height,
		},
		transactions: transactions,
	}
}

func GenerateMockBlockchain(size int) *Blockchain {
	//create a new block chain
	s := storage.NewRamStorage()
	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc := CreateBlockchain(addr, s, nil)

	for i := 0; i < size; i++ {
		tailBlk, _ := bc.GetTailBlock()
		b := NewBlock([]*Transaction{MockTransaction()}, tailBlk)
		b.SetHash(b.CalculateHash())
		bc.AddBlockToTail(b)
	}
	return bc
}

func GenerateMockBlockchainWithCoinbaseTxOnlyWithConsensus(size int, consensus Consensus) *Blockchain {
	//create a new block chain
	s := storage.NewRamStorage()
	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc := CreateBlockchain(addr, s, consensus)

	for i := 0; i < size; i++ {
		tailBlk, _ := bc.GetTailBlock()
		cbtx := NewCoinbaseTX(addr.Address, "", bc.GetMaxHeight())
		b := NewBlock([]*Transaction{&cbtx}, tailBlk)
		b.SetHash(b.CalculateHash())
		bc.AddBlockToTail(b)
	}
	return bc
}

func GenerateMockBlockchainWithCoinbaseTxOnly(size int) *Blockchain {
	//create a new block chain
	s := storage.NewRamStorage()
	addr := NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc := CreateBlockchain(addr, s, nil)

	for i := 0; i < size; i++ {
		tailBlk, _ := bc.GetTailBlock()
		cbtx := NewCoinbaseTX(addr.Address, "", bc.GetMaxHeight())
		b := NewBlock([]*Transaction{&cbtx}, tailBlk)
		b.SetHash(b.CalculateHash())
		bc.AddBlockToTail(b)
	}
	return bc
}

//the first item is the tail of the fork
func GenerateMockForkWithValidTx(size int, parent *Block) []*Block {
	fork := []*Block{}
	b := NewBlock(nil, parent)
	b.SetHash(b.CalculateHash())
	fork = append(fork, b)

	for i := 1; i < size; i++ {
		b = NewBlock([]*Transaction{MockTransaction()}, b)
		b.SetHash(b.CalculateHash())
		fork = append([]*Block{b}, fork...)
	}
	return fork
}

func MockTransaction() *Transaction {
	return &Transaction{
		ID:   util.GenerateRandomAoB(1),
		Vin:  MockTxInputs(),
		Vout: MockTxOutputs(),
		Tip:  5,
	}
}

func MockTxInputs() []TXInput {
	return []TXInput{
		{util.GenerateRandomAoB(2),
			6,
			util.GenerateRandomAoB(2),
			util.GenerateRandomAoB(2)},
		{util.GenerateRandomAoB(2),
			2,
			util.GenerateRandomAoB(2),
			util.GenerateRandomAoB(2)},
	}
}

func MockTxOutputs() []TXOutput {
	return []TXOutput{
		{common.NewAmount(5), util.GenerateRandomAoB(2)},
		{common.NewAmount(7), util.GenerateRandomAoB(2)},
	}
}

func GenerateMockTransactionPool(numOfTxs int) *TransactionPool {
	txPool := &TransactionPool{}
	for i := 0; i < numOfTxs; i++ {
		txPool.Transactions.StructPush(*MockTransaction())
	}
	return txPool
}

