package blockchain_logic

import (
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Storage interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, val []byte) error
}

type Consensus interface {
	Validate(*block.Block) bool

	Setup(string, *BlockchainManager)
	GetProducerAddress() string

	SetKey(string)

	// Start runs the consensus algorithm and begins to produce blocks
	Start()

	// Stop ceases the consensus algorithm and block production
	Stop()

	// IsProducingBlock returns true if this node itself is currently producing a block
	IsProducingBlock() bool
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
	SendCommand(
		commandName string,
		message proto.Message,
		destination peer.ID,
		isBroadcast bool,
		priority network_model.DappCmdPriority)
	Listen(command string, handler network_model.CommandHandlerFunc)
	Relay(dappCmd *network_model.DappCmd, destination peer.ID, priority network_model.DappCmdPriority)
}
