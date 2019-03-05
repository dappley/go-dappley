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
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"
)

const (
	protocalName           = "dappley/1.0.0"
	syncPeerTimeLimitMs    = 1000
	MaxMsgCountBeforeReset = 999999
	maxGetBlocksNum        = 10
	maxSyncPeersCount      = 32
)

var (
	ErrDapMsgNoCmd  = errors.New("command not specified")
	ErrIsInPeerlist = errors.New("peer already exists in peerlist")
)

type NodeConfig struct {
	MaxConnectionOutCount int
	MaxConnectionInCount  int
}

type streamMsg struct {
	msg  *DapMsg
	from peer.ID
}

type Node struct {
	host                   host.Host
	info                   *PeerInfo
	bm                     *core.BlockChainManager
	streamExitCh           chan *Stream
	exitCh                 chan bool
	recentlyRcvedDapMsgs   *sync.Map
	dapMsgBroadcastCounter *uint64
	privKey                crypto.PrivKey
	dispatch               chan *streamMsg
	downloadManager        *DownloadManager
	peerManager            *PeerManager
	isDownBlockChain            chan bool
}

func newMsg(dapMsg *DapMsg, id peer.ID) *streamMsg {
	return &streamMsg{dapMsg, id}
}

//create new Node instance
func NewNode(bc *core.Blockchain, pool *core.BlockPool) *Node {
	return NewNodeWithConfig(bc, pool, nil)
}

func NewNodeWithConfig(bc *core.Blockchain, pool *core.BlockPool, config *NodeConfig) *Node {
	placeholder := uint64(0)
	bm := core.NewBlockChainManager()
	bm.SetblockPool(pool)
	bm.Setblockchain(bc)
	node := &Node{
		host:                   nil,
		info:                   nil,
		bm:                     bm,
		streamExitCh:           make(chan *Stream, 10),
		exitCh:                 make(chan bool, 1),
		recentlyRcvedDapMsgs:   &sync.Map{},
		dapMsgBroadcastCounter: &placeholder,
		privKey:                nil,
		dispatch:               make(chan *streamMsg, 1000),
		downloadManager:        nil,
		peerManager:            nil,
		isDownBlockChain:		make(chan bool),
	}
	node.downloadManager = NewDownloadManager(node)
	node.peerManager = NewPeerManager(node, config)
	return node
}

func (n *Node) isNetworkRadiation(dapmsg DapMsg) bool {
	if _, value := n.recentlyRcvedDapMsgs.Load(dapmsg.GetKey()); value == true {
		return true
	}
	return false
}

func (n *Node) GetBlockchain() *core.Blockchain      { return n.bm.Getblockchain() }
func (n *Node) GetBlockPool() *core.BlockPool        { return n.bm.GetblockPool() }
func (n *Node) GetRecentlyRcvedDapMsgs() *sync.Map   { return n.recentlyRcvedDapMsgs }
func (n *Node) GetDownloadManager() *DownloadManager { return n.downloadManager }
func (n *Node) GetPeerManager() *PeerManager         { return n.peerManager }
func (n *Node) GetInfo() *PeerInfo                   { return n.info }

func (n *Node) Start(listenPort int) error {

	h, addrs, err := createBasicHost(listenPort, n.privKey)
	if err != nil {
		logger.WithError(err).Error("Node: failed to create basic host.")
		return err
	}

	n.host = h
	n.info, err = CreatePeerInfoFromMultiaddrs(addrs)

	//set streamhandler. streamHanlder function is called upon stream connection
	n.host.SetStreamHandler(protocalName, n.streamHandler)
	n.StartRequestLoop()
	n.StartListenLoop()
	n.StartExitListener()

	n.peerManager.Start()
	return err
}

func (n *Node) Stop() {
	n.exitCh <- true
	n.peerManager.StopAllStreams(nil)
	n.host.RemoveStreamHandler(protocalName)
	err := n.host.Close()
	if err != nil {
		logger.WithError(err).Warn("Node: host was not closed properly.")
	}
}

func (n *Node) StartExitListener() {
	go func() {
		for {
			if s, ok := <-n.streamExitCh; ok {
				n.DisconnectPeer(s)
			}
		}
	}()
}

