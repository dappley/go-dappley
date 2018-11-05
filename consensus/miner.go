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

	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
)

type State int

var maxNonce int64 = math.MaxInt64

type Miner struct {
	exitCh      chan bool
	bc          *core.Blockchain
	beneficiary string
	key         string
	newBlock    *NewBlock
	nonce       int64
	requirement Requirement
	newBlockCh  chan *NewBlock
	idle        bool
}

func NewMiner() *Miner {
	m := &Miner{
		exitCh:      make(chan bool, 1),
		bc:          nil,
		beneficiary: "",
		newBlock:    &NewBlock{nil, false},
		nonce:       0,
		requirement: noRequirement,
		idle:        true,
	}
	return m
}

func (miner *Miner) SetPrivateKey(key string) {
	miner.key = key
}

func (miner *Miner) Beneficiary() string {
	return miner.beneficiary
}

func (miner *Miner) SetRequirement(requirement Requirement) {
	miner.requirement = requirement
}

func (miner *Miner) Setup(bc *core.Blockchain, beneficiaryAddr string, newBlockCh chan *NewBlock) {
	miner.bc = bc
	miner.beneficiary = beneficiaryAddr
	miner.newBlockCh = newBlockCh
}

func (miner *Miner) Start() {
	go func() {
		if miner.bc.GetBlockPool().GetSyncState() {
			return
		}
		logger.Info("Miner: Start Mining A Block...")
		miner.resetExitCh()
		miner.idle = false
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
		miner.idle = true
		logger.Info("Miner: Mining Ends...")
	}()
}

func (miner *Miner) Stop() {
	if len(miner.exitCh) == 0 {
		miner.exitCh <- true
	}
}

func (miner *Miner) IsIdle() bool {
	return miner.idle
}

func (miner *Miner) prepare() {
	miner.newBlock = miner.prepareBlock()
}

func (miner *Miner) returnBlk() {
	if !miner.newBlock.IsValid {
		miner.newBlock.Rollback(miner.bc.GetTxPool())
	}
	miner.newBlockCh <- miner.newBlock
}

func (miner *Miner) resetExitCh() {
	if len(miner.exitCh) > 0 {
		<-miner.exitCh
	}
}

func (miner *Miner) prepareBlock() *NewBlock {

	parentBlock, err := miner.bc.GetTailBlock()
	if err != nil {
		logger.Error(err)
	}
	utxoIndex := core.LoadUTXOIndex(miner.bc.GetDb())
	validTxs := miner.bc.GetTxPool().GetValidTxs(utxoIndex)

	// update UTXO set
	for i, tx := range validTxs {
		// remove transaction if utxo set cannot be updated
		if !utxoIndex.UpdateUtxo(tx) {
			validTxs = append(validTxs[:i], validTxs[i + 1:]...)
		}
	}

	//calculate tips
	totalTips := common.NewAmount(0)
	for _, tx := range validTxs {
		totalTips = totalTips.Add(common.NewAmount(tx.Tip))
	}
	//add coinbase transaction to transaction pool
	cbtx := core.NewCoinbaseTX(miner.beneficiary, "", miner.bc.GetMaxHeight()+1, totalTips)
	validTxs = append(validTxs, &cbtx)

	miner.nonce = 0
	//prepare the new block (without the correct nonce value)
	return &NewBlock{core.NewBlock(validTxs, parentBlock), false}
}

//returns true if a block is mined; returns false if the nonce value does not satisfy the difficulty requirement
func (miner *Miner) mineBlock(nonce int64) bool {
	hash := miner.newBlock.CalculateHashWithNonce(nonce)
	miner.newBlock.SetHash(hash)
	miner.newBlock.SetNonce(nonce)
	fulfilled := miner.requirement(miner.newBlock.Block)
	if fulfilled {
		hash = miner.newBlock.CalculateHashWithoutNonce()
		miner.newBlock.SetHash(hash)
		if len(miner.key) > 0 {
			signed := miner.newBlock.SignBlock(miner.key, hash)
			if !signed {
				logger.Warn("Miner Key= ", miner.key)
				return false
			}
		}
		miner.newBlock.IsValid = true
	}
	return fulfilled
}

//verify transactions and remove invalid transactions

