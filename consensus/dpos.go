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
	"encoding/hex"
	"strings"
	"time"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	lru "github.com/hashicorp/golang-lru"
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
	newBlockCh  chan *core.Block
	node        core.NetService
	stopCh      chan bool
	stopLibCh   chan bool
	dynasty     *Dynasty
	slot        *lru.Cache
}

func NewDPOS() *DPOS {
	dpos := &DPOS{
		bp:         NewBlockProducer(),
		newBlockCh: make(chan *core.Block, 1),
		node:       nil,
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

func (dpos *DPOS) AddBlockToSlot(block *core.Block) {
	dpos.slot.Add(int(block.GetTimestamp()/int64(dpos.GetDynasty().timeBetweenBlk)), block)
}

func (dpos *DPOS) Setup(node core.NetService, cbAddr string) {
	dpos.node = node
	dpos.bp.Setup(node.GetBlockchain(), cbAddr)
	dpos.bp.SetProcess(dpos.hashAndSign)
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

// Validate checks that the block fulfills the dpos requirement and accepts the block in the time slot
func (dpos *DPOS) Validate(block *core.Block) bool {
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
		logger.WithFields(logger.Fields{
			"peer_id": dpos.node.GetPeerID(),
		}).Info("DPoS starts...")
		if len(dpos.stopCh) > 0 {
			<-dpos.stopCh
		}
		ticker := time.NewTicker(time.Second).C

		for {
			select {
			case now := <-ticker:
				if dpos.dynasty.IsMyTurn(dpos.bp.Beneficiary(), now.Unix()) {
					deadlineInMs := now.UnixNano()/NanoSecsInMilliSec + maxMintingTimeInMs
					logger.WithFields(logger.Fields{
						"peer_id": dpos.node.GetPeerID(),
					}).Info("DPoS: it is my turn to produce block.")
					// Do not produce block if block pool is syncing
					if dpos.node.GetBlockchain().GetState() != core.BlockchainReady {
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
	logger.WithFields(logger.Fields{
		"peer_id": dpos.node.GetPeerID(),
	}).Info("DPoS stops...")
	dpos.stopCh <- true
}

func (dpos *DPOS) hashAndSign(ctx *core.BlockContext) {
	//block.SetNonce(0)
	hash := ctx.Block.CalculateHash()
	ctx.Block.SetHash(hash)
	ok := ctx.Block.SignBlock(dpos.producerKey, hash)
	if !ok {
		logger.Warn("DPoS: failed to sign the new block.")
	}
}

func (dpos *DPOS) isForking() bool {
	return false
}

func (dpos *DPOS) isDoubleMint(block *core.Block) bool {
	existBlock, exist := dpos.slot.Get(int(block.GetTimestamp() / int64(dpos.GetDynasty().timeBetweenBlk)))
	if !exist {
		return false
	}

	return !core.IsHashEqual(existBlock.(*core.Block).GetHash(), block.GetHash())
}

// verifyProducer verifies a given block is produced by the valid producer by verifying the signature of the block
func (dpos *DPOS) verifyProducer(block *core.Block) bool {
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

	pubKeyHash, err := core.NewUserPubKeyHash(pubkey[1:])
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
func (dpos *DPOS) beneficiaryIsProducer(block *core.Block) bool {
	if block == nil {
		logger.Debug("DPoS: block is empty.")
		return false
	}

	producer := dpos.dynasty.ProducerAtATime(block.GetTimestamp())
	producerHash, _ := core.NewAddress(producer).GetPubKeyHash()

	cbtx := block.GetCoinbaseTransaction()
	if cbtx == nil {
		logger.Debug("DPoS: coinbase tx is empty.")
		return false
	}

	if len(cbtx.Vout) == 0 {
		logger.Debug("DPoS: coinbase vout is empty.")
		return false
	}

	return bytes.Compare(producerHash, []byte(cbtx.Vout[0].PubKeyHash)) == 0
}

func (dpos *DPOS) IsProducingBlock() bool {
	return !dpos.bp.IsIdle()
}

func (dpos *DPOS) updateNewBlock(ctx *core.BlockContext) {
	logger.WithFields(logger.Fields{
		"peer_id": dpos.node.GetPeerID(),
		"height":  ctx.Block.GetHeight(),
		"hash":    hex.EncodeToString(ctx.Block.GetHash()),
	}).Info("DPoS: produced a new block.")
	if !ctx.Block.VerifyHash() {
		logger.Warn("DPoS: hash of the new block is invalid.")
		return
	}

	// TODO Refactoring lib calculate position, check lib when create BlockContext instance
	lib, ok := dpos.CheckLibPolicy(ctx.Block)
	if !ok {
		logger.Warn("DPoS: the number of producers is not enough.")
		tailBlock, _ := dpos.node.GetBlockchain().GetTailBlock()
		dpos.node.BroadcastBlock(tailBlock)
		return
	}
	ctx.Lib = lib

	err := dpos.node.GetBlockchain().AddBlockContextToTail(ctx)
	if err != nil {
		logger.Warn(err)
		return
	}
	dpos.node.BroadcastBlock(ctx.Block)
}

func (dpos *DPOS) CheckLibPolicy(b *core.Block) (*core.Block, bool) {
	//Do not check genesis block
	if b.GetHeight() == 0 {
		return b, true
	}

	// If producers number is less than MinConsensusSize, pass all blocks
	if len(dpos.dynasty.GetProducers()) < MinConsensusSize {
		return nil, true
	}

	lib, err := dpos.node.GetBlockchain().GetLIB()
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

		newBlock, err := dpos.node.GetBlockchain().GetBlockByHash(checkingBlock.GetPrevHash())
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
