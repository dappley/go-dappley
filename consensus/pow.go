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
	"math/big"

	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

const defaultTargetBits = 0

var maxNonce int64 = math.MaxInt64

type ProofOfWork struct {
	miner  *BlockProducer
	target *big.Int
	node   core.NetService
	stopCh chan bool
}

func NewProofOfWork() *ProofOfWork {
	p := &ProofOfWork{
		miner:  NewBlockProducer(),
		node:   nil,
		stopCh: make(chan bool, 1),
	}
	p.SetTargetBit(defaultTargetBits)
	return p
}

func (pow *ProofOfWork) Setup(node core.NetService, cbAddr string) {
	pow.node = node
	pow.miner.Setup(node.GetBlockchain(), cbAddr)
	pow.miner.SetProcess(pow.calculateValidHash)
}

func (pow *ProofOfWork) SetTargetBit(bit int) {
	if bit < 0 || bit > 256 {
		return
	}
	target := big.NewInt(1)
	pow.target = target.Lsh(target, uint(256-bit))
}

func (pow *ProofOfWork) SetKey(key string) {
	// pow does not require block signing
}

func (pow *ProofOfWork) Start() {
	logger.Info("PoW starts...")
	pow.resetStopCh()
	go pow.mineBlocks()
}

func (pow *ProofOfWork) Stop() {
	logger.Info("PoW stops...")
	pow.stopCh <- true
}

func (pow *ProofOfWork) mineBlocks() {
	logger.Info("PoW: mining starts.")
	for {
		select {
		case <-pow.stopCh:
			logger.Info("PoW: mining stopped.")
			return
		default:
			if pow.node.GetBlockchain().GetState() != core.BlockchainReady {
				logger.Debug("BlockProducer: Paused while block pool is syncing")
				continue
			}
			newBlock := pow.miner.ProduceBlock()
			if newBlock == nil || !pow.Validate(newBlock.Block) {
				logger.WithFields(logger.Fields{"block": newBlock}).Debug("PoW: the block mined is invalid.")
				return
			}
			pow.updateNewBlock(newBlock)
			pow.miner.BlockProduceFinish()
		}
	}
}

func (pow *ProofOfWork) resetStopCh() {
L:
	for {
		select {
		case <-pow.stopCh:
		default:
			break L
		}
	}
}

func (pow *ProofOfWork) calculateValidHash(ctx *core.BlockContext) {
	for {
		select {
		case <-pow.stopCh:
			pow.stopCh <- true
			return
		default:
			hash := ctx.Block.CalculateHashWithNonce(ctx.Block.GetNonce())
			ctx.Block.SetHash(hash)
			if !pow.isHashBelowTarget(ctx.Block) {
				pow.tryDifferentNonce(ctx.Block)
				continue
			}
			return
		}
	}

}

func (pow *ProofOfWork) IsProducingBlock() bool {
	return !pow.miner.IsIdle()
}

func (pow *ProofOfWork) isHashBelowTarget(block *core.Block) bool {
	var hashInt big.Int

	hash := block.GetHash()
	hashInt.SetBytes(hash)

	return hashInt.Cmp(pow.target) == -1
}

func (pow *ProofOfWork) Validate(block *core.Block) bool {
	return pow.isHashBelowTarget(block)
}

func (pow *ProofOfWork) tryDifferentNonce(block *core.Block) {
	nonce := block.GetNonce()
	if nonce >= maxNonce {
		logger.Warn("PoW: tried all possible nonce.")
	}
	block.SetNonce(nonce + 1)
}

func (pow *ProofOfWork) updateNewBlock(ctx *core.BlockContext) {
	logger.WithFields(logger.Fields{"height": ctx.Block.GetHeight()}).Info("PoW: minted a new block.")
	if !ctx.Block.VerifyHash() {
		logger.Warn("PoW: the new block contains invalid hash (mining might have been interrupted).")
		return
	}
	err := pow.node.GetBlockchain().AddBlockContextToTail(ctx)
	if err != nil {
		logger.Warn(err)
		return
	}
	pow.node.BroadcastBlock(ctx.Block)
}

func (pow *ProofOfWork) AddProducer(producer string) error {
	return nil
}

func (pow *ProofOfWork) GetProducers() []string {
	return nil
}

func (pow *ProofOfWork) CheckLibPolicy(b *core.Block) (*core.Block, bool) {
	return nil, true
}