func (n *Node) StartRequestLoop() {

	go func() {
		for {
			select {
			case <-n.exitCh:
				return
			case brPars := <-n.bm.GetblockPool().BlockRequestCh():
				n.RequestBlockUnicast(brPars.BlockHash, brPars.Pid)
			case recv := <-n.isDownBlockChain:
				if recv {
						n.DownloadBlocks(n.bm.Getblockchain())
				}
			}
		}
	}()

}

func (n *Node) StartListenLoop() {
	go func() {
		for {
			if streamMsg, ok := <-n.dispatch; ok {
				n.handle(streamMsg.msg, streamMsg.from)
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
func createBasicHost(listenPort int, priv crypto.PrivKey) (host.Host, []ma.Multiaddr, error) {

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
	}

	if priv != nil {
		opts = append(opts, libp2p.Identity(priv))
	}

	basicHost, err := libp2p.New(context.Background(), opts...)

	if err != nil {
		logger.WithError(err).Error("Node: failed to create a new libp2p node.")
		return nil, nil, err
	}

	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:

	fullAddrs := make([]ma.Multiaddr, len(basicHost.Addrs()))

	for index, addr := range basicHost.Addrs() {
		fullAddr := addr.Encapsulate(hostAddr)
		logger.WithFields(logger.Fields{
			"index":   index,
			"address": fullAddr,
		}).Info("Node: host is up.")

		fullAddrs[index] = fullAddr
	}

	return basicHost, fullAddrs, nil
}

func (n *Node) DisconnectPeer(stream *Stream) {
	n.peerManager.StopStream(stream)

	if n.downloadManager != nil {
		n.downloadManager.DisconnectPeer(stream.peerID)
	}
}

func (n *Node) handle(msg *DapMsg, id peer.ID) {
	switch msg.GetCmd() {
	case GetBlockchainInfo:
		n.GetBlockchainInfoHandler(msg, id)

	case ReturnBlockchainInfo:
		n.ReturnBlockchainInfoHandler(msg, id)

	case SyncBlock:
		n.SyncBlockHandler(msg, id)

	case GetPeerList:
		n.GetNodePeers(msg.GetData(), id)

	case ReturnPeerList:
		n.ReturnNodePeers(msg.GetData(), id)

	case RequestBlock:
		n.SendRequestedBlock(msg.GetData(), id)

	case BroadcastTx:
		n.AddTxToPool(msg)

	case GetBlocks:
		n.GetBlocksHandler(msg, id)

	case ReturnBlocks:
		n.ReturnBlocksHandler(msg, id)

	case GetCommonBlocks:
		n.GetCommonBlocksHandler(msg, id)

	case ReturnCommonBlocks:
		n.ReturnCommonBlocksHandler(msg, id)

	default:
		logger.WithFields(logger.Fields{
			"from": id,
		}).Debug("Node: received an invalid command.")
	}
}

func (n *Node) StartStream(stream *Stream) {
	stream.Start(n.streamExitCh, n.dispatch)
}

func (n *Node) streamHandler(s net.Stream) {
	//start stream

	ns := NewStream(s)
	ns.Start(n.streamExitCh, n.dispatch)
	n.peerManager.AddStream(ns)
}

func (n *Node) GetPeerMultiaddr() []ma.Multiaddr {
	if n.info == nil {
		return nil
	}
	return n.info.Addrs
}

func (n *Node) GetPeerID() peer.ID { return n.info.PeerId }

func (n *Node) RelayDapMsg(dm DapMsg) {
	msgData := dm.ToProto()
	bytes, _ := proto.Marshal(msgData)
	n.peerManager.Broadcast(bytes)
}

func (n *Node) prepareData(msgData proto.Message, cmd string, uniOrBroadcast int, msgKey string) ([]byte, error) {
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
	dm := NewDapmsg(cmd, bytes, msgKey, uniOrBroadcast, n.dapMsgBroadcastCounter)
	if dm.cmd == SyncBlock || dm.cmd == BroadcastTx {
		n.cacheDapMsg(*dm)
	}
	data, err := proto.Marshal(dm.ToProto())
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (n *Node) BroadcastBlock(block *core.Block) error {
	logger.WithFields(logger.Fields{
		"peer_id": n.GetPeerID(),
		"height":  block.GetHeight(),
		"hash":    hex.EncodeToString(block.GetHash()),
	}).Info("Node: is broadcasting a block.")
	data, err := n.prepareData(block.ToProto(), SyncBlock, Broadcast, hex.EncodeToString(block.GetHash()))
	if err != nil {
		return err
	}
	n.peerManager.Broadcast(data)
	return nil
}

func (n *Node) BroadcastGetBlockchainInfo() {
	request := &networkpb.GetBlockchainInfo{Version: protocalName}
	data, err := n.prepareData(request, GetBlockchainInfo, Broadcast, "")
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warn("Node: broadcast GetBlockchainInfo failed.")
	}

	n.peerManager.Broadcast(data)
}

func (n *Node) GetPeerlistBroadcast(maxNum int) error {
	request := &networkpb.GetPeerList{MaxNumber: int32(maxNum)}

	data, err := n.prepareData(request, GetPeerList, Broadcast, "")
	if err != nil {
		return err
	}
	n.peerManager.Broadcast(data)
	return nil
}

func (n *Node) TxBroadcast(tx *core.Transaction) error {
	data, err := n.prepareData(tx.ToProto(), BroadcastTx, Broadcast, hex.EncodeToString(tx.ID))
	if err != nil {
		return err
	}
	n.peerManager.Broadcast(data)
	return nil
}

func (n *Node) SyncPeersBroadcast() error {
	getPeerListPb := &networkpb.GetPeerList{
		MaxNumber: int32(maxSyncPeersCount),
	}
	data, err := n.prepareData(getPeerListPb, GetPeerList, Broadcast, "")
	if err != nil {
		return err
	}
	n.peerManager.Broadcast(data)
	return nil
}

func (n *Node) SendBlockUnicast(block *core.Block, pid peer.ID) error {
	data, err := n.prepareData(block.ToProto(), SyncBlock, Unicast, hex.EncodeToString(block.GetHash()))
	if err != nil {
		return err
	}
	n.peerManager.Unicast(data, pid)
	return nil
}

func (n *Node) SendPeerListUnicast(peers []*PeerInfo, pid peer.ID) error {
	var peerPbs []*networkpb.PeerInfo
	for _, peerInfo := range peers {
		peerPbs = append(peerPbs, peerInfo.ToProto().(*networkpb.PeerInfo))
	}

	data, err := n.prepareData(&networkpb.ReturnPeerList{PeerList: peerPbs}, ReturnPeerList, Unicast, "")
	if err != nil {
		return err
	}
	n.peerManager.Unicast(data, pid)
	return nil
}

func (n *Node) RequestBlockUnicast(hash core.Hash, pid peer.ID) error {
	//build a deppley message

	dm := NewDapmsg(RequestBlock, hash, hex.EncodeToString(hash), Unicast, n.dapMsgBroadcastCounter)
	data, err := proto.Marshal(dm.ToProto())
	if err != nil {
		return err
	}
	n.peerManager.Unicast(data, pid)
	return nil
}

func (n *Node) GetCommonBlocksUnicast(blockHeaders []*SyncCommandBlocksHeader, pid peer.ID, msgId int32) error {
	var blockHeaderPbs []*corepb.BlockHeader
	for _, blockHeader := range blockHeaders {
		blockHeaderPbs = append(blockHeaderPbs,
			&corepb.BlockHeader{Hash: blockHeader.hash, Height: blockHeader.height})
	}

	getCommonBlocksPb := &networkpb.GetCommonBlocks{MsgId: msgId, BlockHeaders: blockHeaderPbs}

	data, err := n.prepareData(getCommonBlocksPb, GetCommonBlocks, Unicast, "")
	if err != nil {
		return nil
	}

	n.peerManager.Unicast(data, pid)
	return nil
}

func (n *Node) DownloadBlocksUnicast(hashes []core.Hash, pid peer.ID) error {
	blkHashes := make([][]byte, len(hashes))
	for index, hash := range hashes {
		blkHashes[index] = hash
	}

	getBlockPb := &networkpb.GetBlocks{StartBlockHashes: blkHashes}

	data, err := n.prepareData(getBlockPb, GetBlocks, Unicast, "")
	if err != nil {
		return nil
	}

	n.peerManager.Unicast(data, pid)
	return nil
}

func (n *Node) addBlockToPool(block *core.Block, pid peer.ID) {
	//add block to blockpool. Make sure this is none blocking.
	p := false
	n.bm.Push(block, pid, &p)
	if p {
			n.isDownBlockChain <- true

	}
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

func (n *Node) SyncBlockHandler(dm *DapMsg, pid peer.ID) {
	if len(dm.data) == 0 {
		logger.WithFields(logger.Fields{
			"cmd": "sync block",
		}).Warn("Node: can not find block information.")
		return
	}

	if dm.uniOrBroadcast == Broadcast {
		if n.isNetworkRadiation(*dm) {
			return
		}
		n.cacheDapMsg(*dm)
		blk := n.getFromProtoBlockMsg(dm.GetData())
		n.addBlockToPool(blk, pid)
		if dm.uniOrBroadcast == Broadcast {
			n.RelayDapMsg(*dm)
		}
	} else {
		blk := n.getFromProtoBlockMsg(dm.GetData())
		n.addBlockToPool(blk, pid)
	}
}

func (n *Node) GetBlockchainInfoHandler(dm *DapMsg, pid peer.ID) {
	tailBlock, err := n.GetBlockchain().GetTailBlock()
	if err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "GetBlockchainInfoRequest",
		}).Warn("Node: get  tail block failed.")
		return
	}

	response := &networkpb.ReturnBlockchainInfo{
		TailBlockHash: n.GetBlockchain().GetTailBlockHash(),
		BlockHeight:   n.GetBlockchain().GetMaxHeight(),
		Timestamp:     tailBlock.GetTimestamp(),
	}

	data, err := n.prepareData(response, ReturnBlockchainInfo, Unicast, "")
	if err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "GetBlockchainInfoRequest",
		}).Warn("Node: prepare data failed.")
		return
	}

	n.peerManager.Unicast(data, pid)
}

