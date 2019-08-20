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
	"github.com/dappley/go-dappley/network/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/core"
	corepb "github.com/dappley/go-dappley/core/pb"
	networkpb "github.com/dappley/go-dappley/network/pb"
)

const (
	maxGetBlocksNum      = 10
	maxSyncPeersCount    = 32
	GetBlockchainInfo    = "GetBlockchainInfo"
	ReturnBlockchainInfo = "ReturnGetBlockchainInfo"
	SyncBlock            = "SyncBlock"
	GetBlocks            = "GetBlocks"
	ReturnBlocks         = "ReturnBlocks"
	SyncPeerList         = "SyncPeerList"
	GetPeerList          = "GetPeerList"
	ReturnPeerList       = "ReturnPeerList"
	RequestBlock         = "requestBlock"
	BroadcastTx          = "BroadcastTx"
	BroadcastBatchTxs    = "BraodcastBatchTxs"
	GetCommonBlocks      = "GetCommonBlocks"
	ReturnCommonBlocks   = "ReturnCommonBlocks"
	Unicast              = 0
	Broadcast            = 1
	dispatchChLen        = 1024 * 4
)

const (
	GetBlockchainInfoPriority    = NormalPriorityCommand
	ReturnBlockchainInfoPriority = NormalPriorityCommand
	SyncBlockPriority            = HighPriorityCommand
	GetBlocksPriority            = HighPriorityCommand
	ReturnBlocksPriority         = HighPriorityCommand
	SyncPeerListPriority         = HighPriorityCommand
	GetPeerListPriority          = HighPriorityCommand
	ReturnPeerListPriority       = HighPriorityCommand
	RequestBlockPriority         = HighPriorityCommand
	BroadcastTxPriority          = NormalPriorityCommand
	BroadcastBatchTxsPriority    = NormalPriorityCommand
	GetCommonBlocksPriority      = HighPriorityCommand
	ReturnCommonBlocksPriority   = HighPriorityCommand
)

var (
	ErrDapMsgNoCmd = errors.New("command not specified")
)

type Node struct {
	network           *Network
	bm                *core.BlockChainManager
	exitCh            chan bool
	privKey           crypto.PrivKey
	dispatcher        chan *StreamMsg
	downloadManager   *DownloadManager
	messageDispatcher *MessageDispatcher
}

//create new Node instance
func NewNode(bc *core.Blockchain, pool *core.BlockPool) *Node {
	return NewNodeWithConfig(bc, pool, nil)
}

func NewNodeWithConfig(bc *core.Blockchain, pool *core.BlockPool, config *NodeConfig) *Node {
	var err error
	var db storage.Storage

	bm := core.NewBlockChainManager()
	bm.SetblockPool(pool)
	bm.Setblockchain(bc)
	node := &Node{
		bm:              bm,
		exitCh:          make(chan bool, 1),
		privKey:         nil,
		dispatcher:      make(chan *StreamMsg, dispatchChLen),
		downloadManager: nil,
	}
	node.messageDispatcher = NewMessageDispatcher()

	if bc != nil {
		db = node.bm.Getblockchain().GetDb()
	}

	node.network = NewNetwork(config, node.dispatcher, db)
	node.network.OnStreamStop(node.OnStreamStop)

	if err != nil {
		logger.WithError(err).Panic("Node: Can not initialize lru cache for recentlyRcvdDapMsgs!")
	}
	node.downloadManager = NewDownloadManager(node)
	return node
}

func (n *Node) GetBlockchain() *core.Blockchain      { return n.bm.Getblockchain() }
func (n *Node) GetBlockPool() *core.BlockPool        { return n.bm.GetblockPool() }
func (n *Node) GetDownloadManager() *DownloadManager { return n.downloadManager }
func (n *Node) GetPeerManager() *PeerManager         { return n.network.peerManager }
func (n *Node) GetInfo() *PeerInfo                   { return n.network.host.info }

func (n *Node) Start(listenPort int) error {
	err := n.network.Start(listenPort, n.privKey)
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
			case brPars := <-n.bm.GetblockPool().BlockRequestCh():
				n.RequestBlockUnicast(brPars.BlockHash, brPars.Pid)
			case <-n.bm.GetblockPool().DownloadBlocksCh():
				go n.DownloadBlocks(n.bm.Getblockchain())
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
					}).Warn("Node: dispatcher channel full")
				}
				cmdMsg := ParseDappMsgFromDappPacket(streamMsg.msg)
				n.handle(cmdMsg, streamMsg.source)
				n.messageDispatcher.Dispatch(cmdMsg.GetCmd(), streamMsg.msg.data)
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

func (n *Node) Subscribe(cmd string, dispatcher chan []byte) error {
	return n.messageDispatcher.Subscribe(cmd, dispatcher)
}

