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
	"github.com/libp2p/go-libp2p-peer"
)

var maxNonce int64 = math.MaxInt64

const targetBits = int64(14)

type State int

const (
	prepareBlockState   State = iota
	mineBlockState
	updateNewBlockState
	mergeForkState
)

type ProofOfWork struct {
	target    *big.Int
	exitCh    chan bool
	bc        *core.Blockchain
	nextState State
	cbAddr    string
	node      *network.Node
	newBlock  *core.Block
	newBlkRcvd bool
	nonce	   int64
}

func NewProofOfWork() *ProofOfWork{
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	p := &ProofOfWork{
		target: 		target,
		exitCh: 		make(chan bool, 1),
		bc:     		nil,
		nextState:		prepareBlockState,
		cbAddr: 		"",
		node: 			nil,
		newBlock:		nil,
		newBlkRcvd:		false,
		nonce:			0,
	}
	return p
}

func (pow *ProofOfWork) Setup(bc *core.Blockchain, cbAddr string){
	pow.bc = bc
	pow.cbAddr = cbAddr
	pow.node = network.NewNode(bc)
}

func (pow *ProofOfWork) GetNode() *network.Node{
	return pow.node
}

func (pow *ProofOfWork) GetCurrentState() State {
	return pow.nextState
}

func (pow *ProofOfWork) SetTargetBit(bit int){
	target := big.NewInt(1)
	pow.target = target.Lsh(target, uint(256-bit))
}

func (pow *ProofOfWork) Start() {
	go func() {
		logger.Info("PoW started...")
		pow.nextState = prepareBlockState
		for {
			select {
			case rcvedblk := <- pow.bc.BlockPool().BlockReceivedCh():
				pow.handleRcvdBlock(rcvedblk.Block, rcvedblk.Pid)
			case <-pow.exitCh:
				logger.Info("PoW stopped...")
				return
			default:
				pow.runNextState()
			}
		}
	}()
}

func (pow *ProofOfWork) Stop() {
	pow.exitCh <- true
}

func (pow *ProofOfWork) runNextState(){
	switch pow.nextState {
	case prepareBlockState:
		pow.newBlock = pow.prepareBlock()
		pow.nextState = mineBlockState
	case mineBlockState:
		if pow.nonce < maxNonce {
			if ok := pow.mineBlock(); ok {
				pow.nextState = updateNewBlockState
			}
		}else{
			pow.nextState = prepareBlockState
		}
	case updateNewBlockState:
		pow.updateNewBlock()
		pow.nextState = mergeForkState
	case mergeForkState:
		pow.bc.MergeFork()
		pow.nextState = prepareBlockState
	}
}


func (pow *ProofOfWork) handleRcvdBlock(blk *core.Block, sender peer.ID){
	logger.Debug("PoW: Received a new block from peer!")
	if pow.ValidateDifficulty(blk){
		logger.Debug("PoW: Block difficulty has been validated!")
		tailBlock,err := pow.bc.GetTailBlock()
		if err != nil {
			logger.Warn("PoW: Get Tail Block failed! Err:", err)
		}
		if core.IsParentBlock(tailBlock, blk){
			pow.newBlock.Rollback()
			pow.newBlock = blk
			pow.newBlkRcvd = true
			pow.nextState = updateNewBlockState
		}else{
			pow.updateFork(blk, sender)
		}
	}
}

func (pow *ProofOfWork) ValidateDifficulty(blk *core.Block) bool {
	var hashInt big.Int

	hash := blk.GetHash()
	hashInt.SetBytes(hash)

	isValid := hashInt.Cmp(pow.target) == -1

	return isValid
}


func (pow *ProofOfWork) prepareBlock() *core.Block{

	parentBlock,err := pow.bc.GetTailBlock()
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

	pow.nonce = 0
	//prepare the new block (without the correct nonce value)
	return core.NewBlock(txs, parentBlock)
}

//returns true if a block is mined; returns false if the nonce value does not satisfy the difficulty requirement
func (pow *ProofOfWork) mineBlock() bool{
		hash, ok := pow.verifyNonce(pow.nonce, pow.newBlock)
		if ok {
			pow.newBlock.SetHash(hash)
			pow.newBlock.SetNonce(pow.nonce)
		}else{
			pow.nonce ++
		}
		return ok
}

func (pow *ProofOfWork) verifyNonce(nonce int64, blk *core.Block) (core.Hash, bool){
	var hashInt big.Int
	var hash core.Hash

	hash = blk.CalculateHashWithNonce(nonce)
	hashInt.SetBytes(hash[:])

	return hash, hashInt.Cmp(pow.target) == -1
}

func (pow *ProofOfWork) updateNewBlock(){
	pow.bc.UpdateNewBlock(pow.newBlock)
	if !pow.newBlkRcvd {
		logger.Debug("PoW: Minted a new block")
		pow.broadcastNewBlock(pow.newBlock)
	}else{
		logger.Debug("PoW: Add a rcved block to blockchain")
	}
	pow.newBlkRcvd = false
}

func (pow *ProofOfWork) broadcastNewBlock(blk *core.Block){
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

func (pow *ProofOfWork) updateFork(block *core.Block, pid peer.ID){
	if pow.attemptToAddTailToFork(block){return}
	if pow.attemptToAddParentToFork(block, pid){return}
	if pow.attempToStartNewFork(block, pid){return}
}

func (pow *ProofOfWork) attemptToAddTailToFork(newblock *core.Block) bool{
	return pow.bc.BlockPool().AddTailToFork(newblock)
}

//returns true if successful
func (pow *ProofOfWork) attemptToAddParentToFork(newblock *core.Block, sender peer.ID) bool{

	isSuccessful := pow.bc.BlockPool().AddParentToFork(newblock)
	if isSuccessful{
		//if the parent of the current fork is found in blockchain, merge the fork
		if pow.bc.IsInBlockchain(newblock.GetPrevHash()){
			pow.newBlock.Rollback()
			pow.nextState = mergeForkState
		}else{
			//if the fork could not be added to the current blockchain, ask for the head block's parent
			pow.RequestBlock(newblock.GetPrevHash(), sender)
		}
	}
	return isSuccessful
}

func (pow *ProofOfWork) attempToStartNewFork(newblock *core.Block, sender peer.ID) bool{
	startNewFork := pow.bc.BlockPool().IsHigherThanFork(newblock)
	if startNewFork{
		pow.bc.BlockPool().ReInitializeForkPool(newblock)
		pow.RequestBlock(newblock.GetPrevHash(),sender)
	}
	return startNewFork
}

//returns the height of the parent block
func (pow *ProofOfWork) findParentBlockInBlockchain() int{
	//TODO
	return 0
}

func AddForkToBlockchain(bc *core.Blockchain){
	//TODO
}

func (pow *ProofOfWork)checkAndAddForkToBlockchain() bool{
	//If the returned height is equal or larger than 0, it means the parent is found in local blockchain
	if pow.findParentBlockInBlockchain() >= 0 {
		AddForkToBlockchain(pow.bc)
		return true
	}
	return false
}

func (pow *ProofOfWork) RequestBlock(hash core.Hash, pid peer.ID){
	pow.bc.BlockPool().RequestBlock(hash, pid)
}