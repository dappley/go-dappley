package network

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Storage interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, val []byte) error
}

type Subscriber interface {
	GetSubscribedTopics() []string
	SetCommandSendCh(commandSendCh chan *network_model.DappSendCmdContext)
	GetCommandHandler(commandName string) network_model.CommandHandlerFunc
}

type NetService interface {
	SendCommand(
		commandName string,
		message proto.Message,
		destination peer.ID,
		isBroadcast bool,
		priority network_model.DappCmdPriority)
	Subscribe(command string, handler network_model.CommandHandlerFunc)
}
