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
	SendCommand(
		commandName string,
		message proto.Message,
		destination network_model.PeerInfo,
		isBroadcast bool,
		priority network_model.DappCmdPriority)
	Listen(command string, handler network_model.CommandHandlerFunc)
}
