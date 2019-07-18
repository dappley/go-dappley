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
	"encoding/hex"
	"errors"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
	"io/ioutil"
)

const (
	maxSyncPeersCount = 32

	SyncPeerList = "SyncPeerList"

	BroadcastTx       = "BroadcastTx"
	BroadcastBatchTxs = "BraodcastBatchTxs"

	TopicOnStreamStop = "TopicOnStreamStop"

	Unicast       = false
	Broadcast     = true
	dispatchChLen = 1024 * 4
	requestChLen  = 1024
)

const (
	SyncBlockPriority         = HighPriorityCommand
	BroadcastTxPriority       = NormalPriorityCommand
	BroadcastBatchTxsPriority = NormalPriorityCommand
)

var (
	ErrDapMsgNoCmd = errors.New("command not specified")
)

var (
	reservedTopics = []string{
		TopicOnStreamStop,
	}
)

type Node struct {
	network       *Network
	bm            *core.BlockChainManager
	exitCh        chan bool
	privKey       crypto.PrivKey
	dispatcher    chan *DappPacketContext
	commandSendCh chan *DappSendCmdContext
	commandBroker *CommandBroker
}

//create new Node instance
func NewNode(bm *core.BlockChainManager) *Node {
	return NewNodeWithConfig(bm, nil)
}

func NewNodeWithConfig(bm *core.BlockChainManager, config *NodeConfig) *Node {
	var err error
	var db storage.Storage

	node := &Node{
		bm:            bm,
		exitCh:        make(chan bool, 1),
		privKey:       nil,
		dispatcher:    make(chan *DappPacketContext, dispatchChLen),
		commandSendCh: make(chan *DappSendCmdContext, requestChLen),
		commandBroker: NewCommandBroker(reservedTopics),
	}

	if bm != nil && bm.Getblockchain() != nil {
		db = node.bm.Getblockchain().GetDb()
	}

	node.network = NewNetwork(config, node.dispatcher, node.commandSendCh, db)
	node.network.OnStreamStop(node.OnStreamStop)
	node.network.Subscirbe(node.commandBroker)

	if err != nil {
		logger.WithError(err).Panic("Node: Can not initialize lru cache for recentlyRcvdDapMsgs!")
	}

	return node
}

func (n *Node) GetBlockchain() *core.Blockchain               { return n.bm.Getblockchain() }
func (n *Node) GetInfo() *PeerInfo                            { return n.network.host.info }
func (n *Node) GetNetwork() *Network                          { return n.network }
func (n *Node) GetCommandSendCh() chan *DappSendCmdContext    { return n.commandSendCh }
func (n *Node) GetCommandBroker() *CommandBroker              { return n.commandBroker }
func (n *Node) GetBlockchainManager() *core.BlockChainManager { return n.bm }

func (n *Node) Start(listenPort int, seeds []string) error {
	err := n.network.Start(listenPort, n.privKey, seeds)
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

				if len(n.dispatcher) == dispatchChLen {
					logger.WithFields(logger.Fields{
						"lenOfDispatchChan": len(n.dispatcher),
					}).Warn("Node: streamMsgDispatcherCh channel full")
				}
				cmdMsg := ParseDappMsgFromDappPacket(streamMsg.packet)
				n.handle(cmdMsg, streamMsg.source)
				dappRcvdCmd := NewDappRcvdCmdContext(cmdMsg, streamMsg.source)
				err := n.commandBroker.Dispatch(dappRcvdCmd)

				if err != nil {
					logger.WithError(err).Warn("Node: Dispatch received message failed")
				}
			}
		}
	}()

}

//LoadNetworkKeyFromFile reads the network privatekey source a file
func (n *Node) LoadNetworkKeyFromFile(filePath string) error {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	data, err := base64.StdEncoding.DecodeString(string(bytes))
	if err != nil {
		return err
	}

	n.privKey, err = crypto.UnmarshalPrivateKey(data)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) OnStreamStop(stream *Stream) {

	peerInfo := PeerInfo{PeerId: stream.peerID}
	bytes, err := proto.Marshal(peerInfo.ToProto())

	logger.WithError(err).Warn("Node: Marshal peerInfo failed")

	dappCmd := NewDapCmd(TopicOnStreamStop, bytes, false)
	dappCmdCtx := NewDappRcvdCmdContext(dappCmd, n.network.host.ID())

	n.commandBroker.Dispatch(dappCmdCtx)
}

