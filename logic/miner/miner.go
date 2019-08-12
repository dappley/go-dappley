package miner

import (
	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/block/pb"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/logic/block_logic"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
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
	con        Consensus
	producer   *consensus.BlockProducer
	bc         *blockchain_logic.Blockchain
	netService NetService
	stopCh     chan bool
}

func NewMiner(bc *blockchain_logic.Blockchain, con Consensus, netService NetService) *Miner {
	miner := &Miner{
		con:        con,
		producer:   consensus.NewBlockProducer(),
		bc:         bc,
		netService: netService,
		stopCh:     make(chan bool, 1),
	}
	miner.producer.SetProcess(con.GetProcess())
	miner.ListenToNetService()
	return miner
}

func (miner *Miner) ListenToNetService() {
	if miner.netService == nil {
		return
	}

	for _, command := range minerSubscribedTopics {
		miner.netService.Listen(command, miner.GetCommandHandler(command))
	}
}

func (miner *Miner) GetCommandHandler(commandName string) network_model.CommandHandlerFunc {

	switch commandName {
	case SendBlock:
		return miner.SendBlockHandler
	case RequestBlock:
		return miner.RequestBlockHandler
	}
	return nil
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
				if miner.bc.GetState() != blockchain.BlockchainReady {
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
	lib, ok := miner.con.CheckLibPolicy(ctx.Block)
	if !ok {
		logger.Warn("Miner: the number of producers is not enough.")
		tailBlock, _ := miner.bc.GetTailBlock()
		miner.BroadcastBlock(tailBlock)
		return
	}
	ctx.Lib = lib

	err := miner.bc.AddBlockContextToTail(ctx)
	if err != nil {
		logger.Warn(err)
		return
	}
	miner.BroadcastBlock(ctx.Block)
}

//RequestBlock sends a requestBlock command to its peer with pid through network module
func (miner *Miner) RequestBlock(hash hash.Hash, pid peer.ID) {
	request := &corepb.RequestBlock{Hash: hash}

	miner.netService.SendCommand(RequestBlock, request, pid, network_model.Unicast, network_model.HighPriorityCommand)
}

//RequestBlockhandler handles when blockchain manager receives a requestBlock command from its peers
func (miner *Miner) RequestBlockHandler(command *network_model.DappRcvdCmdContext) {
	request := &corepb.RequestBlock{}

	if err := proto.Unmarshal(command.GetData(), request); err != nil {
		logger.WithFields(logger.Fields{
			"name": command.GetCommandName(),
		}).Info("Miner: parse data failed.")
	}

	block, err := miner.bc.GetBlockByHash(request.Hash)
	if err != nil {
		logger.WithError(err).Warn("Miner: failed to get the requested block.")
		return
	}

	miner.SendBlockToPeer(block, command.GetSource())
}

//SendBlockToPeer unicasts a block to the peer with peer id "pid"
func (miner *Miner) SendBlockToPeer(blk *block.Block, pid peer.ID) {

	miner.SendBlock(blk, pid, network_model.Unicast)
}

//BroadcastBlock broadcasts a block to all peers
func (miner *Miner) BroadcastBlock(blk *block.Block) {
	var broadcastPid peer.ID
	miner.SendBlock(blk, broadcastPid, network_model.Broadcast)
}

//SendBlock sends a SendBlock command to its peer with pid by finding the block from its database
func (miner *Miner) SendBlock(blk *block.Block, pid peer.ID, isBroadcast bool) {

	miner.netService.SendCommand(SendBlock, blk.ToProto(), pid, isBroadcast, network_model.HighPriorityCommand)
}

//SendBlockHandler handles when blockchain manager receives a sendBlock command from its peers
func (miner *Miner) SendBlockHandler(command *network_model.DappRcvdCmdContext) {
	pb := &blockpb.Block{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(command.GetData(), pb); err != nil {
		logger.WithError(err).Warn("Miner: parse data failed.")
		return
	}

	blk := &block.Block{}
	blk.FromProto(pb)
	miner.Push(blk, command.GetSource())

	if command.IsBroadcast() {
		//relay the original command
		var broadcastPid peer.ID
		miner.netService.Relay(command.GetCommand(), broadcastPid, network_model.HighPriorityCommand)
	}
}
