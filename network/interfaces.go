package network

import "github.com/dappley/go-dappley/network/network_model"

type Storage interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, val []byte) error
}

type Subscriber interface {
	GetSubscribedTopics() []string
	SetCommandSendCh(commandSendCh chan *network_model.DappSendCmdContext)
	GetCommandHandler(cmd string) network_model.CommandHandlerFunc
}
