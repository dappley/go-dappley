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
func (pd *Miner) Start() {
	pd.run()
}

func UpdateTxPool(txs core.TransactionPool) {
	core.TransactionPoolSingleton = txs
}

//start the state machine
func (pd *Miner) run() {

Loop:
	for {
		switch pd.nextState {
		case prepareTxPoolState:
			pd.prepareTxPool()
			pd.nextState = mineState

		case mineState:
			pd.mine()
			pd.nextState = updateNewBlock
		case updateNewBlock:
			pd.updateNewBlock()
			pd.nextState = cleanUpState
		case cleanUpState:
			pd.cleanUp()
			break Loop
		}
	}
}

//prepare transaction pool
func (pd *Miner) prepareTxPool() {
	// verify all transactions
	pd.verifyTransactions()
}

//start proof of work process
func (pd *Miner) mine() {

	//get the hash of last newBlock
	lastHash, err := pd.bc.GetLastHash()
	if err != nil {
		//TODO
	}

	//create a new newBlock with the transaction pool and last hash
	pd.consensus = NewProofOfWork(pd.coinBaseAddr)
	pd.newBlock = pd.consensus.ProduceBlock(lastHash)
}

//update the blockchain with the new block
func (pd *Miner) updateNewBlock() {
	pd.bc.UpdateNewBlock(pd.newBlock)
}

func (pd *Miner) cleanUp() {
	pd.nextState = prepareTxPoolState
}

//verify transactions and remove invalid transactions
func (pd *Miner) verifyTransactions() {
	//for TransactionPool.Len() > 0 {
	//
	//	var txn = heap.Pop(&TransactionPool).(core.Transaction)
	//
	//	//if pd.bc.VerifyTransaction(txn) != true {
	//	//	//Remove transaction from transaction pool if the transaction is not verified
	//	//	pd.txPool = append(pd.txPool[0:i],pd.txPool[i+1:len(pd.txPool)]...)
	//	//}
	//}
	//}
}
