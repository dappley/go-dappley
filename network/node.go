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
	"context"
	"fmt"
	"math/rand"
	"time"

	"errors"
	"strconv"

	"encoding/base64"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
	"io/ioutil"
	"sync"
)

const (
	protocalName           = "dappley/1.0.0"
	syncPeerTimeLimitMs    = 1000
	MaxMsgCountBeforeReset = 999999
)

var (
	ErrDapMsgNoCmd  = errors.New("ERROR: Dappley message has no command input")
	ErrIsInPeerlist = errors.New("ERROR: Peer already exists in peerlist")
)

type Node struct {
	host                   host.Host
	info                   *Peer
	bc                     *core.Blockchain
	streams                map[peer.ID]*Stream
	peerList               *PeerList
	exitCh                 chan bool
	recentlyRcvedDapMsgs   *sync.Map
	dapMsgBroadcastCounter *uint64
	privKey                crypto.PrivKey
}

//create new Node instance
func NewNode(bc *core.Blockchain) *Node {
	placeholder := uint64(0)
	return &Node{nil,
		nil,
		bc,
		make(map[peer.ID]*Stream, 10),
		NewPeerList(nil),
		make(chan bool, 1),
		&sync.Map{},
		&placeholder,
		nil,
	}
}

func (n *Node) isNetworkRadiation(dapmsg DapMsg) bool {
	if _, value := n.recentlyRcvedDapMsgs.Load(dapmsg.GetKey()); value == true {
		return true
	}
	return false
}

func (n *Node) GetBlockchain() *core.Blockchain    { return n.bc }
func (n *Node) GetPeerList() *PeerList             { return n.peerList }
func (n *Node) GetRecentlyRcvedDapMsgs() *sync.Map { return n.recentlyRcvedDapMsgs }

func (n *Node) Start(listenPort int) error {

	h, addr, err := createBasicHost(listenPort, n.privKey)
	if err != nil {
		return err
	}

	n.host = h
	n.info, err = CreatePeerFromMultiaddr(addr)

	//set streamhandler. streamHanlder function is called upon stream connection
	n.host.SetStreamHandler(protocalName, n.streamHandler)
	n.StartRequestLoop()
	return err
}

func (n *Node) StartRequestLoop() {

	go func() {
		for {
			select {
			case <-n.exitCh:
				return
			case brPars := <-n.bc.GetBlockPool().BlockRequestCh():
				n.RequestBlockUnicast(brPars.BlockHash, brPars.Pid)
			}
		}
	}()

}

//LoadNetworkKeyFromFile reads the network privatekey from a file
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

//create basic host. Returns host object, host address and error
func createBasicHost(listenPort int, priv crypto.PrivKey) (host.Host, ma.Multiaddr, error) {

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
	}

	if priv != nil {
		opts = append(opts, libp2p.Identity(priv))
	}

	basicHost, err := libp2p.New(context.Background(), opts...)

	if err != nil {
		return nil, nil, err
	}

	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	addr := basicHost.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	logger.Info("Full Address is ", fullAddr)

	return basicHost, fullAddr, nil
}

func (n *Node) AddStreamByString(targetFullAddr string) error {
	addr, err := ma.NewMultiaddr(targetFullAddr)
	if err != nil {
		return err
	}
	return n.AddStreamMultiAddr(addr)
}

//AddStreamMultiAddr stream to the targetFullAddr address. If the targetFullAddr is nil, the node goes to listening mode
func (n *Node) AddStreamMultiAddr(targetFullAddr ma.Multiaddr) error {

	//If there is a target address, connect to that address
	if targetFullAddr != nil {

		peerInfo, err := CreatePeerFromMultiaddr(targetFullAddr)
		if err != nil {
			return err
		}

		//Add Stream
		n.AddStream(peerInfo.peerid, peerInfo.addr)
	}

	return nil
}

