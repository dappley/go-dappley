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
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
	"io/ioutil"
)

const (
	TopicOnStreamStop = "TopicOnStreamStop"

	Unicast       = false
	Broadcast     = true
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
	exitCh        chan bool
	dispatcher    chan *DappPacketContext
	commandSendCh chan *DappSendCmdContext
	commandBroker *CommandBroker
}

//create new Node instance
func NewNode(db Storage) *Node {
	return NewNodeWithConfig(db, nil)
}

func NewNodeWithConfig(db Storage, config *NodeConfig) *Node {
	var err error

	node := &Node{
		exitCh:        make(chan bool, 1),
		dispatcher:    make(chan *DappPacketContext, dispatchChLen),
		commandSendCh: make(chan *DappSendCmdContext, requestChLen),
		commandBroker: NewCommandBroker(reservedTopics),
	}

	node.network = NewNetwork(config, node.dispatcher, node.commandSendCh, db)
	node.network.OnStreamStop(node.OnStreamStop)
	node.network.Subscirbe(node.commandBroker)

	if err != nil {
		logger.WithError(err).Panic("Node: Can not initialize lru cache for recentlyRcvdDapMsgs!")
	}

	return node
}

func (n *Node) GetInfo() *PeerInfo                         { return n.network.host.info }
func (n *Node) GetNetwork() *Network                       { return n.network }
func (n *Node) GetCommandSendCh() chan *DappSendCmdContext { return n.commandSendCh }
func (n *Node) GetCommandBroker() *CommandBroker           { return n.commandBroker }

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

func (n *Node) Stop() {
	n.exitCh <- true
	n.network.Stop()
}

func (n *Node) StartRequestLoop() {

	go func() {
		for {
			select {
			case <-n.exitCh:
				return
			case cmdCtx := <-n.commandSendCh:
				if cmdCtx.command == nil {
					continue
				}

				rawBytes := cmdCtx.command.GetRawBytes()

				if cmdCtx.IsBroadcast() {
					n.GetNetwork().Broadcast(rawBytes, cmdCtx.priority)
				} else {
					n.GetNetwork().Unicast(rawBytes, cmdCtx.destination, cmdCtx.priority)
				}

			}
		}
	}()
}

func (n *Node) StartListenLoop() {
	go func() {
		for {
			if streamMsg, ok := <-n.dispatcher; ok {

				cmdMsg := ParseDappMsgFromDappPacket(streamMsg.packet)
				dappRcvdCmd := NewDappRcvdCmdContext(cmdMsg, streamMsg.source)
				err := n.commandBroker.Dispatch(dappRcvdCmd)
				if err != nil {
					logger.WithError(err).Warn("Node: Dispatch received message failed")
				}
			}
		}
	}()

}

func (n *Node) GetPeerMultiaddr() []ma.Multiaddr {
	if n.GetInfo() == nil {
		return nil
	}
	return n.GetInfo().Addrs
}

func (n *Node) GetPeerID() peer.ID { return n.GetInfo().PeerId }

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

func (n *Node) OnStreamStop(stream *Stream) {

	peerInfo := PeerInfo{PeerId: stream.peerID}
	bytes, err := proto.Marshal(peerInfo.ToProto())

	logger.WithError(err).Warn("Node: Marshal peerInfo failed")

	dappCmd := NewDapCmd(TopicOnStreamStop, bytes, false)
	dappCmdCtx := NewDappRcvdCmdContext(dappCmd, n.network.host.ID())

	n.commandBroker.Dispatch(dappCmdCtx)
}