func (n *Node) handle(msg *DappCmd, id peer.ID) {
	logger.WithFields(logger.Fields{
		"name": msg.GetName(),
	}).Info("Node: Received command")

	switch msg.GetName() {
	case BroadcastTx:
		n.AddTxToPool(msg)

	case BroadcastBatchTxs:
		n.AddBatchTxsToPool(msg)

	default:
		logger.WithFields(logger.Fields{
			"source": id,
		}).Debug("Node: received an invalid command.")
	}
}

func (n *Node) GetPeerMultiaddr() []ma.Multiaddr {
	if n.GetInfo() == nil {
		return nil
	}
	return n.GetInfo().Addrs
}

func (n *Node) GetPeerID() peer.ID { return n.GetInfo().PeerId }

func (n *Node) RelayDapMsg(dm DappCmd, priority DappCmdPriority) {
	msgData := dm.ToProto()
	bytes, _ := proto.Marshal(msgData)
	n.network.Broadcast(bytes, priority)
}

func (n *Node) prepareData(msgData proto.Message, cmd string, isBroadcast bool, msgKey string) ([]byte, error) {
	if cmd == "" {
		return nil, ErrDapMsgNoCmd
	}

	bytes := []byte{}
	var err error

	if msgData != nil {
		//marshal the block to wire format
		bytes, err = proto.Marshal(msgData)
		if err != nil {
			return nil, err
		}
	}

	//build a dappley message
	dm := NewDapCmd(cmd, bytes, isBroadcast)

	data, err := proto.Marshal(dm.ToProto())
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (n *Node) TxBroadcast(tx *core.Transaction) error {
	data, err := n.prepareData(tx.ToProto(), BroadcastTx, Broadcast, hex.EncodeToString(tx.ID))
	if err != nil {
		return err
	}
	n.network.Broadcast(data, BroadcastTxPriority)
	return nil
}

func (n *Node) BatchTxBroadcast(txs []core.Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	transactions := core.NewTransactions(txs)

	data, err := n.prepareData(transactions.ToProto(), BroadcastBatchTxs, Broadcast, hex.EncodeToString(txs[0].ID))
	if err != nil {
		return err
	}
	n.network.Broadcast(data, BroadcastBatchTxsPriority)
	return nil
}

func (n *Node) AddTxToPool(dm *DappCmd) {
	if n.GetBlockchain().GetState() != core.BlockchainReady {
		return
	}

	n.RelayDapMsg(*dm, BroadcastTxPriority)

	txpb := &corepb.Transaction{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(dm.GetData(), txpb); err != nil {
		logger.Warn(err)
	}

	//create an empty tx
	tx := &core.Transaction{}

	//load the tx with proto
	tx.FromProto(txpb)
	//add tx to txpool
	utxoIndex := core.NewUTXOIndex(n.GetBlockchain().GetUtxoCache())

	if tx.IsFromContract(utxoIndex) {
		return
	}

	n.bm.Getblockchain().GetTxPool().Push(*tx)
}

func (n *Node) AddBatchTxsToPool(dm *DappCmd) {
	if n.GetBlockchain().GetState() != core.BlockchainReady {
		return
	}

	n.RelayDapMsg(*dm, BroadcastBatchTxsPriority)

	txspb := &corepb.Transactions{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(dm.GetData(), txspb); err != nil {
		logger.Warn(err)
	}

	//create an empty tx
	txs := &core.Transactions{}

	//load the tx with proto
	txs.FromProto(txspb)
	//add tx to txpool
	utxoIndex := core.NewUTXOIndex(n.GetBlockchain().GetUtxoCache())

	for _, tx := range txs.GetTransactions() {
		if tx.IsFromContract(utxoIndex) {
			continue
		}
		n.bm.Getblockchain().GetTxPool().Push(tx)
	}

}
