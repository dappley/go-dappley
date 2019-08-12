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
	"bytes"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/logic/block_logic"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	"strings"
	"time"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/hashicorp/golang-lru"
	logger "github.com/sirupsen/logrus"
)

const (
	MinConsensusSize = 4
)

const maxMintingTimeInMs = 2000
const NanoSecsInMilliSec = 1000000

type DPOS struct {
	bp          *BlockProducer
	producerKey string
	newBlockCh  chan *block.Block
	bm          *blockchain_logic.BlockchainManager
	stopCh      chan bool
	stopLibCh   chan bool
	dynasty     *Dynasty
	slot        *lru.Cache
}

func NewDPOS() *DPOS {
	dpos := &DPOS{
		bp:         NewBlockProducer(),
		newBlockCh: make(chan *block.Block, 1),
		stopCh:     make(chan bool, 1),
		stopLibCh:  make(chan bool, 1),
	}

	slot, err := lru.New(128)
	if err != nil {
		logger.Panic(err)
	}
	dpos.slot = slot
	return dpos
}

func (dpos *DPOS) GetSlot() *lru.Cache {
	return dpos.slot
}

func (dpos *DPOS) AddBlockToSlot(block *block.Block) {
	dpos.slot.Add(int(block.GetTimestamp()/int64(dpos.GetDynasty().timeBetweenBlk)), block)
}

func (dpos *DPOS) Setup(cbAddr string, bm *blockchain_logic.BlockchainManager) {
	dpos.bp.Setup(bm.Getblockchain(), cbAddr)
	dpos.bp.SetProcess(dpos.hashAndSign)
	dpos.bm = bm
}

func (dpos *DPOS) SetKey(key string) {
	dpos.producerKey = key
}

func (dpos *DPOS) SetDynasty(dynasty *Dynasty) {
	dpos.dynasty = dynasty
}

func (dpos *DPOS) GetDynasty() *Dynasty {
	return dpos.dynasty
}

func (dpos *DPOS) AddProducer(producer string) error {
	err := dpos.dynasty.AddProducer(producer)
	return err
}

func (dpos *DPOS) GetProducers() []string {
	return dpos.dynasty.GetProducers()
}

func (dpos *DPOS) GetProducerAddress() string {
	return dpos.bp.Beneficiary()
}

// Validate checks that the block fulfills the dpos requirement and accepts the block in the time slot
func (dpos *DPOS) Validate(block *block.Block) bool {
	producerIsValid := dpos.verifyProducer(block)
	if !producerIsValid {
		return false
	}

	if !dpos.beneficiaryIsProducer(block) {
		logger.Debug("DPoS: failed to validate producer.")
		return false
	}
	if dpos.isDoubleMint(block) {
		logger.Warn("DPoS: double-minting is detected.")
		return false
	}

	dpos.AddBlockToSlot(block)
	return true
}

func (dpos *DPOS) Start() {
	go func() {
		logger.Info("DPoS starts...")
		if len(dpos.stopCh) > 0 {
			<-dpos.stopCh
		}
		ticker := time.NewTicker(time.Second).C

		for {
			select {
			case now := <-ticker:
				if dpos.dynasty.IsMyTurn(dpos.bp.Beneficiary(), now.Unix()) {
					deadlineInMs := now.UnixNano()/NanoSecsInMilliSec + maxMintingTimeInMs
					index := dpos.dynasty.GetProducerIndex(dpos.bp.Beneficiary())
					logger.Infof("DPoS: it is my turn to produce block. ***node is %v,time is %v***", index, now.Unix())

					// Do not produce block if block pool is syncing
					if dpos.bm.Getblockchain().GetState() != blockchain.BlockchainReady {
						logger.Info("DPoS: block producer paused because block pool is syncing.")
						continue
					}
					ctx := dpos.bp.ProduceBlock(deadlineInMs)
					if ctx == nil || !dpos.Validate(ctx.Block) {
						dpos.bp.BlockProduceFinish()
						logger.Error("DPoS: produced an invalid block!")
						continue
					}
					dpos.updateNewBlock(ctx)
					dpos.bp.BlockProduceFinish()
				}
			case <-dpos.stopCh:
				return

			}
		}
	}()
}

func (dpos *DPOS) Stop() {
	logger.Info("DPoS stops...")
	dpos.stopCh <- true
}

func (dpos *DPOS) Produced(blk *block.Block) bool {
	if blk != nil {
		return dpos.bp.Produced(blk)
	}
	return false
}

func (dpos *DPOS) hashAndSign(blk *block.Block) {
	hash := block_logic.CalculateHash(blk)
	blk.SetHash(hash)
	ok := block_logic.SignBlock(blk, dpos.producerKey)
	if !ok {
		logger.Warn("DPoS: failed to sign the new block.")
	}
}

func (dpos *DPOS) isForking() bool {
	return false
}

