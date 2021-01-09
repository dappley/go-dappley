// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//
package lblockchain

import (
	"bytes"

	"github.com/dappley/go-dappley/common/log"

	"github.com/pkg/errors"

	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/common/pubsub"
	"github.com/dappley/go-dappley/core/block"
	blockpb "github.com/dappley/go-dappley/core/block/pb"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/logic/lblock"
	"github.com/dappley/go-dappley/logic/lutxo"

	"github.com/dappley/go-dappley/common"
	lblockchainpb "github.com/dappley/go-dappley/logic/lblockchain/pb"
	"github.com/dappley/go-dappley/network/networkmodel"
	"github.com/dappley/go-dappley/storage"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

const (
	HeightDiffThreshold = 10
	SendBlock           = "SendBlockByHash"
	RequestBlock        = "requestBlock"
)

var (
	bmSubscribedTopics = []string{
		SendBlock,
		RequestBlock,
	}
)

var (
	ErrParentBlockNotFound = errors.New("Not able to find parent block in blockchain")
)

type BlockchainManager struct {
	blockchain        *Blockchain
	blockPool         *blockchain.BlockPool
	consensus         Consensus
	downloadRequestCh chan chan bool
	netService        NetService
}

func NewBlockchainManager(blockchain *Blockchain, blockpool *blockchain.BlockPool, service NetService, consensus Consensus) *BlockchainManager {
	bm := &BlockchainManager{
		blockchain:        blockchain,
		blockPool:         blockpool,
		netService:        service,
		consensus:         consensus,
		downloadRequestCh: make(chan chan bool, 100),
	}
	bm.ListenToNetService()
	return bm
}
func (bm *BlockchainManager) GetDownloadRequestCh() chan chan bool {
	return bm.downloadRequestCh
}

func (bm *BlockchainManager) RequestDownloadBlockchain() {
	go func() {
		defer log.CrashHandler()

		finishChan := make(chan bool, 1)

		bm.Getblockchain().mutex.Lock()
		logger.Info("BlockchainManager: requestDownloadBlockchain start, set blockchain status to downloading!")
		bm.Getblockchain().SetState(blockchain.BlockchainDownloading)
		bm.Getblockchain().mutex.Unlock()

		select {
		case bm.downloadRequestCh <- finishChan:
		default:
			logger.Warn("BlockchainManager: Request download failed! download request channel is full!")
		}

		<-finishChan
		bm.Getblockchain().mutex.Lock()
		bm.Getblockchain().SetState(blockchain.BlockchainReady)
		bm.Getblockchain().mutex.Unlock()
		logger.Info("BlockchainManager: requestDownloadBlockchain finished, set blockchain status to ready!")

	}()
}

func (bm *BlockchainManager) ListenToNetService() {
	if bm.netService == nil {
		return
	}

	bm.netService.Listen(bm)
}

func (bm *BlockchainManager) GetSubscribedTopics() []string {
	return bmSubscribedTopics
}

func (bm *BlockchainManager) GetTopicHandler(topic string) pubsub.TopicHandler {

	switch topic {
	case SendBlock:
		return bm.SendBlockHandler
	case RequestBlock:
		return bm.RequestBlockHandler
	}
	return nil
}

func (bm *BlockchainManager) Getblockchain() *Blockchain {
	return bm.blockchain
}

func (bm *BlockchainManager) GetblockPool() *blockchain.BlockPool {
	return bm.blockPool
}

func (bm *BlockchainManager) VerifyBlock(blk *block.Block) bool {
	if !lblock.VerifyHash(blk) {
		logger.Warn("BlockchainManager: Block hash verification failed!")
		return false
	}
	//TODO: Verify double spending transactions in the same blk
	if !(bm.consensus.Validate(blk)) {
		logger.Warn("BlockchainManager: blk is invalid according to libPolicy!")
		return false
	}
	logger.Debug("BlockchainManager: blk is valid according to libPolicy.")
	return true
}

func (bm *BlockchainManager) Push(blk *block.Block, pid networkmodel.PeerInfo) {
	logger.WithFields(logger.Fields{
		"from":   pid.PeerId.String(),
		"hash":   blk.GetHash().String(),
		"height": blk.GetHeight(),
	}).Info("BlockChainManager: received a new block.")

	if bm.blockchain.GetState() != blockchain.BlockchainReady {
		logger.Infof("BlockchainManager: Blockchain not ready, discard received blk. Current status is %v", bm.blockchain.GetState())
		return
	}
	if !bm.VerifyBlock(blk) {
		return
	}

	receiveBlockHeight := blk.GetHeight()
	ownBlockHeight := bm.Getblockchain().GetMaxHeight()
	// Do the subtraction calculation after judging the size to avoid the overflow of the symbol uint64
	if receiveBlockHeight > ownBlockHeight && receiveBlockHeight-ownBlockHeight >= HeightDiffThreshold &&
		bm.blockchain.GetState() == blockchain.BlockchainReady {
		logger.WithFields(logger.Fields{
			"receiveBlockHeight": receiveBlockHeight,
			"ownBlockHeight":     ownBlockHeight,
		}).Warn("The height of the received blk is higher than the height of its own blk,to start download blockchain")
		bm.RequestDownloadBlockchain()
		return
	}

	bm.blockPool.AddBlock(blk)
	forkHeadBlk := bm.blockPool.GetForkHead(blk)
	if forkHeadBlk == nil {
		return
	}

	if !bm.blockchain.IsInBlockchain(forkHeadBlk.GetHash()) {
		if !bm.blockchain.IsInBlockchain(forkHeadBlk.GetPrevHash()) {
			logger.WithFields(logger.Fields{
				"parent_hash": forkHeadBlk.GetPrevHash(),
				"from":        pid,
			}).Info("BlockchainManager: cannot find the parent of the received blk from blockchain. Requesting the parent...")
			bm.RequestBlock(forkHeadBlk.GetPrevHash(), pid)
			return
		}
	}

	fork := bm.blockPool.GetFork(forkHeadBlk.GetHash())

	if fork == nil {
		return
	}

	if fork[0].GetHeight() <= bm.Getblockchain().GetMaxHeight() {
		return
	}

	bm.Getblockchain().mutex.Lock()
	if bm.blockchain.GetState() != blockchain.BlockchainReady {
		logger.Infof("Push: MergeFork cancelled  because blockchain is not ready. Current status is %v", bm.blockchain.GetState())
		bm.Getblockchain().mutex.Unlock()
		return
	}
	bm.blockchain.SetState(blockchain.BlockchainSync)
	bm.Getblockchain().mutex.Unlock()


	logger.Info("Push: set blockchain status to sync.")

	err := bm.MergeFork(fork, forkHeadBlk.GetPrevHash())
	if err != nil {
		logger.Warn("Merge fork failed.err:", err)
	}
	bm.blockPool.RemoveFork(fork)

	bm.Getblockchain().mutex.Lock()
	bm.blockchain.SetState(blockchain.BlockchainReady)
	bm.Getblockchain().mutex.Unlock()
	logger.Info("Push: set blockchain status to ready.")

	return
}

func (bm *BlockchainManager) MergeFork(forkBlks []*block.Block, forkParentHash hash.Hash) error {

	//find parent block
	if len(forkBlks) == 0 {
		return nil
	}
	forkHeadBlock := forkBlks[len(forkBlks)-1]
	if forkHeadBlock == nil {
		return nil
	}

	//utxo has been reverted to forkParentHash in this step
	utxo, scState, err := RevertUtxoAndScStateAtBlockHash(bm.blockchain.GetDb(), bm.blockchain, forkParentHash)
	if err != nil {
		logger.Error("BlockchainManager: blockchain is corrupted! Delete the database file and resynchronize to the network.")
		return err
	}
	bm.blockchain.Rollback(forkParentHash, scState)

	parentBlk, err := bm.blockchain.GetBlockByHash(forkParentHash)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
			"hash":  forkParentHash.String(),
		}).Error("BlockchainManager: get fork parent block failed.")
	}

	for i := len(forkBlks) - 1; i >= 0; i-- {
		if !bm.Getblockchain().CheckLibPolicy(forkBlks[i]) {
			return ErrProducerNotEnough
		}

		logger.WithFields(logger.Fields{
			"height": forkBlks[i].GetHeight(),
			"hash":   forkBlks[i].GetHash().String(),
		}).Info("BlockchainManager: is verifying a block in the fork.")

		if !lblock.VerifyTransactions(forkBlks[i], utxo, scState, parentBlk) {
			return ErrTransactionVerifyFailed
		}

		ctx := BlockContext{Block: forkBlks[i], UtxoIndex: utxo, State: scState}
		err = bm.blockchain.AddBlockContextToTail(&ctx)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error":  err,
				"height": forkBlks[i].GetHeight(),
			}).Error("BlockchainManager: add fork to tail failed.")
		}
		parentBlk = forkBlks[i]
	}

	return nil
}

