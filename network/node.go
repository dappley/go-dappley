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
	"io/ioutil"
)

const (
	maxGetBlocksNum   = 10
	maxSyncPeersCount = 32

	SyncBlock    = "SyncBlock"
	SyncPeerList = "SyncPeerList"

	RequestBlock      = "requestBlock"
	BroadcastTx       = "BroadcastTx"
	BroadcastBatchTxs = "BraodcastBatchTxs"

	Unicast       = false
	Broadcast     = true
	dispatchChLen = 1024 * 4
	requestChLen  = 1024
)

const (
	GetBlockchainInfoPriority    = NormalPriorityCommand
	ReturnBlockchainInfoPriority = NormalPriorityCommand
	SyncBlockPriority            = HighPriorityCommand
	GetBlocksPriority            = HighPriorityCommand
	ReturnBlocksPriority         = HighPriorityCommand
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
	network         *Network
	bm              *core.BlockChainManager
	exitCh          chan bool
	privKey         crypto.PrivKey
	dispatcher      chan *DappPacketContext
	commandSendCh   chan *DappSendCmdContext
	downloadManager *DownloadManager
	commandBroker   *CommandBroker
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
		dispatcher:      make(chan *DappPacketContext, dispatchChLen),
		downloadManager: nil,
		commandSendCh:   make(chan *DappSendCmdContext, requestChLen),
		commandBroker:   NewCommandBroker(),
	}

	if bc != nil {
		db = node.bm.Getblockchain().GetDb()
	}

	node.network = NewNetwork(config, node.dispatcher, node.commandSendCh, db)
	node.network.OnStreamStop(node.OnStreamStop)
	node.network.Subscirbe(node.commandBroker)

	if err != nil {
		logger.WithError(err).Panic("Node: Can not initialize lru cache for recentlyRcvdDapMsgs!")
	}
	node.downloadManager = NewDownloadManager(node, node.commandSendCh)
	node.downloadManager.SubscribeCommandBroker(node.commandBroker)
	return node
}

func (n *Node) GetBlockchain() *core.Blockchain            { return n.bm.Getblockchain() }
func (n *Node) GetBlockPool() *core.BlockPool              { return n.bm.GetblockPool() }
func (n *Node) GetDownloadManager() *DownloadManager       { return n.downloadManager }
func (n *Node) GetInfo() *PeerInfo                         { return n.network.host.info }
func (n *Node) GetNetwork() *Network                       { return n.network }
func (n *Node) GetCommandSendCh() chan *DappSendCmdContext { return n.commandSendCh }
func (n *Node) GetCommandBroker() *CommandBroker           { return n.commandBroker }

func (n *Node) Start(listenPort int, seeds []string) error {
	err := n.network.Start(listenPort, n.privKey, seeds)
	if err != nil {
		return err
	}

	//TODO: Remove this later
	n.StartRequestLoop()

	//TODO: Rename this to StartRequestLoop later
	n.StartRequestLoop2()
	n.StartListenLoop()
	return nil
}

func (n *Node) Stop() {
	n.exitCh <- true
	n.network.Stop()
}

