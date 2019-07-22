// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package network

import (
	"encoding/base64"
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	logger "github.com/sirupsen/logrus"
	"io/ioutil"
)

const (
	TopicOnStreamStop = "TopicOnStreamStop"

	requestChLen  = 1024
	dispatchChLen = 1024 * 4
)

var (
	reservedTopics = []string{
		TopicOnStreamStop,
	}
)

type Node struct {
	network       *Network
	commandBroker *CommandBroker
	exitCh        chan bool
	dispatcher    chan *network_model.DappPacketContext
	commandSendCh chan *network_model.DappSendCmdContext
}

//NewNode creates a new Node instance
func NewNode(db Storage) *Node {
	return NewNodeWithConfig(db, network_model.PeerConnectionConfig{})
}

//NewNodeWithConfig creates a new Node instance with configurations
func NewNodeWithConfig(db Storage, config network_model.PeerConnectionConfig) *Node {
	var err error

	node := &Node{
		exitCh:        make(chan bool, 1),
		dispatcher:    make(chan *network_model.DappPacketContext, dispatchChLen),
		commandSendCh: make(chan *network_model.DappSendCmdContext, requestChLen),
		commandBroker: NewCommandBroker(reservedTopics),
	}

	node.network = NewNetwork(config, node.dispatcher, db)
	node.network.OnStreamStop(node.OnStreamStop)
	node.RegisterSubscriber(node.network.peerManager)

	if err != nil {
		logger.WithError(err).Panic("Node: Can not initialize lru cache for recentlyRcvdDapMsgs!")
	}

	return node
}

//GetHostPeerInfo returns the host's peerInfo
func (n *Node) GetHostPeerInfo() *network_model.PeerInfo { return n.network.host.GetPeerInfo() }

//GetPeers returns all peers
func (n *Node) GetPeers() []*network_model.PeerInfo { return n.network.GetPeers() }

//GetNetwork returns its network object
func (n *Node) GetNetwork() *Network { return n.network }

//Start starts the network, command listener and received message listener
func (n *Node) Start(listenPort int, seeds []string, privKeyFilePath string) error {

	privKey := loadNetworkKeyFromFile(privKeyFilePath)

	err := n.network.Start(listenPort, privKey, seeds)
	if err != nil {
		return err
	}

	n.StartRequestLoop()
	n.StartListenLoop()
	return nil
}

//RegisterSubscriber registers a subscriber
func (n *Node) RegisterSubscriber(subscriber Subscriber) {
	subscriber.SetCommandSendCh(n.commandSendCh)
	n.commandBroker.Subscribe(subscriber)
}

//RegisterMultipleSubscribers registers multiple subscribers
func (n *Node) RegisterMultipleSubscribers(subscribers []Subscriber) {
	for _, subscriber := range subscribers {
		n.RegisterSubscriber(subscriber)
	}
}

//Stop stops the node
func (n *Node) Stop() {
	n.exitCh <- true
	n.network.Stop()
}

//StartRequestLoop starts a command sending request listener
func (n *Node) StartRequestLoop() {

	go func() {
		for {
			select {
			case <-n.exitCh:
				return
			case cmdCtx := <-n.commandSendCh:
				if cmdCtx.GetCommand() == nil {
					continue
				}

				rawBytes := cmdCtx.GetCommand().Serialize()

				if cmdCtx.IsBroadcast() {
					n.GetNetwork().Broadcast(rawBytes, cmdCtx.GetPriority())
				} else {
					n.GetNetwork().Unicast(rawBytes, cmdCtx.GetDestination(), cmdCtx.GetPriority())
				}

			}
		}
	}()
}

//StartListenLoop starts a received message listener
func (n *Node) StartListenLoop() {
	go func() {
		for {
			if streamMsg, ok := <-n.dispatcher; ok {

				cmdMsg := network_model.ParseDappMsgFromDappPacket(streamMsg.Packet)
				dappRcvdCmd := network_model.NewDappRcvdCmdContext(cmdMsg, streamMsg.Source)
				n.commandBroker.Dispatch(dappRcvdCmd)

			}
		}
	}()

}

//loadNetworkKeyFromFile reads the network privatekey source a file
func loadNetworkKeyFromFile(filePath string) crypto.PrivKey {
	if filePath == "" {
		return nil
	}

	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		logger.WithError(err).Warn("Node: LoadNetworkKeyFromFile failed.")
		return nil
	}

	data, err := base64.StdEncoding.DecodeString(string(bytes))
	if err != nil {
		logger.WithError(err).Warn("Node: LoadNetworkKeyFromFile failed.")
		return nil
	}

	privKey, err := crypto.UnmarshalPrivateKey(data)
	if err != nil {
		logger.WithError(err).Warn("Node: LoadNetworkKeyFromFile failed.")
		return nil
	}

	return privKey
}

//OnStreamStop runs when a stream is disconnected
func (n *Node) OnStreamStop(stream *Stream) {

	peerInfo := network_model.PeerInfo{PeerId: stream.GetPeerId()}
	bytes, err := proto.Marshal(peerInfo.ToProto())

	logger.WithError(err).Warn("Node: Marshal peerInfo failed")

	dappCmd := network_model.NewDappCmd(TopicOnStreamStop, bytes, false)
	dappCmdCtx := network_model.NewDappRcvdCmdContext(dappCmd, n.network.host.ID())

	n.commandBroker.Dispatch(dappCmdCtx)
}
