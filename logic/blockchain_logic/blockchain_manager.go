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
package blockchain_logic

import (
	"bytes"

	"github.com/dappley/go-dappley/common/hash"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/block"
	blockpb "github.com/dappley/go-dappley/core/block/pb"
	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/logic/block_logic"
	"github.com/dappley/go-dappley/logic/utxo_logic"

	"github.com/dappley/go-dappley/common"
	corepb "github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/dappley/go-dappley/storage"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
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

type BlockchainManager struct {
	blockchain        *Blockchain
	blockPool         *core.BlockPool
	downloadRequestCh chan chan bool
	netService        NetService
}

func NewBlockchainManager(blockchain *Blockchain, blockpool *core.BlockPool, service NetService) *BlockchainManager {
	bm := &BlockchainManager{
		blockchain: blockchain,
		blockPool:  blockpool,
		netService: service,
	}
	bm.ListenToNetService()
	return bm
}

func (bm *BlockchainManager) SetDownloadRequestCh(requestCh chan chan bool) {
	bm.downloadRequestCh = requestCh
}

func (bm *BlockchainManager) RequestDownloadBlockchain() {
	go func() {
		finishChan := make(chan bool, 1)

		bm.Getblockchain().SetState(blockchain.BlockchainDownloading)

		select {
		case bm.downloadRequestCh <- finishChan:
		default:
			logger.Warn("BlockchainManager: Request download failed! download request channel is full!")
		}

		<-finishChan
		bm.Getblockchain().SetState(blockchain.BlockchainReady)
	}()
}

func (bm *BlockchainManager) ListenToNetService() {
	if bm.netService == nil {
		return
	}

	for _, command := range bmSubscribedTopics {
		bm.netService.Listen(command, bm.GetCommandHandler(command))
	}
}

func (bm *BlockchainManager) GetCommandHandler(commandName string) network_model.CommandHandlerFunc {

	switch commandName {
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

func (bm *BlockchainManager) GetblockPool() *core.BlockPool {
	return bm.blockPool
}

func (bm *BlockchainManager) VerifyBlock(blk *block.Block) bool {
	if !block_logic.VerifyHash(blk) {
		logger.Warn("BlockchainManager: Block hash verification failed!")
		return false
	}
	//TODO: Verify double spending transactions in the same blk
	if !(bm.blockchain.GetConsensus().Validate(blk)) {
		logger.Warn("BlockchainManager: blk is invalid according to consensus!")
		return false
	}
	logger.Debug("BlockchainManager: blk is valid according to consensus.")
	return true
}

func (bm *BlockchainManager) Push(blk *block.Block, pid peer.ID) {
	logger.WithFields(logger.Fields{
		"from":   pid.String(),
		"hash":   blk.GetHash().String(),
		"height": blk.GetHeight(),
	}).Info("BlockchainManager: received a new blk.")

	if bm.blockchain.GetState() != blockchain.BlockchainReady {
		logger.Info("BlockchainManager: Blockchain not ready, discard received blk")
		return
	}
	if !bm.VerifyBlock(blk) {
		return
	}

	receiveBlockHeight := blk.GetHeight()
	ownBlockHeight := bm.Getblockchain().GetMaxHeight()
	if receiveBlockHeight >= ownBlockHeight &&
		receiveBlockHeight-ownBlockHeight >= HeightDiffThreshold &&
		bm.blockchain.GetState() == blockchain.BlockchainReady {
		logger.Info("The height of the received blk is higher than the height of its own blk,to start download blockchain")
		bm.RequestDownloadBlockchain()
		return
	}

	bm.blockPool.Add(blk)
	fork := bm.blockPool.GetFork(blk)
	if fork == nil {
		return
	}
	forkHead := fork[len(fork)-1]
	forkHeadParentHash := forkHead.GetPrevHash()
	if forkHeadParentHash == nil {
		return
	}
	parent, _ := bm.blockchain.GetBlockByHash(forkHeadParentHash)
	if parent == nil {
		logger.WithFields(logger.Fields{
			"parent_hash":   forkHeadParentHash,
			"parent_height": forkHead.GetHeight() - 1,
			"from":          pid,
		}).Info("BlockchainManager: cannot find the parent of the received blk from blockchain. Requesting the parent...")
		bm.RequestBlock(forkHead.GetPrevHash(), pid)
		return
	}

	bm.blockchain.SetState(blockchain.BlockchainSync)
	_ = bm.MergeFork(fork, forkHeadParentHash)
	bm.blockPool.RemoveFork(fork)
	bm.blockchain.SetState(blockchain.BlockchainReady)
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

	//verify transactions in the fork
	utxo, scState, err := RevertUtxoAndScStateAtBlockHash(bm.blockchain.GetDb(), bm.blockchain, forkParentHash)
	if err != nil {
		logger.Error("BlockchainManager: blockchain is corrupted! Delete the database file and resynchronize to the network.")
		return err
	}
	rollBackUtxo := utxo.DeepCopy()
	rollScState := scState.DeepCopy()

	parentBlk, err := bm.blockchain.GetBlockByHash(forkParentHash)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
			"hash":  forkParentHash.String(),
		}).Error("BlockchainManager: get fork parent block failed.")
	}

	firstCheck := true

	for i := len(forkBlks) - 1; i >= 0; i-- {
		logger.WithFields(logger.Fields{
			"height": forkBlks[i].GetHeight(),
			"hash":   forkBlks[i].GetHash().String(),
		}).Debug("BlockchainManager: is verifying a block in the fork.")

		if !block_logic.VerifyTransactions(forkBlks[i], utxo, scState, bm.blockchain.GetSCManager(), parentBlk) {
			return ErrTransactionVerifyFailed
		}

		lib, ok := bm.Getblockchain().GetConsensus().CheckLibPolicy(forkBlks[i])
		if !ok {
			return ErrProducerNotEnough
		}

		if firstCheck {
			firstCheck = false
			bm.blockchain.Rollback(forkParentHash, rollBackUtxo, rollScState)
		}

		ctx := BlockContext{Block: forkBlks[i], Lib: lib, UtxoIndex: utxo, State: scState}
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
func (bm *BlockchainManager) RequestBlock(hash hash.Hash, pid peer.ID) {
	request := &corepb.RequestBlock{Hash: hash}

	bm.netService.SendCommand(RequestBlock, request, pid, network_model.Unicast, network_model.HighPriorityCommand)
}

//RequestBlockhandler handles when blockchain manager receives a requestBlock command from its peers
func (bm *BlockchainManager) RequestBlockHandler(command *network_model.DappRcvdCmdContext) {
	request := &corepb.RequestBlock{}

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
func (bm *BlockchainManager) SendBlockToPeer(blk *block.Block, pid peer.ID) {

	bm.SendBlock(blk, pid, network_model.Unicast)
}

//BroadcastBlock broadcasts a block to all peers
func (bm *BlockchainManager) BroadcastBlock(blk *block.Block) {
	var broadcastPid peer.ID
	bm.SendBlock(blk, broadcastPid, network_model.Broadcast)
}

//SendBlock sends a SendBlock command to its peer with pid by finding the block from its database
func (bm *BlockchainManager) SendBlock(blk *block.Block, pid peer.ID, isBroadcast bool) {

	bm.netService.SendCommand(SendBlock, blk.ToProto(), pid, isBroadcast, network_model.HighPriorityCommand)
}

//SendBlockHandler handles when blockchain manager receives a sendBlock command from its peers
func (bm *BlockchainManager) SendBlockHandler(command *network_model.DappRcvdCmdContext) {
	pb := &blockpb.Block{}

	//unmarshal byte to proto
	if err := proto.Unmarshal(command.GetData(), pb); err != nil {
		logger.WithError(err).Warn("BlockchainManager: parse data failed.")
		return
	}

	blk := &block.Block{}
	blk.FromProto(pb)
	bm.Push(blk, command.GetSource())

	if command.IsBroadcast() {
		//relay the original command
		var broadcastPid peer.ID
		bm.netService.Relay(command.GetCommand(), broadcastPid, network_model.HighPriorityCommand)
	}
}

// RevertUtxoAndScStateAtBlockHash returns the previous snapshot of UTXOIndex when the block of given hash was the tail block.
func RevertUtxoAndScStateAtBlockHash(db storage.Storage, bc *Blockchain, hash hash.Hash) (*utxo_logic.UTXOIndex, *core.ScState, error) {
	index := utxo_logic.NewUTXOIndex(bc.GetUtxoCache())
	scState := core.LoadScStateFromDatabase(db)
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
	}

	return index, scState, nil
}

/* NumForks returns the number of forks in the BlockPool and the height of the current longest fork */
func (bm *BlockchainManager) NumForks() (int64, int64) {
	var numForks, maxHeight int64 = 0, 0

	bm.blockPool.ForkHeadRange(func(blkHash string, tree *common.Tree) {
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