func (n *Node) AddStream(peerid peer.ID, targetAddr ma.Multiaddr) error {
	// We have a peer ID and a targetAddr so we add it to the peerstore
	// so LibP2P knows how to contact it
	p := Peer{peerid, targetAddr}
	if n.peerList.IsInPeerlist(&p) {
		logger.Debug(targetAddr.String() + " is already in peerlist of " + n.GetPeerMultiaddr().String())
		return ErrIsInPeerlist
	}

	n.host.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)

	// make a new stream
	stream, err := n.host.NewStream(context.Background(), peerid, protocalName)
	if err != nil {
		return err
	}
	// Create a buffered stream so that read and write are non blocking.
	n.streamHandler(stream)

	// Add the peer list
	if len(n.peerList.peers) >= PEERLISTMAXSIZE {
		n.peerList.RemoveOneIP(&Peer{peerid, targetAddr})
	}
	n.peerList.Add(&Peer{peerid, targetAddr})

	return nil
}

func (n *Node) streamHandler(s net.Stream) {
	// Create a buffer stream for non blocking read and write.
	logger.Info(n.GetPeerMultiaddr(), " Connected Stream to Peer Addr:", s.Conn().RemoteMultiaddr())

	peer := &Peer{s.Conn().RemotePeer(), s.Conn().RemoteMultiaddr()}
	if !n.peerList.ListIsFull() && !n.peerList.IsInPeerlist(peer) {
		n.peerList.Add(peer)
		//start stream
		ns := NewStream(s, n)
		n.streams[s.Conn().RemotePeer()] = ns
		ns.Start()

		n.SyncPeersUnicast(peer.peerid)
	}
}

func (n *Node) GetInfo() *Peer { return n.info }

func (n *Node) GetPeerMultiaddr() ma.Multiaddr {
	if n.info == nil {
		return nil
	}
	return n.info.addr
}

func (n *Node) GetPeerID() peer.ID { return n.info.peerid }

func (n *Node) RelayDapMsg(dm DapMsg) {
	msgData := dm.ToProto()
	bytes, _ := proto.Marshal(msgData)
	n.broadcast(bytes)
}

