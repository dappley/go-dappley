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
	"fmt"
	"github.com/dappley/go-dappley/common/log"
	"github.com/dappley/go-dappley/common/pubsub"
	"io/ioutil"

	"github.com/libp2p/go-libp2p-core/host"

	"github.com/dappley/go-dappley/network/networkmodel"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
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
	commandBroker *pubsub.CommandBroker
	exitCh        chan bool
	dispatcher    chan *networkmodel.DappPacketContext
	commandSendCh chan *networkmodel.DappSendCmdContext
}

//NewNode creates a new Node instance
func NewNode(db Storage, seeds []string) *Node {
	return NewNodeWithConfig(db, networkmodel.PeerConnectionConfig{}, seeds)
}

//NewNodeWithConfig creates a new Node instance with configurations
func NewNodeWithConfig(db Storage, config networkmodel.PeerConnectionConfig, seeds []string) *Node {
	var err error

	node := &Node{
		exitCh:        make(chan bool, 1),
		dispatcher:    make(chan *networkmodel.DappPacketContext, dispatchChLen),
		commandSendCh: make(chan *networkmodel.DappSendCmdContext, requestChLen),
		commandBroker: pubsub.NewCommandBroker(reservedTopics),
	}

	node.network = NewNetwork(
		&NetworkContext{
			node,
			config,
			node.dispatcher,
			db,
			node.onStreamStop,
			seeds,
		})

	if err != nil {
		logger.WithError(err).Panic("Node: Can not initialize lru cache for recentlyRcvdDapMsgs!")
	}

	return node
}

func (n *Node) GetIPFSAddresses() []string {

	addrs := n.GetHostPeerInfo().Addrs

	addresses := make([]string, len(addrs))
	addr, err := buildHostMultiAddress(n.GetNetwork().GetHost())
	if err != nil {
		logger.Error(err)
		return addresses
	}
	for i, v := range addrs {
		addresses[i] = v.Encapsulate(addr).String()
	}
	return addresses
}

//GetHostPeerInfo returns the host's peerInfo
func (n *Node) GetHostPeerInfo() networkmodel.PeerInfo { return n.network.GetHost().GetPeerInfo() }

//GetConnectedPeers returns all peers
func (n *Node) GetPeers() []networkmodel.PeerInfo { return n.network.GetConnectedPeers() }

//GetNetwork returns its network object
func (n *Node) GetNetwork() *Network { return n.network }

//Start starts the network, command listener and received message listener
func (n *Node) Start(listenPort int, privKeyFilePath string) error {

	privKey := loadNetworkKeyFromFile(privKeyFilePath)

	err := n.network.Start(listenPort, privKey)
	if err != nil {
		return err
	}

	n.StartRequestLoop()
	n.StartListenLoop()
	return nil
}

func buildHostMultiAddress(host host.Host) (ma.Multiaddr, error) {
	return ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", host.ID().Pretty()))
}

//UnicastNormalPriorityCommand sends a normal priority command to a peer
func (n *Node) UnicastNormalPriorityCommand(commandName string, message proto.Message, destination networkmodel.PeerInfo) {
	n.unicast(commandName, message, destination, networkmodel.NormalPriorityCommand)
}

//UnicastHighProrityCommand sends a high priority command to a peer
func (n *Node) UnicastHighProrityCommand(commandName string, message proto.Message, destination networkmodel.PeerInfo) {
	n.unicast(commandName, message, destination, networkmodel.HighPriorityCommand)
}

//BroadcastNormalPriorityCommand sends a normal priority command to all peers
func (n *Node) BroadcastNormalPriorityCommand(commandName string, message proto.Message) {
	n.broadcast(commandName, message, networkmodel.NormalPriorityCommand)
}

//BroadcastHighProrityCommand sends a high priority command to all peers
func (n *Node) BroadcastHighProrityCommand(commandName string, message proto.Message) {
	n.broadcast(commandName, message, networkmodel.HighPriorityCommand)
}

//Unicast sends a command to a peer
func (n *Node) unicast(commandName string, message proto.Message, destination networkmodel.PeerInfo, priority networkmodel.DappCmdPriority) {
	n.sendCommand(commandName, message, destination, networkmodel.Unicast, priority)
}

//Broadcast sends a command to all peers
func (n *Node) broadcast(commandName string, message proto.Message, priority networkmodel.DappCmdPriority) {
	n.sendCommand(commandName, message, networkmodel.PeerInfo{}, networkmodel.Broadcast, priority)
}

//Relay relays a command to a peer or all peers
func (n *Node) Relay(dappCmd *networkmodel.DappCmd, destination networkmodel.PeerInfo, priority networkmodel.DappCmdPriority) {
	command := networkmodel.NewDappSendCmdContextFromDappCmd(dappCmd, destination.PeerId, priority)
	select {
	case n.commandSendCh <- command:
	default:
		logger.WithFields(logger.Fields{
			"lenOfDispatchChan": len(n.commandSendCh),
		}).Warn("DappSendCmdContext: request channel full")
	}
}

//sendCommand sens a command to a peer or all peers
func (n *Node) sendCommand(commandName string, message proto.Message, destination networkmodel.PeerInfo, isBroadcast bool, priority networkmodel.DappCmdPriority) {
	command := networkmodel.NewDappSendCmdContext(commandName, message, destination.PeerId, isBroadcast, priority)
	select {
	case n.commandSendCh <- command:
	default:
		logger.WithFields(logger.Fields{
			"lenOfDispatchChan": len(n.commandSendCh),
		}).Warn("DappSendCmdContext: request channel full")
	}
}

//Listen registers a callback function for a topic
func (n *Node) Listen(subscriber pubsub.Subscriber) {
	n.commandBroker.AddSubscriber(subscriber)
}

//Stop stops the node
func (n *Node) Stop() {
	n.exitCh <- true
	n.network.Stop()
}

//StartRequestLoop starts a command sending request listener
func (n *Node) StartRequestLoop() {

	go func() {
		defer log.CrashHandler()

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
		defer log.CrashHandler()

		for {
			if streamMsg, ok := <-n.dispatcher; ok {

				cmdMsg := networkmodel.ParseDappMsgFromDappPacket(streamMsg.Packet)
				dappRcvdCmd := networkmodel.NewDappRcvdCmdContext(cmdMsg, streamMsg.Source)
				n.commandBroker.Dispatch(cmdMsg.GetName(), dappRcvdCmd)

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

//onStreamStop runs when a stream is disconnected
func (n *Node) onStreamStop(stream *Stream) {

	peerInfo := networkmodel.PeerInfo{PeerId: stream.GetPeerId()}
	bytes, err := proto.Marshal(peerInfo.ToProto())

	if err != nil {
		logger.WithError(err).Warn("Node: Marshal peerInfo failed")
	}

	dappCmd := networkmodel.NewDappCmd(TopicOnStreamStop, bytes, false)
	dappCmdCtx := networkmodel.NewDappRcvdCmdContext(dappCmd, n.network.GetHost().GetPeerInfo())

	n.commandBroker.Dispatch(TopicOnStreamStop, dappCmdCtx)
}
