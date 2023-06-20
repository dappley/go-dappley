package lblockchain

import (
	"github.com/dappley/go-dappley/common/pubsub"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/network/networkmodel"
	"github.com/golang/protobuf/proto"
)

type Storage interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, val []byte) error
}

type Consensus interface {
	Validate(*block.Block) bool
	GetDynasty() *consensus.Dynasty
	SetDynasty(dynasty *consensus.Dynasty)
	ChangeDynasty(height uint64)
	ChangeDynastyTemp(height uint64)
	AddReplacement(original, new string, height uint64, kind int)
}

type LIBPolicy interface {
	GetMinConfirmationNum() int
	IsBypassingLibCheck() bool
	GetTotalProducersNum() int
}

type NetService interface {
	UnicastNormalPriorityCommand(commandName string, message proto.Message, destination networkmodel.PeerInfo)
	UnicastHighProrityCommand(commandName string, message proto.Message, destination networkmodel.PeerInfo)
	BroadcastNormalPriorityCommand(commandName string, message proto.Message)
	BroadcastHighProrityCommand(commandName string, message proto.Message)
	Listen(subscriber pubsub.Subscriber)
	Relay(dappCmd *networkmodel.DappCmd, destination networkmodel.PeerInfo, priority networkmodel.DappCmdPriority)
}
