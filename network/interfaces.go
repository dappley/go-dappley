package network

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
)

type Storage interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, val []byte) error
}

type NetService interface {
	UnicastNormalPriorityCommand(commandName string, message proto.Message, destination network_model.PeerInfo)
	UnicastHighProrityCommand(commandName string, message proto.Message, destination network_model.PeerInfo)
	BroadcastNormalPriorityCommand(commandName string, message proto.Message)
	BroadcastHighProrityCommand(commandName string, message proto.Message)
	Listen(command string, handler network_model.CommandHandlerFunc)
}