func (n *Node) prepareData(msgData proto.Message, cmd string, uniOrBroadcast int) ([]byte, error) {
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
	dm := NewDapmsg(cmd, bytes, n.info.peerid.String()+strconv.FormatUint(*n.dapMsgBroadcastCounter, 10), uniOrBroadcast, n.dapMsgBroadcastCounter)
	if dm.cmd == SyncBlock {
		logger.Debug("Node: ",n.info.peerid," broadcasting block with key ", dm.key)
		n.cacheDapMsg(*dm)
	}
	data, err := proto.Marshal(dm.ToProto())
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (n *Node) BroadcastBlock(block *core.Block) error {
	logger.Debug("Node: BroadcastBlock: Hash:", block.GetHash(), ", Height:", block.GetHeight())
	data, err := n.prepareData(block.ToProto(), SyncBlock, Broadcast)
	if err != nil {
		return err
	}
	n.broadcast(data)
	return nil
}

func (n *Node) SyncPeersBroadcast() error {
	data, err := n.prepareData(n.peerList.ToProto(), SyncPeerList, Broadcast)
	if err != nil {
		return err
	}
	n.broadcast(data)
	return nil
}

func (n *Node) TxBroadcast(tx *core.Transaction) error {
	data, err := n.prepareData(tx.ToProto(), BroadcastTx, Broadcast)
	if err != nil {
		return err
	}
	n.broadcast(data)
	return nil
}

func (n *Node) SyncPeersUnicast(pid peer.ID) error {
	data, err := n.prepareData(n.peerList.ToProto(), SyncPeerList, Unicast)
	if err != nil {
		return err
	}
	n.unicast(data, pid)
	return nil
}

func (n *Node) BroadcastTxCmd(txn *core.Transaction) error {
	data, err := n.prepareData(txn.ToProto(), BroadcastTx, Broadcast)
	if err != nil {
		return err
	}
	n.broadcast(data)
	return nil
}

func (n *Node) SendBlockUnicast(block *core.Block, pid peer.ID) error {
	data, err := n.prepareData(block.ToProto(), SyncBlock, Unicast)
	if err != nil {
		return err
	}
	n.unicast(data, pid)
	return nil
}

func (n *Node) RequestBlockUnicast(hash core.Hash, pid peer.ID) error {
	//build a deppley message

	dm := NewDapmsg(RequestBlock, hash, n.info.peerid.String()+strconv.FormatUint(*n.dapMsgBroadcastCounter, 10), Unicast, n.dapMsgBroadcastCounter)
	data, err := proto.Marshal(dm.ToProto())
	if err != nil {
		return err
	}
	n.unicast(data, pid)
	return nil
}

//broadcast data
func (n *Node) broadcast(data []byte) {
	for _, s := range n.streams {
		s.Send(data)
	}
}

//unicast data
func (n *Node) unicast(data []byte, pid peer.ID) {
	n.streams[pid].Send(data)
}

func (n *Node) addBlockToPool(block *core.Block, pid peer.ID) {
	//add block to blockpool. Make sure this is none blocking.
	n.bc.GetBlockPool().Push(block, pid)
}

func (n *Node) getFromProtoBlockMsg(data []byte) *core.Block {
	//create a block proto
	blockpb := &corepb.Block{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(data, blockpb); err != nil {
		logger.Warn(err)
	}

	//create an empty block
	block := &core.Block{}

	//load the block with proto
	block.FromProto(blockpb)

	return block
}
func (n *Node) syncBlockHandler(dm *DapMsg, pid peer.ID) {
	if n.isNetworkRadiation(*dm) {
		logger.Debug("Node: ", n.GetPeerID() ," Already received ", dm.GetKey(), " before")
		return
	}

	n.RelayDapMsg(*dm)
	n.cacheDapMsg(*dm)
	blk := n.getFromProtoBlockMsg(dm.GetData())
	logger.Debug("Node: ", n.GetPeerID() ," Received Block: Hash:", blk.GetHash(), ", Height:", blk.GetHeight())

	n.addBlockToPool(blk, pid)
}

func (n *Node) cacheDapMsg(dm DapMsg) {
	n.recentlyRcvedDapMsgs.Store(dm.GetKey(), true)
}

func (n *Node) addTxToPool(data []byte) {

	//create a block proto
	txpb := &corepb.Transaction{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(data, txpb); err != nil {
		logger.Warn(err)
	}

	//create an empty tx
	tx := &core.Transaction{}

	//load the tx with proto
	tx.FromProto(txpb)
	//add tx to txpool
	n.bc.GetTxPool().ConditionalAdd(*tx)
}

func (n *Node) addMultiPeers(data []byte) {

	go func() {
		//create a peerList proto
		plpb := &networkpb.Peerlist{}

		//unmarshal byte to proto
		if err := proto.Unmarshal(data, plpb); err != nil {
			logger.Warn(err)
		}

		//create an empty peerList
		pl := &PeerList{}

		//load the block with proto
		pl.FromProto(plpb)

		//remove the node's own peer info from the list
		newpl := &PeerList{[]*Peer{n.info}}
		newpl = newpl.FindNewPeers(pl)
		//find the new added peers
		newpl = n.peerList.FindNewPeers(newpl)

		//wait for random time within the time limit
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(syncPeerTimeLimitMs)))

		//add streams for new peers
		for _, p := range newpl.GetPeerlist() {
			if !n.peerList.IsInPeerlist(p) && p.peerid != n.info.peerid {
				n.AddStream(p.peerid, p.addr)
			}
		}

		//add peers
		n.peerList.MergePeerlist(newpl)
	}()
}

func (n *Node) sendRequestedBlock(hash []byte, pid peer.ID) {
	blockBytes, err := n.bc.GetDb().Get(hash)
	if err != nil {
		logger.Warn("Unable to get block data. Block request failed")
		return
	}
	block := core.Deserialize(blockBytes)
	n.SendBlockUnicast(block, pid)
}