func (n *Node) ReturnBlockchainInfoHandler(dm *DapMsg, pid peer.ID) {
	blockchainInfo := &networkpb.ReturnBlockchainInfo{}
	if err := proto.Unmarshal(dm.data, blockchainInfo); err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "ReturnBlockchainInfo",
		}).Info("Node: parse data failed.")
		return
	}

	n.downloadManager.AddPeerBlockChainInfo(pid, blockchainInfo.GetBlockHeight())
}

//TODO  Refactor getblocks in rpcService and node
func (n *Node) GetBlocksHandler(dm *DapMsg, pid peer.ID) {
	param := &networkpb.GetBlocks{}
	if err := proto.Unmarshal(dm.data, param); err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "GetBlocks",
		}).Info("Node: parse data failed.")
		return
	}

	block := n.findBlockInRequestHash(param.GetStartBlockHashes())

	// Reach the blockchain's tail
	if block.GetHeight() >= n.GetBlockchain().GetMaxHeight() {
		logger.WithFields(logger.Fields{
			"cmd": "GetBlocks",
		}).Info("Node: reach blockchain tail.")
		return
	}

	var blocks []*core.Block

	block, err := n.GetBlockchain().GetBlockByHeight(block.GetHeight() + 1)
	for i := int32(0); i < maxGetBlocksNum && err == nil; i++ {
		if block.GetHeight() == 0 {
			logger.Panicf("Error %v", hex.EncodeToString(block.GetHash()))
		}
		blocks = append(blocks, block)
		block, err = n.GetBlockchain().GetBlockByHeight(block.GetHeight() + 1)
	}

	var blockPbs []*corepb.Block
	for i := len(blocks) - 1; i >= 0; i-- {
		blockPbs = append(blockPbs, blocks[i].ToProto().(*corepb.Block))
	}

	result := &networkpb.ReturnBlocks{Blocks: blockPbs, StartBlockHashes: param.GetStartBlockHashes()}

	data, err := n.prepareData(result, ReturnBlocks, Unicast, "")
	if err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "GetBlocks",
		}).Warn("Node: prepare data failed.")
		return
	}

	n.peerManager.Unicast(data, pid)
}

