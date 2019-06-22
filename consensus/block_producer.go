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
	"time"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/contract"
	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

// process defines the procedure to produce a valid block modified from a raw (unhashed/unsigned) block
type process func(ctx *core.BlockContext)

type BlockProducer struct {
	bc          *core.Blockchain
	beneficiary string
	process     process
	idle        bool
}

func NewBlockProducer() *BlockProducer {
	return &BlockProducer{
		bc:          nil,
		beneficiary: "",
		process:     nil,
		idle:        true,
	}
}

// Setup tells the producer to give rewards to beneficiaryAddr and return the new block through newBlockCh
func (bp *BlockProducer) Setup(bc *core.Blockchain, beneficiaryAddr string) {
	bp.bc = bc
	bp.beneficiary = beneficiaryAddr
}

// Beneficiary returns the address which receives rewards
func (bp *BlockProducer) Beneficiary() string {
	return bp.beneficiary
}

// SetProcess tells the producer to follow the given process to produce a valid block
func (bp *BlockProducer) SetProcess(process process) {
	bp.process = process
}

// ProduceBlock produces a block by preparing its raw contents and applying the predefined process to it.
// deadlineInMs = 0 means no deadline
func (bp *BlockProducer) ProduceBlock(deadlineInMs int64) *core.BlockContext {
	logger.Info("BlockProducer: started producing new block...")
	bp.idle = false
	ctx := bp.prepareBlock(deadlineInMs)
	if ctx != nil && bp.process != nil {
		bp.process(ctx)
	}
	return ctx
}

func (bp *BlockProducer) BlockProduceFinish() {
	bp.idle = true
}

func (bp *BlockProducer) IsIdle() bool {
	return bp.idle
}

func (bp *BlockProducer) prepareBlock(deadlineInMs int64) *core.BlockContext {
	parentBlock, err := bp.bc.GetTailBlock()
	if err != nil {
		logger.WithError(err).Error("BlockProducer: cannot get the current tail block!")
		return nil
	}

	// Retrieve all valid transactions from tx pool
	utxoIndex := core.NewUTXOIndex(bp.bc.GetUtxoCache())

	validTxs, state := bp.collectTransactions(utxoIndex, parentBlock, deadlineInMs)

	cbtx := bp.calculateTips(validTxs)
	validTxs = append(validTxs, cbtx)
	utxoIndex.UpdateUtxo(cbtx)

	logger.WithFields(logger.Fields{
		"valid_txs": len(validTxs),
	}).Info("BlockProducer: prepared a block.")

	ctx := core.BlockContext{Block: core.NewBlock(validTxs, parentBlock, bp.beneficiary), UtxoIndex: utxoIndex, State: state}
	return &ctx
}

func (bp *BlockProducer) collectTransactions(utxoIndex *core.UTXOIndex, parentBlk *core.Block, deadlineInMs int64) ([]*core.Transaction, *core.ScState) {
	var validTxs []*core.Transaction
	totalSize := 0

	scStorage := core.LoadScStateFromDatabase(bp.bc.GetDb())
	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	var generatedTxs []*core.Transaction
	rewards := make(map[string]string)
	currBlkHeight := parentBlk.GetHeight() + 1

	for totalSize < bp.bc.GetBlockSizeLimit() && bp.bc.GetTxPool().GetNumOfTxInPool() > 0 && !isExceedingDeadline(deadlineInMs) {

		txNode := bp.bc.GetTxPool().PopTransactionWithMostTips(utxoIndex)
		if txNode == nil {
			break
		}
		totalSize += txNode.Size

		ctx := txNode.Value.ToContractTx()
		if ctx != nil {
			generatedTxs = ctx.Execute(*utxoIndex, scStorage, rewards, engine, currBlkHeight, parentBlk)
			validTxs = append(validTxs, generatedTxs...)
			utxoIndex.UpdateUtxoState(generatedTxs)
		}

		validTxs = append(validTxs, txNode.Value)
		utxoIndex.UpdateUtxo(txNode.Value)
	}

	// append reward transaction
	if len(rewards) > 0 {
		rtx := core.NewRewardTx(currBlkHeight, rewards)
		validTxs = append(validTxs, &rtx)
		utxoIndex.UpdateUtxo(&rtx)
	}

	return validTxs, scStorage
}

func (bp *BlockProducer) calculateTips(txs []*core.Transaction) *core.Transaction {
	//calculate tips
	totalTips := common.NewAmount(0)
	for _, tx := range txs {
		totalTips = totalTips.Add(tx.Tip)
	}
	cbtx := core.NewCoinbaseTX(core.NewAddress(bp.beneficiary), "", bp.bc.GetMaxHeight()+1, totalTips)
	return &cbtx
}

//executeSmartContract executes all smart contracts
func (bp *BlockProducer) executeSmartContract(utxoIndex *core.UTXOIndex,
	txs []*core.Transaction, currBlkHeight uint64, parentBlk *core.Block) ([]*core.Transaction, *core.ScState) {
	//start a new smart contract engine

	scStorage := core.LoadScStateFromDatabase(bp.bc.GetDb())
	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	var generatedTXs []*core.Transaction
	rewards := make(map[string]string)

	for _, tx := range txs {
		ctx := tx.ToContractTx()
		if ctx == nil {
			// add utxo from txs into utxoIndex
			utxoIndex.UpdateUtxo(tx)
			continue
		}
		generatedTXs = append(generatedTXs, ctx.Execute(*utxoIndex, scStorage, rewards, engine, currBlkHeight, parentBlk)...)
		// add utxo from txs into utxoIndex
		utxoIndex.UpdateUtxo(tx)
	}

	// append reward transaction
	if len(rewards) > 0 {
		rtx := core.NewRewardTx(currBlkHeight, rewards)
		generatedTXs = append(generatedTXs, &rtx)
	}
	return generatedTXs, scStorage
}

func isExceedingDeadline(deadlineInMs int64) bool {
	return deadlineInMs > 0 && time.Now().UnixNano()/1000000 >= deadlineInMs
}