//RequestBlock sends a requestBlock command to its peer with pid through network module
func (bm *BlockchainManager) RequestBlock(hash hash.Hash, pid networkmodel.PeerInfo) {
	request := &lblockchainpb.RequestBlock{Hash: hash}

	bm.netService.UnicastHighProrityCommand(RequestBlock, request, pid)
}

//RequestBlockhandler handles when blockchain manager receives a requestBlock command from its peers
func (bm *BlockchainManager) RequestBlockHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	request := &lblockchainpb.RequestBlock{}

	if err := proto.Unmarshal(command.GetData(), request); err != nil {
		logger.WithFields(logger.Fields{
			"name": command.GetCommandName(),
		}).Info("BlockchainManager: parse data failed.")
	}

	block, err := bm.Getblockchain().GetBlockByHash(request.Hash)
	if err != nil {
		logger.WithError(err).Warn("BlockchainManager: failed to get the requested block.")
		return
	}

	bm.SendBlockToPeer(block, command.GetSource())
}

//SendBlockToPeer unicasts a block to the peer with peer id "pid"
func (bm *BlockchainManager) SendBlockToPeer(block *block.Block, pid networkmodel.PeerInfo) {

	bm.netService.UnicastNormalPriorityCommand(SendBlock, block.ToProto(), pid)
}

