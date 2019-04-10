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

// ProduceBlock produces a block by preparing its raw contents and applying the predefined process to it
func (bp *BlockProducer) ProduceBlock() *core.BlockContext {
	logger.Info("BlockProducer: started producing new block...")
	bp.idle = false
	ctx := bp.prepareBlock()
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

func (bp *BlockProducer) prepareBlock() *core.BlockContext {
	parentBlock, err := bp.bc.GetTailBlock()
	if err != nil {
		logger.WithError(err).Error("BlockProducer: cannot get the current tail block!")
		return nil
	}

	// Retrieve all valid transactions from tx pool
	utxoIndex := core.NewUTXOIndex(bp.bc.GetUtxoCache())
	validTxs := bp.bc.GetTxPool().PopTransactionsWithMostTips(utxoIndex, bp.bc.GetBlockSizeLimit())

	cbtx := bp.calculateTips(validTxs)
	rewards := make(map[string]string)

	scGeneratedTXs, state := bp.executeSmartContract(utxoIndex, validTxs, rewards, parentBlock.GetHeight()+1, parentBlock)
	validTxs = append(validTxs, scGeneratedTXs...)

	utxoIndex.UpdateUtxo(cbtx)
	validTxs = append(validTxs, cbtx)
	if len(rewards) > 0 {
		rtx := core.NewRewardTx(parentBlock.GetHeight()+1, rewards)
		utxoIndex.UpdateUtxo(&rtx)
		validTxs = append(validTxs, &rtx)
	}

	logger.WithFields(logger.Fields{
		"valid_txs": len(validTxs),
	}).Info("BlockProducer: prepared a block.")

	ctx := core.BlockContext{Block: core.NewBlock(validTxs, parentBlock), UtxoIndex: utxoIndex, State: state}
	return &ctx
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
	txs []*core.Transaction, rewards map[string]string,
	currBlkHeight uint64, parentBlk *core.Block) ([]*core.Transaction, *core.ScState) {
	//start a new smart contract engine

	scStorage := core.LoadScStateFromDatabase(bp.bc.GetDb())
	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	var generatedTXs []*core.Transaction

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

	return generatedTXs, scStorage
}