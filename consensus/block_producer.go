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
	"encoding/hex"
	"time"

	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	"github.com/dappley/go-dappley/logic/transaction_logic"
	"github.com/dappley/go-dappley/logic/utxo_logic"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"

	"github.com/dappley/go-dappley/vm"
	logger "github.com/sirupsen/logrus"
)

// process defines the procedure to produce a valid block modified from a raw (unhashed/unsigned) block
type process func(ctx *blockchain_logic.BlockContext)

type BlockProducerInfo struct {
	bc          *blockchain_logic.Blockchain
	beneficiary string
	process     process
	idle        bool
}

func NewBlockProducerInfo() *BlockProducerInfo {
	return &BlockProducerInfo{
		bc:          nil,
		beneficiary: "",
		process:     nil,
		idle:        true,
	}
}

// Setup tells the producer to give rewards to beneficiaryAddr and return the new block through newBlockCh
func (bp *BlockProducerInfo) Setup(bc *blockchain_logic.Blockchain, beneficiaryAddr string) {
	bp.bc = bc
	bp.beneficiary = beneficiaryAddr
}

// Beneficiary returns the address which receives rewards
func (bp *BlockProducerInfo) Beneficiary() string {
	return bp.beneficiary
}

// SetProcess tells the producer to follow the given process to produce a valid block
func (bp *BlockProducerInfo) SetProcess(process process) {
	bp.process = process
}

// ProduceBlock produces a block by preparing its raw contents and applying the predefined process to it.
// deadlineInMs = 0 means no deadline
func (bp *BlockProducerInfo) ProduceBlock(deadlineInMs int64) *blockchain_logic.BlockContext {
	logger.Info("BlockProducerInfo: started producing new block...")
	bp.idle = false
	ctx := bp.prepareBlock(deadlineInMs)
	if ctx != nil && bp.process != nil {
		bp.process(ctx)
	}
	return ctx
}

func (bp *BlockProducerInfo) BlockProduceFinish() {
	bp.idle = true
}

func (bp *BlockProducerInfo) IsIdle() bool {
	return bp.idle
}

func (bp *BlockProducerInfo) prepareBlock(deadlineInMs int64) *blockchain_logic.BlockContext {
	parentBlock, err := bp.bc.GetTailBlock()
	if err != nil {
		logger.WithError(err).Error("BlockProducerInfo: cannot get the current tail block!")
		return nil
	}

	// Retrieve all valid transactions from tx pool
	utxoIndex := utxo_logic.NewUTXOIndex(bp.bc.GetUtxoCache())

	validTxs, state := bp.collectTransactions(utxoIndex, parentBlock, deadlineInMs)

	cbtx := bp.calculateTips(validTxs)
	validTxs = append(validTxs, cbtx)
	utxoIndex.UpdateUtxo(cbtx)

	logger.WithFields(logger.Fields{
		"valid_txs": len(validTxs),
	}).Info("BlockProducerInfo: prepared a block.")

	ctx := blockchain_logic.BlockContext{Block: block.NewBlock(validTxs, parentBlock, bp.beneficiary), UtxoIndex: utxoIndex, State: state}
	return &ctx
}

func (bp *BlockProducerInfo) collectTransactions(utxoIndex *utxo_logic.UTXOIndex, parentBlk *block.Block, deadlineInMs int64) ([]*transaction.Transaction, *scState.ScState) {
	var validTxs []*transaction.Transaction
	totalSize := 0

	scStorage := scState.LoadScStateFromDatabase(bp.bc.GetDb())
	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	rewards := make(map[string]string)
	currBlkHeight := parentBlk.GetHeight() + 1

	for totalSize < bp.bc.GetBlockSizeLimit() && bp.bc.GetTxPool().GetNumOfTxInPool() > 0 && !isExceedingDeadline(deadlineInMs) {

		txNode := bp.bc.GetTxPool().PopTransactionWithMostTips(utxoIndex)
		if txNode == nil {
			break
		}
		totalSize += txNode.Size

		ctx := txNode.Value.ToContractTx()
		minerAddr := account.NewAddress(bp.beneficiary)
		if ctx != nil {
			prevUtxos, err := utxo_logic.FindVinUtxosInUtxoPool(*utxoIndex, ctx.Transaction)
			if err != nil {
				logger.WithError(err).WithFields(logger.Fields{
					"txid": hex.EncodeToString(ctx.ID),
				}).Warn("Transaction: cannot find vin while executing smart contract")
				return nil, nil
			}
			isSCUTXO := (*utxoIndex).GetAllUTXOsByPubKeyHash([]byte(ctx.Vout[0].PubKeyHash)).Size() == 0

			validTxs = append(validTxs, txNode.Value)
			utxoIndex.UpdateUtxo(txNode.Value)

			gasCount, generatedTxs, err := transaction_logic.Execute(ctx, prevUtxos, isSCUTXO, *utxoIndex, scStorage, rewards, engine, currBlkHeight, parentBlk)

			// record gas used
			if err != nil {
				// add utxo from txs into utxoIndex
				logger.WithError(err).Error("executeSmartContract error.")
			}
			if gasCount > 0 {
				grtx, err := transaction.NewGasRewardTx(minerAddr, currBlkHeight, common.NewAmount(gasCount), ctx.GasPrice)
				if err == nil {
					generatedTxs = append(generatedTxs, &grtx)
				}
			}
			gctx, err := transaction.NewGasChangeTx(ctx.GetDefaultFromPubKeyHash().GenerateAddress(), currBlkHeight, common.NewAmount(gasCount), ctx.GasLimit, ctx.GasPrice)
			if err == nil {
				generatedTxs = append(generatedTxs, &gctx)
			}
			validTxs = append(validTxs, generatedTxs...)
			utxoIndex.UpdateUtxoState(generatedTxs)
		} else {
			validTxs = append(validTxs, txNode.Value)
			utxoIndex.UpdateUtxo(txNode.Value)
		}
	}

	// append reward transaction
	if len(rewards) > 0 {
		rtx := transaction.NewRewardTx(currBlkHeight, rewards)
		validTxs = append(validTxs, &rtx)
		utxoIndex.UpdateUtxo(&rtx)
	}

	return validTxs, scStorage
}