func (n *Node) OnStreamStop(stream *Stream) {
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

	case BroadcastBatchTxs:
		n.AddBatchTxsToPool(msg)

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

func (n *Node) RelayDapMsg(dm DapMsg, priority int) {
	msgData := dm.ToProto()
	bytes, _ := proto.Marshal(msgData)
	n.network.peerManager.Broadcast(bytes, priority)
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
	dm := NewDapmsg(cmd, bytes, msgKey, uniOrBroadcast)

	data, err := proto.Marshal(dm.ToProto())
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (n *Node) BroadcastBlock(block *core.Block) error {
	data, err := n.prepareData(block.ToProto(), SyncBlock, Broadcast, hex.EncodeToString(block.GetHash()))
	if err != nil {
		return err
	}
	n.network.peerManager.Broadcast(data, SyncBlockPriority)
	logger.WithFields(logger.Fields{
		"peer_id":     n.GetPeerID(),
		"height":      block.GetHeight(),
		"hash":        hex.EncodeToString(block.GetHash()),
		"num_streams": len(n.network.peerManager.streams),
		"data_len":    len(data),
	}).Info("Node: is broadcasting a block.")
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

	n.network.peerManager.Broadcast(data, GetBlockchainInfoPriority)
}

func (n *Node) GetPeerlistBroadcast(maxNum int) error {
	request := &networkpb.GetPeerList{MaxNumber: int32(maxNum)}

	data, err := n.prepareData(request, GetPeerList, Broadcast, "")
	if err != nil {
		return err
	}
	n.network.peerManager.Broadcast(data, GetPeerListPriority)
	return nil
}

func (n *Node) TxBroadcast(tx *core.Transaction) error {
	data, err := n.prepareData(tx.ToProto(), BroadcastTx, Broadcast, hex.EncodeToString(tx.ID))
	if err != nil {
		return err
	}
	n.network.peerManager.Broadcast(data, BroadcastTxPriority)
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
	n.network.peerManager.Broadcast(data, BroadcastBatchTxsPriority)
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
	n.network.peerManager.Broadcast(data, SyncPeerListPriority)
	return nil
}

func (n *Node) SendBlockUnicast(block *core.Block, pid peer.ID) error {
	data, err := n.prepareData(block.ToProto(), SyncBlock, Unicast, hex.EncodeToString(block.GetHash()))
	if err != nil {
		return err
	}
	n.network.peerManager.Unicast(data, pid, SyncBlockPriority)
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
	n.network.peerManager.Unicast(data, pid, ReturnPeerListPriority)
	return nil
}

func (n *Node) RequestBlockUnicast(hash core.Hash, pid peer.ID) error {
	//build a deppley message

	dm := NewDapmsg(RequestBlock, hash, hex.EncodeToString(hash), Unicast)
	data, err := proto.Marshal(dm.ToProto())
	if err != nil {
		return err
	}
	n.network.peerManager.Unicast(data, pid, RequestBlockPriority)
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

	n.network.peerManager.Unicast(data, pid, GetCommonBlocksPriority)
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

	n.network.peerManager.Unicast(data, pid, GetBlocksPriority)
	return nil
}

func (n *Node) addBlockToPool(block *core.Block, pid peer.ID) {
	//add block to blockpool. Make sure this is none blocking.
	n.bm.Push(block, pid)
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

		blk := n.getFromProtoBlockMsg(dm.GetData())
		n.addBlockToPool(blk, pid)
		if dm.uniOrBroadcast == Broadcast {
			n.RelayDapMsg(*dm, SyncBlockPriority)
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
		LibHash:       n.GetBlockchain().GetLIBHash(),
		LibHeight:     n.GetBlockchain().GetLIBHeight(),
	}

	data, err := n.prepareData(response, ReturnBlockchainInfo, Unicast, "")
	if err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "GetBlockchainInfoRequest",
		}).Warn("Node: prepare data failed.")
		return
	}

	n.network.peerManager.Unicast(data, pid, ReturnBlockchainInfoPriority)
}

func (n *Node) ReturnBlockchainInfoHandler(dm *DapMsg, pid peer.ID) {
	blockchainInfo := &networkpb.ReturnBlockchainInfo{}
	if err := proto.Unmarshal(dm.data, blockchainInfo); err != nil {
		logger.WithFields(logger.Fields{
			"cmd": "ReturnBlockchainInfo",
		}).Info("Node: parse data failed.")
		return
	}

	n.downloadManager.AddPeerBlockChainInfo(pid, blockchainInfo.GetBlockHeight(), blockchainInfo.GetLibHeight())
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

	n.network.peerManager.Unicast(data, pid, ReturnBlocksPriority)
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

	n.network.peerManager.Unicast(data, pid, ReturnCommonBlocksPriority)
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

func (n *Node) AddTxToPool(dm *DapMsg) {
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

func (n *Node) AddBatchTxsToPool(dm *DapMsg) {
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

func (n *Node) GetNodePeers(data []byte, pid peer.ID) {
	getPeerlistRequest := &networkpb.GetPeerList{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(data, getPeerlistRequest); err != nil {
		logger.WithError(err).Warn("Node: parse GetPeerList failed.")
	}

	peers := n.network.peerManager.RandomGetConnectedPeers(int(getPeerlistRequest.GetMaxNumber()))
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

	n.network.peerManager.ReceivePeers(pid, peers)
}

func (n *Node) SendRequestedBlock(hash []byte, pid peer.ID) {
	blockBytes, err := n.bm.Getblockchain().GetDb().Get(hash)
	if err != nil {
		logger.WithError(err).Warn("Node: failed to get the requested block source database.")
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
