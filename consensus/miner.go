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

	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
)

type State int

var maxNonce int64 = math.MaxInt64

type Miner struct {
	exitCh      chan bool
	bc          *core.Blockchain
	beneficiary string
	key         string
	newBlock    *MinedBlock
	nonce       int64
	requirement Requirement
	retChan     chan *MinedBlock
}

func NewMiner() *Miner {
	m := &Miner{
		exitCh:      make(chan bool, 1),
		bc:          nil,
		beneficiary: "",
		newBlock:    &MinedBlock{nil, false},
		nonce:       0,
		requirement: noRequirement,
	}
	return m
}

func (miner *Miner) SetPrivKey(key string) {
	miner.key = key
}

func (miner *Miner) GetPrivKey() string {
	return miner.key
}

func (miner *Miner) Beneficiary() string {
	return miner.beneficiary
}

func (miner *Miner) SetRequirement(requirement Requirement) {
	miner.requirement = requirement
}

func (miner *Miner) Setup(bc *core.Blockchain, beneficiaryAddr string, retChan chan *MinedBlock) {
	miner.bc = bc
	miner.beneficiary = beneficiaryAddr
	miner.retChan = retChan
}

func (miner *Miner) Start() {
	go func() {
		if miner.bc.GetBlockPool().GetSyncState() {
			return
		}
		logger.Info("Miner: Start Mining A Block...")
		miner.resetExitCh()
		miner.prepare()
		nonce := int64(0)
	hashLoop:
		for {
			select {
			case <-miner.exitCh:
				break hashLoop
			default:
				if nonce < maxNonce {
					if ok := miner.mineBlock(nonce); ok {
						break hashLoop
					} else {
						nonce++
					}
				} else {
					break hashLoop
				}
			}
		}
		miner.returnBlk()
		logger.Info("Miner: Mining Ends...")
	}()
}

func (miner *Miner) Stop() {
	if len(miner.exitCh) == 0 {
		miner.exitCh <- true
	}
}

func (miner *Miner) prepare() {
	miner.newBlock = miner.prepareBlock()
}

func (miner *Miner) returnBlk() {
	if !miner.newBlock.isValid {
		miner.newBlock.block.Rollback(miner.bc.GetTxPool())
	}
	miner.retChan <- miner.newBlock
}

func (miner *Miner) resetExitCh() {
	if len(miner.exitCh) > 0 {
		<-miner.exitCh
	}
}

func (miner *Miner) prepareBlock() *MinedBlock {

	parentBlock, err := miner.bc.GetTailBlock()
	if err != nil {
		logger.Error(err)
	}

	//verify all transactions
	miner.verifyTransactions()
	//get all transactions
	txs := miner.bc.GetTxPool().Pop()
	//calculate tips
	totalTips := common.NewAmount(0)
	for _, tx := range txs {
		totalTips = totalTips.Add(common.NewAmount(tx.Tip))
	}
	//add coinbase transaction to transaction pool
	cbtx := core.NewCoinbaseTX(miner.beneficiary, "", miner.bc.GetMaxHeight()+1, totalTips)
	txs = append(txs, &cbtx)

	miner.nonce = 0
	//prepare the new block (without the correct nonce value)
	return &MinedBlock{core.NewBlock(txs, parentBlock), false}
}

//returns true if a block is mined; returns false if the nonce value does not satisfy the difficulty requirement
func (miner *Miner) mineBlock(nonce int64) bool {
	hash := miner.newBlock.block.CalculateHashWithNonce(nonce)
	miner.newBlock.block.SetHash(hash)
	miner.newBlock.block.SetNonce(nonce)
	fulfilled := miner.requirement(miner.newBlock.block)
	if fulfilled {
		hash = miner.newBlock.block.CalculateHashWithoutNonce()
		miner.newBlock.block.SetHash(hash)
		keystring := miner.GetPrivKey()
		if len(keystring) > 0 {
			signed := miner.newBlock.block.SignBlock(miner.GetPrivKey(), hash)
			if !signed {
				logger.Warn("Miner Key= ", miner.GetPrivKey())
				return false
			}
		}
		miner.newBlock.isValid = true
	}
	return fulfilled
}

//verify transactions and remove invalid transactions
func (miner *Miner) verifyTransactions() {
	utxoPool := core.LoadUTXOIndex(miner.bc.GetDb())
	txPool := miner.bc.GetTxPool()
	txPool.RemoveInvalidTransactions(utxoPool)
}