func (n *Node) GetCommonBlocksHandler(dm *DapMsg, pid peer.ID) {
	param := &networkpb.GetCommonBlocks{}
	if err := proto.Unmarshal(dm.data, param); err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "GetCommonBlocks",
		}).Info("Node: parse data failed.")
		return
	}

	index, _ := n.downloadManager.FindCommonBlock(param.GetBlockHeaders())
	var blockHeaderPbs []*corepb.BlockHeader
	if index == 0 {
		blockHeaderPbs = param.GetBlockHeaders()[:1]
	} else {
		blockHeaders := n.downloadManager.GetCommonBlockCheckPoint(
			param.GetBlockHeaders()[index].GetHeight(),
			param.GetBlockHeaders()[index-1].GetHeight(),
		)
		for _, blockHeader := range blockHeaders {
			blockHeaderPbs = append(blockHeaderPbs,
				&corepb.BlockHeader{Hash: blockHeader.hash, Height: blockHeader.height})
		}
	}

	result := &networkpb.ReturnCommonBlocks{MsgId: param.GetMsgId(), BlockHeaders: blockHeaderPbs}

	data, err := n.prepareData(result, ReturnCommonBlocks, Unicast, "")
	if err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "GetBlocks",
		}).Warn("Node: prepare data failed.")
		return
	}

	n.peerManager.Unicast(data, pid)
}

