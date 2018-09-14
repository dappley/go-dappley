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
	"github.com/dappley/go-dappley/crypto/byteutils"
	"github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p-peer"
	logger "github.com/sirupsen/logrus"
)

type BlockRequestPars struct {
	BlockHash Hash
	Pid       peer.ID
}

type RcvedBlock struct {
	Block *Block
	Pid   peer.ID
}

type BlockPool struct {
	blockRequestCh chan BlockRequestPars
	size           int
	bc             *Blockchain
	forkPool       []*Block
	cache          *lru.Cache
}

type linkedBlock struct {
	block      *Block
	chain      *Blockchain
	hash       byteutils.Hash
	parentHash byteutils.Hash

	parentBlock *linkedBlock
	childBlocks map[byteutils.HexHash]*linkedBlock
}

func NewBlockPool(size int) *BlockPool {
	pool := &BlockPool{
		size:           size,
		blockRequestCh: make(chan BlockRequestPars, size),
		bc:             nil,
		forkPool:       []*Block{},
	}
	pool.cache, _ = lru.NewWithEvict(size, func(key interface{}, value interface{}) {
		treenode := value.(*Block)
		if treenode != nil {
			//remove treenode
		}
	})
	return pool
}

func (pool *BlockPool) SetBlockchain(bc *Blockchain) {
	pool.bc = bc
}

func (pool *BlockPool) BlockRequestCh() chan BlockRequestPars {
	return pool.blockRequestCh
}

func (pool *BlockPool) GetForkPool() []*Block { return pool.forkPool }

func (pool *BlockPool) ForkPoolLen() int {
	return len(pool.forkPool)
}

func (pool *BlockPool) GetForkPoolHeadBlk() *Block {
	if len(pool.forkPool) > 0 {
		return pool.forkPool[len(pool.forkPool)-1]
	}
	return nil
}

func (pool *BlockPool) GetForkPoolTailBlk() *Block {
	if len(pool.forkPool) > 0 {
		return pool.forkPool[0]
	}
	return nil
}

func (pool *BlockPool) ResetForkPool() {
	pool.forkPool = []*Block{}
}

func (pool *BlockPool) ReInitializeForkPool(blk *Block) {
	logger.Debug("Fork: Re-initilaize fork with the new block")
	pool.ResetForkPool()
	pool.forkPool = append(pool.forkPool, blk)
}

func (pool *BlockPool) IsParentOfFork(blk *Block) bool {
	if blk == nil || pool.ForkPoolLen() == 0 {
		return false
	}

	return IsParentBlock(blk, pool.GetForkPoolHeadBlk())
}

func (pool *BlockPool) IsTailOfFork(blk *Block) bool {
	if blk == nil || pool.ForkPoolLen() == 0 {
		return false
	}

	return IsParentBlock(pool.GetForkPoolTailBlk(), blk)
}

func (pool *BlockPool) GetBlockchain() *Blockchain {
	return pool.bc
}

//Verify all transactions in a fork
func (pool *BlockPool) VerifyTransactions(utxo UTXOIndex) bool {
	for i := pool.ForkPoolLen() - 1; i >= 0; i-- {
		logger.Info("Start Verify")
		if !pool.forkPool[i].VerifyTransactions(utxo) {
			return false
		}
		logger.Info("Verifyed a block. Height: ", i, "Have ", i, "block left")
		utxoIndex := LoadUTXOIndex(pool.bc.GetDb())
		utxoIndex.Update(pool.forkPool[i], pool.bc.GetDb())
	}
	return true
}

func (pool *BlockPool) updateForkFromTail(blk *Block) bool {

	isTail := pool.IsTailOfFork(blk)
	if isTail {
		//only update if the block is higher than the current blockchain
		if pool.bc.IsHigherThanBlockchain(blk) {
			logger.Debug("BlockPool: Add block to tail")
			pool.addTailToForkPool(blk)
		} else {
			//if the fork's max height is less than the blockchain, delete the fork
			logger.Debug("BlockPool: Fork height too low. Dump the fork...")
			pool.ResetForkPool()
		}
	}
	return isTail
}

