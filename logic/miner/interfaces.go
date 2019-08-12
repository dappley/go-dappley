package miner

import (
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Consensus interface {
	GetBlockProduceNotifier() chan bool
	Validate(*block.Block) bool
	//Return the lib block and new block whether pass lib policy
	CheckLibPolicy(*block.Block) (*block.Block, bool)
	GetProcess() consensus.Process
}

type NetService interface {
	SendCommand(
		commandName string,
		message proto.Message,
		destination peer.ID,
		isBroadcast bool,
		priority network_model.DappCmdPriority)
	Listen(command string, handler network_model.CommandHandlerFunc)
	Relay(dappCmd *network_model.DappCmd, destination peer.ID, priority network_model.DappCmdPriority)
}