func (n *Node) findBlockInRequestHash(startBlockHashes [][]byte) *core.Block {
	for _, hash := range startBlockHashes {
		// hash in blockchain, return
		if block, err := n.GetBlockchain().GetBlockByHash(hash); err == nil {
			return block
		}
	}

	// Return Genesis Block
	block, _ := n.GetBlockchain().GetBlockByHeight(0)
	return block
}

func (n *Node) ReturnBlocksHandler(dm *DapMsg, pid peer.ID) {
	param := &networkpb.ReturnBlocks{}
	if err := proto.Unmarshal(dm.data, param); err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "ReturnBlocks",
		}).Info("Node: parse data failed.")
		return
	}

	n.downloadManager.GetBlocksDataHandler(param, pid)
}

func (n *Node) ReturnCommonBlocksHandler(dm *DapMsg, pid peer.ID) {
	param := &networkpb.ReturnCommonBlocks{}

	if err := proto.Unmarshal(dm.data, param); err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "ReturnCommonBlocks",
		}).Info("Node: parse data failed.")
	}

	n.downloadManager.GetCommonBlockDataHandler(param, pid)
}

func (n *Node) cacheDapMsg(dm DapMsg) {
	n.recentlyRcvedDapMsgs.Store(dm.GetKey(), true)
}

func (n *Node) AddTxToPool(dm *DapMsg) {
	if n.GetBlockchain().GetState() != core.BlockchainReady {
		return
	}

	if n.isNetworkRadiation(*dm) {
		return
	}

	n.RelayDapMsg(*dm)
	n.cacheDapMsg(*dm)

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
	utxoIndex.UpdateUtxoState(n.GetBlockchain().GetTxPool().GetTransactions())

	if tx.Verify(utxoIndex, 0) == false {
		logger.Info("Node: broadcast transaction verify failed.")
		return
	}

	if tx.IsFromContract() {
		return
	}

	n.bm.Getblockchain().GetTxPool().Push(*tx)
}

func (n *Node) GetNodePeers(data []byte, pid peer.ID) {
	getPeerlistRequest := &networkpb.GetPeerList{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(data, getPeerlistRequest); err != nil {
		logger.WithError(err).Warn("Node: parse GetPeerList failed.")
	}

	peers := n.peerManager.RandomGetConnectedPeers(int(getPeerlistRequest.GetMaxNumber()))
	n.SendPeerListUnicast(peers, pid)
}

func (n *Node) ReturnNodePeers(data []byte, pid peer.ID) {
	peerlistPb := &networkpb.ReturnPeerList{}

	if err := proto.Unmarshal(data, peerlistPb); err != nil {
		logger.WithError(err).Warn("Node: parse Peerlist failed.")
	}

	var peers []*PeerInfo
	for _, peerPb := range peerlistPb.GetPeerList() {
		peerInfo := &PeerInfo{}
		if err := peerInfo.FromProto(peerPb); err != nil {
			logger.WithError(err).Warn("Node: parse PeerInfo failed.")
		}
		peers = append(peers, peerInfo)
	}

	n.peerManager.ReceivePeers(pid, peers)
}

func (n *Node) SendRequestedBlock(hash []byte, pid peer.ID) {
	blockBytes, err := n.bm.Getblockchain().GetDb().Get(hash)
	if err != nil {
		logger.WithError(err).Warn("Node: failed to get the requested block from database.")
		return
	}
	block := core.Deserialize(blockBytes)
	n.SendBlockUnicast(block, pid)
}

func (n *Node) DownloadBlocks(bc *core.Blockchain) {
	downloadManager := n.GetDownloadManager()
	finishChan := make(chan bool, 1)

	bc.SetState(core.BlockchainDownloading)
	downloadManager.StartDownloadBlockchain(finishChan)
	<-finishChan
	bc.SetState(core.BlockchainReady)
}
