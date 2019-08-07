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
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/logic/block"
	"time"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
)

func GenerateMockBlock() *block.Block {
	t1 := MockTransaction()
	t2 := MockTransaction()

	return block.NewBlockWithRawInfo(
		[]byte("hash"),
		[]byte("prevhash"),
		1,
		time.Now().Unix(),
		0,
		[]*Transaction{t1, t2},
	)
}

func FakeNewBlockWithTimestamp(t int64, txs []*Transaction, parent *block.Block) *block.Block {
	var prevHash []byte
	var height uint64
	height = 0
	if parent != nil {
		prevHash = parent.GetHash()
		height = parent.GetHeight() + 1
	}

	if txs == nil {
		txs = []*Transaction{}
	}
	blk := block.NewBlockWithRawInfo(
		[]byte{},
		prevHash,
		0,
		t,
		height,
		txs)

	hash := lblock.CalculateHashWithNonce(blk)
	blk.SetHash(hash)
	return blk
}

func GenerateMockBlockchain(size int) *Blockchain {
	//create a new block chain
	s := storage.NewRamStorage()

	addr := account.NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc := CreateBlockchain(addr, s, nil, NewTransactionPool(nil, 128000), nil, 100000)

	for i := 0; i < size; i++ {
		tailBlk, _ := bc.GetTailBlock()
		b := block.NewBlock([]*Transaction{MockTransaction()}, tailBlk, "16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
		b.SetHash(lblock.CalculateHash(b))
		bc.AddBlockContextToTail(PrepareBlockContext(bc, b))
	}
	return bc
}

func PrepareBlockContext(bc *Blockchain, blk *block.Block) *BlockContext {
	state := LoadScStateFromDatabase(bc.GetDb())
	utxoIndex := NewUTXOIndex(bc.GetUtxoCache())
	utxoIndex.UpdateUtxoState(blk.GetTransactions())
	ctx := BlockContext{Block: blk, UtxoIndex: utxoIndex, State: state}
	return &ctx
}

func GenerateBlockWithCbtx(addr account.Address, lastblock *block.Block) *block.Block {
	//create a new block chain
	cbtx := NewCoinbaseTX(addr, "", lastblock.GetHeight(), common.NewAmount(0))
	b := block.NewBlock([]*Transaction{&cbtx}, lastblock, "")
	return b
}
func GenerateMockBlockchainWithCoinbaseTxOnly(size int) *Blockchain {
	//create a new block chain
	s := storage.NewRamStorage()
	addr := account.NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	bc := CreateBlockchain(addr, s, nil, NewTransactionPool(nil, 128000), nil, 100000)

	for i := 0; i < size; i++ {
		tailBlk, _ := bc.GetTailBlock()
		cbtx := NewCoinbaseTX(addr, "", bc.GetMaxHeight(), common.NewAmount(0))
		b := block.NewBlock([]*Transaction{&cbtx}, tailBlk, "16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
		b.SetHash(lblock.CalculateHash(b))
		bc.AddBlockContextToTail(PrepareBlockContext(bc, b))
	}
	return bc
}

func MockTransaction() *Transaction {
	return &Transaction{
		ID:       util.GenerateRandomAoB(1),
		Vin:      MockTxInputs(),
		Vout:     MockTxOutputs(),
		Tip:      common.NewAmount(5),
		GasLimit: common.NewAmount(0),
		GasPrice: common.NewAmount(0),
	}
}

func MockTxInputs() []TXInput {
	return []TXInput{
		{util.GenerateRandomAoB(2),
			6,
			util.GenerateRandomAoB(2),
			[]byte("12345678901234567890123456789013")},
		{util.GenerateRandomAoB(2),
			2,
			util.GenerateRandomAoB(2),
			[]byte("12345678901234567890123456789014")},
	}
}

func MockTxInputsWithPubkey(pubkey []byte) []TXInput {
	return []TXInput{
		{util.GenerateRandomAoB(2),
			6,
			util.GenerateRandomAoB(2),
			pubkey},
		{util.GenerateRandomAoB(2),
			2,
			util.GenerateRandomAoB(2),
			pubkey},
	}
}

func MockUtxos(inputs []TXInput) []*UTXO {
	utxos := make([]*UTXO, len(inputs))

	for index, input := range inputs {
		pubKeyHash, _ := account.NewUserPubKeyHash(input.PubKey)
		utxos[index] = &UTXO{
			TXOutput: TXOutput{Value: common.NewAmount(10), PubKeyHash: pubKeyHash, Contract: ""},
			Txid:     input.Txid,
			TxIndex:  0,
		}
	}

	return utxos
}

func MockTxOutputs() []TXOutput {
	return []TXOutput{
		{common.NewAmount(5), account.PubKeyHash(util.GenerateRandomAoB(2)), ""},
		{common.NewAmount(7), account.PubKeyHash(util.GenerateRandomAoB(2)), ""},
	}
}
