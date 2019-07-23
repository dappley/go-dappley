package download_manager

import "github.com/dappley/go-dappley/network/network_model"

type NetService interface {
	GetPeers() []*network_model.PeerInfo
	GetHostPeerInfo() *network_model.PeerInfo
}
