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
	"math/big"

	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/core"
)

type BlockProducer struct {
	exitCh   chan bool
	bc       *core.Blockchain
	cbAddr   string
	key      string
	newBlock *MinedBlock
	retChan  chan *MinedBlock
	stop     bool
}

func NewBlockProducer() *BlockProducer {
	return &BlockProducer{
		exitCh:   make(chan bool, 1),
		bc:       nil,
		cbAddr:   "",
		newBlock: &MinedBlock{nil, false},
		stop:     true,
	}
}

func (bp *BlockProducer) SetPrivKey(key string) {
	bp.key = key
}

func (bp *BlockProducer) GetPrivKey() string {
	return bp.key
}

func (bp *BlockProducer) Setup(bc *core.Blockchain, cbAddr string, retChan chan *MinedBlock) {
	bp.bc = bc
	bp.cbAddr = cbAddr
	bp.retChan = retChan
}

func (bp *BlockProducer) Start() {
	go func() {
		if bp.bc.GetBlockPool().GetSyncState() {
			return
		}
		logger.Info("BlockProducer: Producing a block...")
		bp.resetExitCh()
		bp.prepare()
		bp.stop = false
		select {
		case <-bp.exitCh:
			logger.Warn("BlockProducer: Block production is interrupted")
		default:
			bp.produceBlock()
		}
		bp.stop = true
		bp.returnBlk()
		logger.Info("BlockProducer: Produced a block")
	}()
}

func (bp *BlockProducer) Stop() {
	if len(bp.exitCh) == 0 {
		bp.exitCh <- true
	}
}

func (bp *BlockProducer) Validate(blk *core.Block) bool {
	var hashInt big.Int

	hash := blk.GetHash()
	hashInt.SetBytes(hash)

	//isValid := hashInt.Cmp(bp.target) == -1

	return true
}

func (bp *BlockProducer) prepare() {
	bp.newBlock = bp.prepareBlock()
}

func (bp *BlockProducer) returnBlk() {
	if !bp.newBlock.isValid {
		bp.newBlock.block.Rollback(bp.bc.GetTxPool())
	}
	bp.retChan <- bp.newBlock
}

func (bp *BlockProducer) resetExitCh() {
	if len(bp.exitCh) > 0 {
		<-bp.exitCh
	}
}

func (bp *BlockProducer) prepareBlock() *MinedBlock {

	parentBlock, err := bp.bc.GetTailBlock()
	if err != nil {
		logger.Error(err)
	}

	//verify all transactions
	bp.verifyTransactions()
	//get all transactions
	txs := bp.bc.GetTxPool().Pop()
	//add coinbase transaction to transaction pool
	cbtx := core.NewCoinbaseTX(bp.cbAddr, "", bp.bc.GetMaxHeight()+1)
	txs = append(txs, &cbtx)
	// TODO: add tips to txs

	//prepare the new block
	return &MinedBlock{core.NewBlock(txs, parentBlock), false}
}

// produceBlock hashes and signs the new block; returns true if it was successful
func (bp *BlockProducer) produceBlock() bool {
	hash := bp.newBlock.block.CalculateHashWithoutNonce()
	bp.newBlock.block.SetHash(hash)
	bp.newBlock.block.SetNonce(0)
	keyString := bp.GetPrivKey()
	if len(keyString) > 0 {
		signed := bp.newBlock.block.SignBlock(keyString, hash)
		if !signed {
			logger.Warn("Miner Key= ", keyString)
			return false
		}
	}
	bp.newBlock.isValid = true
	return true
}

// verifyTransactions removes invalid transactions from transaction pool
func (bp *BlockProducer) verifyTransactions() {
	utxoPool := core.LoadUTXOIndex(bp.bc.GetDb())
	txPool := bp.bc.GetTxPool()
	txPool.RemoveInvalidTransactions(utxoPool)
}
