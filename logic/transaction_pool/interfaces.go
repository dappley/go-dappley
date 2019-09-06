package transaction_pool

import (
	"github.com/dappley/go-dappley/common/pubsub"
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
)

type NetService interface {
	GetHostPeerInfo() network_model.PeerInfo
	UnicastNormalPriorityCommand(commandName string, message proto.Message, destination network_model.PeerInfo)
	UnicastHighProrityCommand(commandName string, message proto.Message, destination network_model.PeerInfo)
	BroadcastNormalPriorityCommand(commandName string, message proto.Message)
	BroadcastHighProrityCommand(commandName string, message proto.Message)
	Listen(subscriber pubsub.Subscriber)
	Relay(dappCmd *network_model.DappCmd, destination network_model.PeerInfo, priority network_model.DappCmdPriority)
}