//BroadcastBlock broadcasts a block to all peers
func (bm *BlockchainManager) BroadcastBlock(block *block.Block) {
	bm.netService.BroadcastHighProrityCommand(SendBlock, block.ToProto())
}

//SendBlockHandler handles when blockchain manager receives a sendBlock command from its peers
func (bm *BlockchainManager) SendBlockHandler(input interface{}) {

	var command *networkmodel.DappRcvdCmdContext
	command = input.(*networkmodel.DappRcvdCmdContext)

	blockpb := &blockpb.Block{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(command.GetData(), blockpb); err != nil {
		logger.WithError(err).Warn("BlockchainManager: parse data failed.")
		return
	}

	blk := &block.Block{}
	blk.FromProto(blockpb)
	bm.Push(blk, command.GetSource())

	if command.IsBroadcast() {
		//relay the original command
		bm.netService.Relay(command.GetCommand(), networkmodel.PeerInfo{}, networkmodel.HighPriorityCommand)
	}
}

// RevertUtxoAndScStateAtBlockHash returns the previous snapshot of UTXOIndex when the block of given hash was the tail block.
func RevertUtxoAndScStateAtBlockHash(db storage.Storage, bc *Blockchain, hash hash.Hash) (*lutxo.UTXOIndex, *scState.ScState, error) {
	index := lutxo.NewUTXOIndex(bc.GetUtxoCache())
	scState := scState.LoadScStateFromDatabase(db)
	bci := bc.Iterator()

	// Start from the tail of blockchain, compute the previous UTXOIndex by undoing transactions
	// in the block, until the block hash matches.
	for {
		block, err := bci.Next()

		if bytes.Compare(block.GetHash(), hash) == 0 {
			break
		}

		if err != nil {
			return nil, nil, err
		}

		if len(block.GetPrevHash()) == 0 {
			return nil, nil, ErrBlockDoesNotExist
		}

		err = index.UndoTxsInBlock(block, db)

		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"hash": block.GetHash(),
			}).Warn("BlockchainManager: failed to calculate previous state of UTXO index for the block")
			return nil, nil, err
		}

		err = scState.RevertState(db, block.GetHash())
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"hash": block.GetHash(),
			}).Warn("BlockchainManager: failed to calculate previous state of scState for the block")
			return nil, nil, err
		}

		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"hash": block.GetHash(),
			}).Errorf("BlockchainManager: failed to delete block %v", err.Error())
			return nil, nil, err
		}
	}
	//updated utxo in db
	err:=index.Save()
	if err!=nil{
		return nil, nil, err
	}
	return index, scState, nil
}

/* NumForks returns the number of forks in the BlockPool and the height of the current longest fork */
func (bm *BlockchainManager) NumForks() (int64, int64) {
	var numForks, maxHeight int64 = 0, 0

	bm.blockPool.ForkHeadRange(func(blkHash string, tree *common.TreeNode) {
		rootBlk := tree.GetValue().(*block.Block)
		_, err := bm.blockchain.GetBlockByHash(rootBlk.GetPrevHash())
		if err == nil {
			/* the cached block is rooted in the BlockChain */
			numForks += tree.NumLeaves()
			t := tree.Height()
			if t > maxHeight {
				maxHeight = t
			}
		}
	})

	return numForks, maxHeight
}
