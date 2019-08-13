package miner

import (
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/logic/block_logic"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	logger "github.com/sirupsen/logrus"
	"time"
)

const (
	maxMintingTimeInMs = 2000
	NanoSecsInMilliSec = 1000000

	SendBlock    = "SendBlockByHash"
	RequestBlock = "requestBlock"
)

var (
	minerSubscribedTopics = []string{
		SendBlock,
		RequestBlock,
	}
)

type Miner struct {
	con      Consensus
	producer *consensus.BlockProducer
	bm       *blockchain_logic.BlockchainManager
	stopCh   chan bool
}

func NewMiner(bm *blockchain_logic.BlockchainManager, con Consensus) *Miner {
	miner := &Miner{
		con:      con,
		producer: consensus.NewBlockProducer(),
		bm:       bm,
		stopCh:   make(chan bool, 1),
	}
	miner.producer.SetProcess(con.GetProcess())
	return miner
}

func (miner *Miner) Start() {
	go func() {
		logger.Info("Miner Starts...")
		for {
			select {
			case <-miner.stopCh:
				return
			case <-miner.con.GetBlockProduceNotifier():
				deadlineInMs := time.Now().UnixNano()/NanoSecsInMilliSec + maxMintingTimeInMs

				logger.Infof("Miner: producing block... ***time is %v***", time.Now().Unix())

				// Do not produce block if block pool is syncing
				if miner.bm.Getblockchain().GetState() != blockchain.BlockchainReady {
					logger.Info("Miner: block producer paused because block pool is syncing.")
					continue
				}
				ctx := miner.producer.ProduceBlock(deadlineInMs)
				if ctx == nil || !miner.con.Validate(ctx.Block) {
					miner.producer.BlockProduceFinish()
					logger.Error("Miner: produced an invalid block!")
					continue
				}
				miner.updateNewBlock(ctx)
				miner.producer.BlockProduceFinish()
			}
		}
	}()
}

func (miner *Miner) Stop() {
	logger.Info("Miner stops...")
	miner.stopCh <- true
}

func (miner *Miner) updateNewBlock(ctx *blockchain_logic.BlockContext) {
	logger.WithFields(logger.Fields{
		"height": ctx.Block.GetHeight(),
		"hash":   ctx.Block.GetHash().String(),
	}).Info("Miner: produced a new block.")
	if !block_logic.VerifyHash(ctx.Block) {
		logger.Warn("Miner: hash of the new block is invalid.")
		return
	}

	// TODO Refactoring lib calculate position, check lib when create BlockContext instance
	if !miner.bm.Getblockchain().CheckLibPolicy(ctx.Block) {
		logger.Warn("Miner: the number of producers is not enough.")
		tailBlock, _ := miner.bm.Getblockchain().GetTailBlock()
		miner.bm.BroadcastBlock(tailBlock)
		return
	}

	err := miner.bm.Getblockchain().AddBlockContextToTail(ctx)
	if err != nil {
		logger.Warn(err)
		return
	}
	miner.bm.BroadcastBlock(ctx.Block)
}