func (bp *BlockProducerInfo) calculateTips(txs []*transaction.Transaction) *transaction.Transaction {
	//calculate tips
	totalTips := common.NewAmount(0)
	for _, tx := range txs {
		totalTips = totalTips.Add(tx.Tip)
	}
	cbtx := transaction_logic.NewCoinbaseTX(account.NewAddress(bp.beneficiary), "", bp.bc.GetMaxHeight()+1, totalTips)
	return &cbtx
}

//executeSmartContract executes all smart contracts
func (bp *BlockProducerInfo) executeSmartContract(utxoIndex *utxo_logic.UTXOIndex,
	txs []*transaction.Transaction, currBlkHeight uint64, parentBlk *block.Block) ([]*transaction.Transaction, *scState.ScState) {
	//start a new smart contract engine

	scStorage := scState.LoadScStateFromDatabase(bp.bc.GetDb())
	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	var generatedTXs []*transaction.Transaction
	rewards := make(map[string]string)

	minerAddr := account.NewAddress(bp.beneficiary)

	for _, tx := range txs {
		ctx := tx.ToContractTx()
		if ctx == nil {
			// add utxo from txs into utxoIndex
			utxoIndex.UpdateUtxo(tx)
			continue
		}
		prevUtxos, err := utxo_logic.FindVinUtxosInUtxoPool(*utxoIndex, ctx.Transaction)
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"txid": hex.EncodeToString(ctx.ID),
			}).Warn("Transaction: cannot find vin while executing smart contract")
			return nil, nil
		}
		isSCUTXO := (*utxoIndex).GetAllUTXOsByPubKeyHash([]byte(ctx.Vout[0].PubKeyHash)).Size() == 0
		gasCount, newTxs, err := transaction_logic.Execute(ctx, prevUtxos, isSCUTXO, *utxoIndex, scStorage, rewards, engine, currBlkHeight, parentBlk)
		generatedTXs = append(generatedTXs, newTxs...)
		// record gas used
		if err != nil {
			// add utxo from txs into utxoIndex
			logger.WithFields(logger.Fields{
				"err": err,
			}).Error("executeSmartContract error.")
		}
		if gasCount > 0 {
			grtx, err := transaction.NewGasRewardTx(minerAddr, currBlkHeight, common.NewAmount(gasCount), ctx.GasPrice)
			if err == nil {
				generatedTXs = append(generatedTXs, &grtx)
			}
		}
		gctx, err := transaction.NewGasChangeTx(ctx.GetDefaultFromPubKeyHash().GenerateAddress(), currBlkHeight, common.NewAmount(gasCount), ctx.GasLimit, ctx.GasPrice)
		if err == nil {

			generatedTXs = append(generatedTXs, &gctx)
		}

		// add utxo from txs into utxoIndex
		utxoIndex.UpdateUtxo(tx)
	}
	// append reward transaction
	if len(rewards) > 0 {
		rtx := transaction.NewRewardTx(currBlkHeight, rewards)
		generatedTXs = append(generatedTXs, &rtx)
	}
	return generatedTXs, scStorage
}

func isExceedingDeadline(deadlineInMs int64) bool {
	return deadlineInMs > 0 && time.Now().UnixNano()/1000000 >= deadlineInMs
}

func (bp *BlockProducerInfo) Produced(blk *block.Block) bool {
	if blk != nil {
		return bp.beneficiary == blk.GetProducer()
	}
	return false
}