//returns if the operation is successful
func (pool *BlockPool) addParentToFork(blk *Block) bool {

	isParent := pool.IsParentOfFork(blk)
	if isParent {
		//check if fork's max height is still higher than the blockchain
		if pool.GetForkPoolTailBlk().GetHeight() > pool.bc.GetMaxHeight() {
			logger.Debug("BlockPool: Add block to head")
			pool.addParentToForkPool(blk)
		} else {
			//if the fork's max height is less than the blockchain, delete the fork
			logger.Debug("BlockPool: Fork height too low. Dump the fork...")
			pool.ResetForkPool()
		}
	}
	return isParent
}

func (pool *BlockPool) IsHigherThanFork(block *Block) bool {
	if block == nil {
		return false
	}
	tailBlk := pool.GetForkPoolTailBlk()
	if tailBlk == nil {
		return true
	}
	return block.GetHeight() > tailBlk.GetHeight()
}

func (pool *BlockPool) Push(block *Block, pid peer.ID) {
	logger.Debug("BlockPool: Has received a new block")

	if !block.VerifyHash() {
		logger.Info("BlockPool: Verify Hash failed!")
		return
	}

	//TODO: Verify double spending transactions in the same block

	logger.Debug("BlockPool: Block has been verified")
	pool.handleRcvdBlock(block, pid)
}

func (pool *BlockPool) handleRcvdBlock(blk *Block, sender peer.ID) {

	logger.Debug("BlockPool: Received a new block.Sender id:", sender.String())
	if pool.bc.GetConsensus().Validate(blk) {
		tailBlock, err := pool.bc.GetTailBlock()
		if err != nil {
			logger.Warn("BlockPool: Get Tail Block failed! Err:", err)
		}
		if IsParentBlock(tailBlock, blk) {
			logger.Info("BlockPool: Add received block to blockchain. Sender id:", sender.String())
			pool.bc.GetConsensus().StartNewBlockMinting()
			pool.bc.AddBlockToTail(blk)
			if IsParentBlock(blk, pool.GetForkPoolHeadBlk()) {
				logger.Info("BlockPool: Start merge process")
				pool.bc.MergeFork()
			}
			//TODO: Might want to relay the block to other nodes
		} else {
			pool.updateFork(blk, sender)
		}
	} else {
		logger.Warn("BlockPool: Consensus validity check fails.Sender id:", sender.String())
	}
}

func (pool *BlockPool) updateFork(block *Block, pid peer.ID) {

	if pool.attemptToAddTailToFork(block) {
		return
	}
	if pool.attemptToAddParentToFork(block, pid) {
		return
	}
	if pool.attemptToStartNewFork(block, pid) {
		return
	}
	logger.Debug("BlockPool: Block dumped")
}

func (pool *BlockPool) attemptToAddTailToFork(newblock *Block) bool {
	return pool.updateForkFromTail(newblock)
}

//returns true if successful
func (pool *BlockPool) attemptToAddParentToFork(newblock *Block, sender peer.ID) bool {

	isSuccessful := pool.addParentToFork(newblock)
	if isSuccessful {
		//if the parent of the current fork is found in blockchain, merge the fork
		if pool.bc.IsInBlockchain(newblock.GetPrevHash()) {
			pool.bc.GetConsensus().StartNewBlockMinting()
			pool.bc.MergeFork()
		} else {
			//if the fork could not be added to the current blockchain, ask for the head block's parent
			pool.requestBlock(newblock.GetPrevHash(), sender)
		}
	}
	return isSuccessful
}

func (pool *BlockPool) attemptToStartNewFork(newblock *Block, sender peer.ID) bool {
	startNewFork := pool.IsHigherThanFork(newblock) &&
		pool.bc.IsHigherThanBlockchain(newblock)
	if startNewFork {
		pool.ReInitializeForkPool(newblock)
		pool.requestBlock(newblock.GetPrevHash(), sender)
	}
	return startNewFork
}

//TODO: RequestChannel should be in PoW.go
func (pool *BlockPool) requestBlock(hash Hash, pid peer.ID) {
	pool.blockRequestCh <- BlockRequestPars{hash, pid}
}

func (pool *BlockPool) addTailToForkPool(blk *Block) {
	pool.forkPool = append([]*Block{blk}, pool.forkPool...)
}

func (pool *BlockPool) addParentToForkPool(blk *Block) {
	pool.forkPool = append(pool.forkPool, blk)
}
