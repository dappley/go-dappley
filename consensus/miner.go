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
package consensus

import (
	"github.com/dappley/go-dappley/core"
	"container/heap"
)

type state int

const (
	prepareTxPoolState state = iota
	mineState
	updateNewBlock
	cleanUpState
)

type Miner struct {
	bc           	*core.Blockchain
	newBlock     	*core.Block
	coinBaseAddr 	string
	nextState    	state
	consensus		core.Consensus
}

//create a new instance
func NewMiner(bc *core.Blockchain, coinBaseAddr string, consensus core.Consensus) *Miner {

	return &Miner{
		bc,
		nil,
		coinBaseAddr,
		prepareTxPoolState,
		consensus,
	}
}

//start mining
func (miner *Miner) Start() {
	miner.run()
}

func UpdateTxPool(txs core.TransactionPool) {
	core.ModifyTxnPoolInstance(&txs)
}

//start the state machine
func (miner *Miner) run() {

Loop:
	for {
		switch miner.nextState {
		case prepareTxPoolState:
			miner.prepareTxPool()
			miner.nextState = mineState

		case mineState:
			miner.mine()
			miner.nextState = updateNewBlock
		case updateNewBlock:
			miner.updateNewBlock()
			miner.nextState = cleanUpState
		case cleanUpState:
			miner.cleanUp()
			break Loop
		}
	}
}

//prepare transaction pool
func (miner *Miner) prepareTxPool() {
	// verify all transactions
	miner.verifyTransactions()
	// add coinbase transaction
	cbtx := core.NewCoinbaseTX(miner.coinBaseAddr, "")
	h := core.GetTxnPoolInstance()
	heap.Init(h)
	heap.Push(core.GetTxnPoolInstance(), cbtx)

}

//start proof of work process
func (miner *Miner) mine() {

	//get the hash of last newBlock
	lastHash, err := miner.bc.GetLastHash()
	if err != nil {
		//TODO
	}

	//create a new newBlock with the transaction pool and last hash
	miner.consensus = NewProofOfWork(miner.coinBaseAddr)
	miner.newBlock = miner.consensus.ProduceBlock(lastHash)
}

//update the blockchain with the new block
func (miner *Miner) updateNewBlock() {
	miner.bc.UpdateNewBlock(miner.newBlock)
}

func (miner *Miner) cleanUp() {
	miner.nextState = prepareTxPoolState
}

//verify transactions and remove invalid transactions
func (miner *Miner) verifyTransactions() {
	//for TransactionPool.Len() > 0 {
	//
	//	var txn = heap.Pop(&TransactionPool).(core.Transaction)
	//
	//	//if miner.bc.VerifyTransaction(txn) != true {
	//	//	//Remove transaction from transaction pool if the transaction is not verified
	//	//	miner.txPool = append(miner.txPool[0:i],miner.txPool[i+1:len(miner.txPool)]...)
	//	//}
	//}
	//}
}
