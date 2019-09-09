package lblockchain

import (
	"github.com/dappley/go-dappley/common/pubsub"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
)

type Storage interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, val []byte) error
}

type Consensus interface {
	Validate(*block.Block) bool

	GetProducerAddress() string

	// Produced returns true iff the underlying block producer of the consensus algorithm produced the specified block
	Produced(*block.Block) bool

	// TODO: Should separate the concept of producers from PoW
	AddProducer(string) error
	GetProducers() []string

	GetLibProducerNum() int
	IsBypassingLibCheck() bool
	IsNonRepeatingBlockProducerRequired() bool
}

type NetService interface {
	UnicastNormalPriorityCommand(commandName string, message proto.Message, destination network_model.PeerInfo)
	UnicastHighProrityCommand(commandName string, message proto.Message, destination network_model.PeerInfo)
	BroadcastNormalPriorityCommand(commandName string, message proto.Message)
	BroadcastHighProrityCommand(commandName string, message proto.Message)
	Listen(subscriber pubsub.Subscriber)
	Relay(dappCmd *network_model.DappCmd, destination network_model.PeerInfo, priority network_model.DappCmdPriority)
}
