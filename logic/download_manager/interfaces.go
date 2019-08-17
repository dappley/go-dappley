package download_manager

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
)

type NetService interface {
	GetPeers() []network_model.PeerInfo
	GetHostPeerInfo() network_model.PeerInfo
	SendCommand(
		commandName string,
		message proto.Message,
		destination network_model.PeerInfo,
		isBroadcast bool,
		priority network_model.DappCmdPriority)
	Listen(command string, handler network_model.CommandHandlerFunc)
}
