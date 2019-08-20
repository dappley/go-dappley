package transactionpool

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
)

type NetService interface {
	GetHostPeerInfo() network_model.PeerInfo
	SendCommand(
		commandName string,
		message proto.Message,
		destination peer.ID,
		isBroadcast bool,
		priority network_model.DappCmdPriority)
	Listen(command string, handler network_model.CommandHandlerFunc)
	Relay(dappCmd *network_model.DappCmd, destination peer.ID, priority network_model.DappCmdPriority)
}
