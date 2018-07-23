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
	"math"
	"math/big"

	"container/heap"

	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
	"github.com/dappley/go-dappley/network"
	"fmt"
	"reflect"
)

var maxNonce int64 = math.MaxInt64

const targetBits = int64(14)

const (
	prepareBlockState state = iota
	mineBlockState
	updateNewBlockState
)

type ProofOfWork struct {
	target 			*big.Int
	exitCh           chan bool
	bc               *core.Blockchain
	nextState        state
	cbAddr			 string
	node 			*network.Node
}

func NewProofOfWork(bc *core.Blockchain, cbAddr string) *ProofOfWork{
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	p := &ProofOfWork{
		target: 		target,
		exitCh: 		make(chan bool, 1),
		bc:     		bc,
		nextState:		prepareBlockState,
		cbAddr: 		cbAddr,
		node: 			network.NewNode(bc),
	}
	return p
}

func (pow *ProofOfWork) SetTargetBit(bit int){
	target := big.NewInt(1)
	pow.target = target.Lsh(target, uint(256-bit))
}

func (pow *ProofOfWork) ValidateDifficulty(blk *core.Block) bool {
	var hashInt big.Int

	hash := blk.GetHash()
	hashInt.SetBytes(hash)

	isValid := hashInt.Cmp(pow.target) == -1

	return isValid
}

func (pow *ProofOfWork) Stop() {
	pow.exitCh <- true
}

func (pow *ProofOfWork) Start() {
	go func() {
		var newBlock *core.Block
		nonce := int64(0)
		logger.Info("PoW started...")
		pow.nextState = prepareBlockState
		for {
			select {
			case blk := <- pow.bc.BlockPool().BlockUpdateCh():
				logger.Debug("PoW: Received a block from peer")
				if pow.ValidateDifficulty(blk){
					logger.Debug("PoW: The block has been verified")
					pow.rollbackBlock(newBlock)
					newBlock = blk
					a := newBlock.GetTransactions()[0].Vin[0].Txid

					if reflect.DeepEqual(a, []uint8{}){
						fmt.Println("Blk txid:", newBlock.GetTransactions()[0].Vin[0].Txid)
					}

					pow.nextState = updateNewBlockState
				}
			case <-pow.exitCh:
				logger.Info("PoW stopped...")
				return
			default:
				switch pow.nextState {
				case prepareBlockState:
					logger.Debug("Pow State: prepareBlockState")
					newBlock = pow.prepareBlock()
					nonce = 0
					pow.nextState = mineBlockState
				case mineBlockState:
					if nonce < maxNonce {
						if hash, ok := pow.verifyNonce(nonce, newBlock); ok {
							newBlock.SetHash(hash)
							newBlock.SetNonce(nonce)
							pow.nextState = updateNewBlockState
						} else {
							nonce++
							pow.nextState = mineBlockState
						}
					}else{
						pow.nextState = prepareBlockState
					}
				case updateNewBlockState:
					logger.Debug("Pow State: updateNewBlockState")
					pow.updateNewBlock(newBlock)
					pow.nextState = prepareBlockState
				}
			}
		}
	}()
}

func (pow *ProofOfWork) GetCurrentState() state{
	return pow.nextState
}

func (pow *ProofOfWork) prepareBlock() *core.Block{

	parentBlock,err := pow.bc.GetLastBlock()
	if err!=nil {
		logger.Error(err)
	}

	//verify all transactions
	pow.verifyTransactions()
	//get all transactions
	txs := core.GetTxnPoolInstance().GetSortedTransactions()
	//add coinbase transaction to transaction pool
	cbtx := core.NewCoinbaseTX(pow.cbAddr, "")
	txs = append(txs, &cbtx)

	//prepare the new block (without the correct nonce value)
	return core.NewBlock(txs, parentBlock)
}

func (pow *ProofOfWork) verifyNonce(nonce int64, blk *core.Block) (core.Hash, bool){
	var hashInt big.Int
	var hash core.Hash

	hash = blk.CalculateHashWithNonce(nonce)
	hashInt.SetBytes(hash[:])

	return hash, hashInt.Cmp(pow.target) == -1
}

func (pow *ProofOfWork) updateNewBlock(blk *core.Block){
	pow.bc.UpdateNewBlock(blk)
	//broadcast the block to other nodes
	pow.node.SendBlock(blk)
}

//verify transactions and remove invalid transactions
func (pow *ProofOfWork) verifyTransactions() {
	txnPool := core.GetTxnPoolInstance()
	txnPoolLength := txnPool.Len()
	for i := 0; i < txnPoolLength; i++ {
		var txn = heap.Pop(txnPool).(core.Transaction)
		if pow.bc.VerifyTransaction(txn) == true {
			//Remove transaction from transaction pool if the transaction is not verified
			txnPool.Push(txn)
		}
	}
}

//When a block mining process is interrupted, roll back the block and
//return all transactions to the transaction pool
func (pow *ProofOfWork) rollbackBlock(blk *core.Block){
	txnPool := core.GetTxnPoolInstance()
	for _,tx := range blk.GetTransactions(){
		if !tx.IsCoinbase() {
			txnPool.Push(tx)
		}
	}
}