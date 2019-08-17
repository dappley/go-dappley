package download_manager

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
)

type NetService interface {
	GetPeers() []network_model.PeerInfo
	GetHostPeerInfo() network_model.PeerInfo
	UnicastNormalPriorityCommand(commandName string, message proto.Message, destination network_model.PeerInfo)
	UnicastHighProrityCommand(commandName string, message proto.Message, destination network_model.PeerInfo)
	BroadcastNormalPriorityCommand(commandName string, message proto.Message)
	BroadcastHighProrityCommand(commandName string, message proto.Message)
	Listen(command string, handler network_model.CommandHandlerFunc)
}
