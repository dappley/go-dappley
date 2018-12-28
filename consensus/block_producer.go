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
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/contract"
	"github.com/dappley/go-dappley/core"
)

// process defines the procedure to produce a valid block modified from a raw (unhashed/unsigned) block
type process func(block *core.Block)

type BlockProducer struct {
	bc          *core.Blockchain
	beneficiary string
	newBlock    *core.Block
	process     process
	idle        bool
}

func NewBlockProducer() *BlockProducer {
	return &BlockProducer{
		bc:          nil,
		beneficiary: "",
		process:     nil,
		newBlock:    nil,
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
func (bp *BlockProducer) ProduceBlock() *core.Block {

	bp.idle = false
	bp.prepareBlock()
	if bp.process != nil {
		bp.process(bp.newBlock)
	}
	return bp.newBlock
}

func (bp *BlockProducer) BlockProduceFinish() {
	bp.idle = true
}

func (bp *BlockProducer) IsIdle() bool {
	return bp.idle
}

func (bp *BlockProducer) prepareBlock() {
	parentBlock, err := bp.bc.GetTailBlock()
	if err != nil {
		logger.WithError(err).Error("BlockProducer: cannot get the current tail block!")
		return
	}

	// Retrieve all valid transactions from tx pool
	utxoIndex := core.LoadUTXOIndex(bp.bc.GetDb())

	txs := bp.bc.GetTxPool().GetAndResetTransactions()
	validTxs := filterValidTxs(txs, utxoIndex, parentBlock.GetHeight()+1)

	cbtx := bp.calculateTips(validTxs)
	rewards := make(map[string]string)
	scGeneratedTXs := bp.executeSmartContract(validTxs, rewards, parentBlock.GetHeight()+1, parentBlock)
	validTxs = append(validTxs, scGeneratedTXs...)
	validTxs = append(validTxs, cbtx)
	if len(rewards) > 0 {
		rtx := core.NewRewardTx(parentBlock.GetHeight()+1, rewards)
		validTxs = append(validTxs, &rtx)
	}

	bp.newBlock = core.NewBlock(validTxs, parentBlock)
}

func filterValidTxs(txs []*core.Transaction, utxoIndex *core.UTXOIndex, blockHeight uint64) []*core.Transaction {
	tempUtxoIndex := utxoIndex.DeepCopy()
	var validTxs []*core.Transaction

	for _, tx := range txs {
		if tx.Verify(tempUtxoIndex, blockHeight) {
			validTxs = append(validTxs, tx)
		}
		tempUtxoIndex.UpdateUtxo(tx)
	}

	return validTxs
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
func (bp *BlockProducer) executeSmartContract(txs []*core.Transaction, rewards map[string]string, currBlkHeight uint64, parentBlk *core.Block) []*core.Transaction {
	//start a new smart contract engine
	utxoIndex := core.LoadUTXOIndex(bp.bc.GetDb())
	scStorage := core.NewScState()
	scStorage.LoadFromDatabase(bp.bc.GetDb(), bp.bc.GetTailBlockHash())
	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	var generatedTXs []*core.Transaction
	for _, tx := range txs {
		generatedTXs = append(generatedTXs, tx.Execute(*utxoIndex, scStorage, rewards, engine, currBlkHeight, parentBlk)...)
	}
	return generatedTXs
}