func (dpos *DPOS) isDoubleMint(blk *block.Block) bool {
	existBlock, exist := dpos.slot.Get(int(blk.GetTimestamp() / int64(dpos.GetDynasty().timeBetweenBlk)))
	if !exist {
		return false
	}

	return !block_logic.IsHashEqual(existBlock.(*block.Block).GetHash(), blk.GetHash())
}

// verifyProducer verifies a given block is produced by the valid producer by verifying the signature of the block
func (dpos *DPOS) verifyProducer(block *block.Block) bool {
	if block == nil {
		logger.Warn("DPoS: block is empty!")
		return false
	}

	hash := block.GetHash()
	sign := block.GetSign()

	producer := dpos.dynasty.ProducerAtATime(block.GetTimestamp())

	if hash == nil {
		logger.Warn("DPoS: block hash is empty!")
		return false
	}
	if sign == nil {
		logger.Warn("DPoS: block signature is empty!")
		return false
	}

	pubkey, err := secp256k1.RecoverECDSAPublicKey(hash, sign)
	if err != nil {
		logger.WithError(err).Warn("DPoS: cannot recover the public key from the block signature!")
		return false
	}

	pubKeyHash, err := account.NewUserPubKeyHash(pubkey[1:])
	if err != nil {
		logger.WithError(err).Warn("DPoS: cannot compute the public key hash!")
		return false
	}

	address := pubKeyHash.GenerateAddress()

	if strings.Compare(address.String(), producer) != 0 {
		logger.Warn("DPoS: the signer is not the producer in this time slot.")
		return false
	}

	return true
}

// beneficiaryIsProducer is a requirement that ensures the reward is paid to the producer at the time slot
func (dpos *DPOS) beneficiaryIsProducer(block *block.Block) bool {
	if block == nil {
		logger.Debug("DPoS: block is empty.")
		return false
	}

	producer := dpos.dynasty.ProducerAtATime(block.GetTimestamp())
	producerHash, _ := account.GeneratePubKeyHashByAddress(account.NewAddress(producer))

	cbtx := block.GetCoinbaseTransaction()
	if cbtx == nil {
		logger.Debug("DPoS: coinbase tx is empty.")
		return false
	}

	if len(cbtx.Vout) == 0 {
		logger.Debug("DPoS: coinbase vout is empty.")
		return false
	}

	return bytes.Compare(producerHash, cbtx.Vout[0].PubKeyHash) == 0
}

func (dpos *DPOS) IsProducingBlock() bool {
	return !dpos.bp.IsIdle()
}

func (dpos *DPOS) updateNewBlock(ctx *blockchain_logic.BlockContext) {
	logger.WithFields(logger.Fields{
		"height": ctx.Block.GetHeight(),
		"hash":   ctx.Block.GetHash().String(),
	}).Info("DPoS: produced a new block.")
	if !block_logic.VerifyHash(ctx.Block) {
		logger.Warn("DPoS: hash of the new block is invalid.")
		return
	}

	// TODO Refactoring lib calculate position, check lib when create BlockContext instance
	lib, ok := dpos.CheckLibPolicy(ctx.Block)
	if !ok {
		logger.Warn("DPoS: the number of producers is not enough.")
		tailBlock, _ := dpos.bm.Getblockchain().GetTailBlock()
		dpos.bm.BroadcastBlock(tailBlock)
		return
	}
	ctx.Lib = lib

	err := dpos.bm.Getblockchain().AddBlockContextToTail(ctx)
	if err != nil {
		logger.Warn(err)
		return
	}
	dpos.bm.BroadcastBlock(ctx.Block)
}

func (dpos *DPOS) CheckLibPolicy(b *block.Block) (*block.Block, bool) {
	//Do not check genesis block
	if b.GetHeight() == 0 {
		return b, true
	}

	// If producers number is less than MinConsensusSize, pass all blocks
	if len(dpos.dynasty.GetProducers()) < MinConsensusSize {
		return nil, true
	}

	lib, err := dpos.bm.Getblockchain().GetLIB()
	if err != nil {
		logger.WithError(err).Warn("DPoS: get lib failed.")
	}

	libProduerNum := dpos.getLibProducerNum()
	existProducers := make(map[string]int)

	checkingBlock := b

	for lib.GetHash().Equals(checkingBlock.GetHash()) == false {
		_, ok := existProducers[checkingBlock.GetProducer()]
		if ok {
			logger.WithFields(logger.Fields{
				"producer": checkingBlock.GetProducer(),
			}).Info("DPoS: duplicate producer when check lib.")
			return nil, false
		}

		existProducers[checkingBlock.GetProducer()] = 1
		if len(existProducers) >= libProduerNum {
			return checkingBlock, true
		}

		newBlock, err := dpos.bm.Getblockchain().GetBlockByHash(checkingBlock.GetPrevHash())
		if err != nil {
			logger.WithError(err).Warn("DPoS: get parent block failed.")
		}

		checkingBlock = newBlock
	}

	// No enough checking blocks
	return nil, true
}

func (dpos *DPOS) getLibProducerNum() int {
	return len(dpos.dynasty.GetProducers())*2/3 + 1
}
