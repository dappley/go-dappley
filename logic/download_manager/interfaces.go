package download_manager

import (
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
)

type NetService interface {
	GetPeers() []*network_model.PeerInfo
	GetHostPeerInfo() *network_model.PeerInfo
	SendCommand(
		commandName string,
		message proto.Message,
		destination peer.ID,
		isBroadcast bool,
		priority network_model.DappCmdPriority)
	Subscribe(command string, handler network_model.CommandHandlerFunc)
}