func (n *Node) StartRequestLoop2() {

	go func() {
		for {
			select {
			case cmdCtx := <-n.commandSendCh:
				if cmdCtx.command == nil {
					continue
				}
				logger.WithFields(logger.Fields{
					"command": cmdCtx.GetCommandName(),
				}).Warn("Node: Send Command!")
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
	if n.downloadManager != nil {
		n.downloadManager.DisconnectPeer(stream.peerID)
	}
}

func (n *Node) handle(msg *DappCmd, id peer.ID) {
	switch msg.GetName() {
	case BlockchainInfoRequest:
		n.GetBlockchainInfoHandler(msg, id)

	case BlockchainInfoResponse:
		n.ReturnBlockchainInfoHandler(msg, id)

	case SyncBlock:
		n.SyncBlockHandler(msg, id)

	case RequestBlock:
		n.SendRequestedBlock(msg.GetData(), id)

	case BroadcastTx:
		n.AddTxToPool(msg)

	case BroadcastBatchTxs:
		n.AddBatchTxsToPool(msg)

	case GetBlocksRequest:
		n.GetBlocksHandler(msg, id)

	case GetBlocksResponse:
		n.ReturnBlocksHandler(msg, id)

	//case GetCommonBlocksRequest:
	//	n.GetCommonBlocksHandler(msg, id)
	//
	//case GetCommonBlocksResponse:
	//	n.ReturnCommonBlocksHandler(msg, id)

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

func (n *Node) BroadcastBlock(block *core.Block) error {
	data, err := n.prepareData(block.ToProto(), SyncBlock, Broadcast, hex.EncodeToString(block.GetHash()))
	if err != nil {
		return err
	}
	n.network.Broadcast(data, SyncBlockPriority)
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
	data, err := n.prepareData(request, BlockchainInfoRequest, Broadcast, "")
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Warn("Node: broadcast BlockchainInfoRequest failed.")
	}

	n.network.Broadcast(data, GetBlockchainInfoPriority)
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

func (n *Node) SendBlockUnicast(block *core.Block, pid peer.ID) error {
	data, err := n.prepareData(block.ToProto(), SyncBlock, Unicast, hex.EncodeToString(block.GetHash()))
	if err != nil {
		return err
	}
	n.network.Unicast(data, pid, SyncBlockPriority)
	return nil
}

func (n *Node) RequestBlockUnicast(hash core.Hash, pid peer.ID) error {
	//build a deppley message

	dm := NewDapCmd(RequestBlock, hash, Unicast)
	data, err := proto.Marshal(dm.ToProto())
	if err != nil {
		return err
	}
	n.network.Unicast(data, pid, RequestBlockPriority)
	return nil
}

func (n *Node) GetCommonBlocksUnicast(blockHeaders []*SyncCommandBlocksHeader, pid peer.ID, msgId int32) error {
	var blockHeaderPbs []*corepb.BlockHeader
	for _, blockHeader := range blockHeaders {
		blockHeaderPbs = append(blockHeaderPbs,
			&corepb.BlockHeader{Hash: blockHeader.hash, Height: blockHeader.height})
	}

	getCommonBlocksPb := &networkpb.GetCommonBlocks{MsgId: msgId, BlockHeaders: blockHeaderPbs}

	data, err := n.prepareData(getCommonBlocksPb, GetCommonBlocksRequest, Unicast, "")
	if err != nil {
		return nil
	}

	n.network.Unicast(data, pid, GetCommonBlocksPriority)
	return nil
}

func (n *Node) DownloadBlocksUnicast(hashes []core.Hash, pid peer.ID) error {
	blkHashes := make([][]byte, len(hashes))
	for index, hash := range hashes {
		blkHashes[index] = hash
	}

	getBlockPb := &networkpb.GetBlocks{StartBlockHashes: blkHashes}

	data, err := n.prepareData(getBlockPb, GetBlocksRequest, Unicast, "")
	if err != nil {
		return nil
	}

	n.network.Unicast(data, pid, GetBlocksPriority)
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

func (n *Node) SyncBlockHandler(dm *DappCmd, pid peer.ID) {
	if len(dm.data) == 0 {
		logger.WithFields(logger.Fields{
			"name": "sync block",
		}).Warn("Node: can not find block information.")
		return
	}

	if dm.isBroadcast == Broadcast {

		blk := n.getFromProtoBlockMsg(dm.GetData())
		n.addBlockToPool(blk, pid)
		if dm.isBroadcast == Broadcast {
			n.RelayDapMsg(*dm, SyncBlockPriority)
		}
	} else {
		blk := n.getFromProtoBlockMsg(dm.GetData())
		n.addBlockToPool(blk, pid)
	}
}

func (n *Node) GetBlockchainInfoHandler(dm *DappCmd, pid peer.ID) {
	tailBlock, err := n.GetBlockchain().GetTailBlock()
	if err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetBlockchainInfoRequest",
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

	data, err := n.prepareData(response, BlockchainInfoResponse, Unicast, "")
	if err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetBlockchainInfoRequest",
		}).Warn("Node: prepare data failed.")
		return
	}

	n.network.Unicast(data, pid, ReturnBlockchainInfoPriority)
}

func (n *Node) ReturnBlockchainInfoHandler(dm *DappCmd, pid peer.ID) {
	blockchainInfo := &networkpb.ReturnBlockchainInfo{}
	if err := proto.Unmarshal(dm.data, blockchainInfo); err != nil {
		logger.WithFields(logger.Fields{
			"name": "BlockchainInfoResponse",
		}).Info("Node: parse data failed.")
		return
	}

	n.downloadManager.AddPeerBlockChainInfo(pid, blockchainInfo.GetBlockHeight(), blockchainInfo.GetLibHeight())
}

//TODO  Refactor getblocks in rpcService and node
func (n *Node) GetBlocksHandler(dm *DappCmd, pid peer.ID) {
	param := &networkpb.GetBlocks{}
	if err := proto.Unmarshal(dm.data, param); err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetBlocksRequest",
		}).Info("Node: parse data failed.")
		return
	}

	block := n.findBlockInRequestHash(param.GetStartBlockHashes())

	// Reach the blockchain's tail
	if block.GetHeight() >= n.GetBlockchain().GetMaxHeight() {
		logger.WithFields(logger.Fields{
			"name": "GetBlocksRequest",
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

	data, err := n.prepareData(result, GetBlocksResponse, Unicast, "")
	if err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetBlocksRequest",
		}).Warn("Node: prepare data failed.")
		return
	}

	n.network.Unicast(data, pid, ReturnBlocksPriority)
}

func (n *Node) GetCommonBlocksHandler(dm *DappCmd, pid peer.ID) {
	param := &networkpb.GetCommonBlocks{}
	if err := proto.Unmarshal(dm.data, param); err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetCommonBlocksRequest",
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

	data, err := n.prepareData(result, GetCommonBlocksResponse, Unicast, "")
	if err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetBlocksRequest",
		}).Warn("Node: prepare data failed.")
		return
	}

	n.network.Unicast(data, pid, ReturnCommonBlocksPriority)
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

func (n *Node) ReturnBlocksHandler(dm *DappCmd, pid peer.ID) {
	param := &networkpb.ReturnBlocks{}
	if err := proto.Unmarshal(dm.data, param); err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetBlocksResponse",
		}).Info("Node: parse data failed.")
		return
	}

	n.downloadManager.GetBlocksDataHandler(param, pid)
}

func (n *Node) ReturnCommonBlocksHandler(dm *DappCmd, pid peer.ID) {
	param := &networkpb.ReturnCommonBlocks{}

	if err := proto.Unmarshal(dm.data, param); err != nil {
		logger.WithFields(logger.Fields{
			"name": "GetCommonBlocksResponse",
		}).Info("Node: parse data failed.")
	}

	n.downloadManager.GetCommonBlockDataHandler(param, pid)
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
