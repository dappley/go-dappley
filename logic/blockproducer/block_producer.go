package blockproducer

import (
	"time"

	"github.com/dappley/go-dappley/common/log"

	"github.com/dappley/go-dappley/common/deadline"
	"github.com/dappley/go-dappley/core/blockchain"

	"github.com/dappley/go-dappley/logic/lblock"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/blockproducerinfo"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/logic/lblockchain"
	"github.com/dappley/go-dappley/logic/ltransaction"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/vm"
	logger "github.com/sirupsen/logrus"
)

type BlockProducer struct {
	bm        *lblockchain.BlockchainManager
	con       Consensus
	producer  *blockproducerinfo.BlockProducerInfo
	stopCh    chan bool
	isRunning bool
}

// NewBlockProducer returns a new block producer instance
func NewBlockProducer(bm *lblockchain.BlockchainManager, con Consensus, producer *blockproducerinfo.BlockProducerInfo) *BlockProducer {
	return &BlockProducer{
		bm:        bm,
		con:       con,
		producer:  producer,
		stopCh:    make(chan bool, 1),
		isRunning: false,
	}
}

// Start starts the block producing process
func (bp *BlockProducer) Start() {
	// clear stop channel buffer
	select {
	case <-bp.stopCh:
	default:
	}
	if bp.isRunning {
		return
	}
	go func() {
		defer log.CrashHandler()

		logger.Info("BlockProducer Starts...")
		bp.isRunning = true
		for {
			select {
			case <-bp.stopCh:
				bp.isRunning = false
				return
			default:
				bp.con.ProduceBlock(bp.produceBlock)
			}
		}
	}()
}

// Stop stops the block producing process
func (bp *BlockProducer) Stop() {
	logger.Info("BlockProducer stops...")
	bp.stopCh <- true
}

// IsProducingBlock returns if the local producer is producing a block
func (bp *BlockProducer) IsProducingBlock() bool {
	return !bp.producer.IsIdle()
}

// IsProducingBlock returns if the local producer is producing a block
func (bp *BlockProducer) GetProduceBlockStatus() bool {
	return bp.isRunning
}

// produceBlock produces a new block and add it to blockchain
func (bp *BlockProducer) produceBlock(processFunc func(*block.Block), deadline deadline.Deadline) {
	// Do not produce block if block pool is syncing
	bp.bm.Getblockchain().GetBlockMutex().Lock()
	if bp.bm.Getblockchain().GetState() != blockchain.BlockchainReady {
		logger.Infof("BlockProducer: block producer paused because blockchain is not ready. Current status is %v", bp.bm.Getblockchain().GetState())
		bp.bm.Getblockchain().GetBlockMutex().Unlock()
		return
	}
	bp.bm.Getblockchain().SetState(blockchain.BlockchainProduce)
	bp.bm.Getblockchain().GetBlockMutex().Unlock()

	defer func() {
		bp.bm.Getblockchain().GetBlockMutex().Lock()
		bp.bm.Getblockchain().SetState(blockchain.BlockchainReady)
		bp.bm.Getblockchain().GetBlockMutex().Unlock()
		logger.Info("BlockProducer: set blockchain status to ready.")
	}()

	//makeup a block, fill in necessary information to check lib policy.
	blk := block.NewBlockByHash(bp.bm.Getblockchain().GetTailBlockHash(), bp.producer.Beneficiary())
	if !bp.bm.Getblockchain().CheckMinProducerPolicy(blk) {
		logger.Warn("BlockProducer: the number of producers is not enough.")
		tailBlock, _ := bp.bm.Getblockchain().GetTailBlock()
		bp.bm.BroadcastBlock(tailBlock)
		return
	}

	bp.producer.BlockProduceStart()
	defer bp.producer.BlockProduceFinish()

	logger.Infof("BlockProducerer: producing block... ***time is %v***", time.Now().Unix())

	ctx := bp.prepareBlock(deadline)

	if ctx != nil && processFunc != nil {
		processFunc(ctx.Block)
	}

	if ctx == nil || !bp.con.Validate(ctx.Block) {
		logger.Error("BlockProducer: produced an invalid block!")
		return
	}
	bp.addBlockToBlockchain(ctx)
}

// prepareBlock generates a new block
func (bp *BlockProducer) prepareBlock(deadline deadline.Deadline) *lblockchain.BlockContext {

	parentBlock, err := bp.bm.Getblockchain().GetTailBlock()
	if err != nil {
		logger.WithError(err).Error("BlockProducerInfo: cannot get the current tail block!")
		return nil
	}

	// Retrieve all valid transactions from tx pool
	utxoIndex := lutxo.NewUTXOIndex(bp.bm.Getblockchain().GetUtxoCache())

	validTxs, state := bp.collectTransactions(utxoIndex, parentBlock, deadline)

	totalTips := bp.calculateTips(validTxs)
	cbtx := ltransaction.NewCoinbaseTX(account.NewAddress(bp.producer.Beneficiary()), "", bp.bm.Getblockchain().GetMaxHeight()+1, totalTips)
	validTxs = append(validTxs, &cbtx)
	if !utxoIndex.UpdateUtxo(&cbtx) {
		logger.Warn("prepareBlock warn")
	}

	logger.WithFields(logger.Fields{
		"valid_txs": len(validTxs),
	}).Info("BlockProducer: prepared a block.")

	ctx := lblockchain.BlockContext{Block: block.NewBlock(validTxs, parentBlock, bp.producer.Beneficiary()), UtxoIndex: utxoIndex, State: state}
	return &ctx
}

// collectTransactions pack transactions from transaction pool to a new block
func (bp *BlockProducer) collectTransactions(utxoIndex *lutxo.UTXOIndex, parentBlk *block.Block, deadline deadline.Deadline) ([]*transaction.Transaction, *scState.ScState) {

	var validTxs []*transaction.Transaction
	totalSize := 0
	count := 0

	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	rewards := make(map[string]string)
	currBlkHeight := parentBlk.GetHeight() + 1

	contractState := scState.NewScState(bp.bm.Getblockchain().GetUtxoCache())

	for totalSize < bp.bm.Getblockchain().GetBlockSizeLimit() && bp.bm.Getblockchain().GetTxPool().GetNumOfTxInPool() > 0 && !deadline.IsPassed() {

		txNode, err := bp.bm.Getblockchain().GetTxPool().PopTransactionWithMostTips(utxoIndex)
		if err != nil {
			break
		}
		if txNode == nil {
			continue
		}
		totalSize += txNode.Size
		count++

		ctx := ltransaction.NewTxContract(txNode.Value)
		if ctx != nil {
			minerAddr := account.NewAddress(bp.producer.Beneficiary())
			gasCount, generatedTxs, err := ltransaction.VerifyAndCollectContractOutput(utxoIndex, ctx, contractState, engine, currBlkHeight, parentBlk, rewards, bp.bm.Getblockchain().GetDb())
			if err != nil {
				logger.Warn("VerifyAndCollectContractOutput error: ", err)
				continue
			}

			if grtx, exists := ltransaction.NewGasRewardTx(account.NewTransactionAccountByAddress(minerAddr), currBlkHeight, common.NewAmount(gasCount), ctx.GasPrice, count); exists {
				generatedTxs = append(generatedTxs, &grtx)
			}

			if gctx, exists := ltransaction.NewGasChangeTx(ctx.GetDefaultFromTransactionAccount(), currBlkHeight, common.NewAmount(gasCount), ctx.GasLimit, ctx.GasPrice, count); exists {
				generatedTxs = append(generatedTxs, &gctx)
			}
			validTxs = append(validTxs, txNode.Value)

			if generatedTxs != nil {
				validTxs = append(validTxs, generatedTxs...)
				if !utxoIndex.UpdateUtxos(generatedTxs) {
					logger.Warn("collectTransactions warn: generatedTxs != nil")
				}
			}
		} else {
			validTxs = append(validTxs, txNode.Value)
			if !utxoIndex.UpdateUtxo(txNode.Value) {
				logger.Warn("collectTransactions warn: update utxo error")
			}
		}
	}

	// append reward transaction
	if len(rewards) > 0 {
		rtx := ltransaction.NewRewardTx(currBlkHeight, rewards)
		validTxs = append(validTxs, &rtx)
		if !utxoIndex.UpdateUtxo(&rtx) {
			logger.Warn("collectTransactions warn: rewards update utxo error")
		}
	}
	return validTxs, contractState
}

// calculateTips calculate how much tips are earned from the input transactions
func (bp *BlockProducer) calculateTips(txs []*transaction.Transaction) *common.Amount {
	//calculate tips
	totalTips := common.NewAmount(0)
	for _, tx := range txs {
		totalTips = totalTips.Add(tx.Tip)
	}
	return totalTips
}

// addBlockToBlockchain adds the new block to blockchain
func (bp *BlockProducer) addBlockToBlockchain(ctx *lblockchain.BlockContext) {
	logger.WithFields(logger.Fields{
		"height": ctx.Block.GetHeight(),
		"hash":   ctx.Block.GetHash().String(),
	}).Info("BlockProducer: produced a new block.")
	if !lblock.VerifyHash(ctx.Block) {
		logger.Warn("BlockProducer: hash of the new block is invalid.")
		return
	}

	err := bp.bm.Getblockchain().AddBlockContextToTail(ctx)
	if err != nil {
		logger.Warn(err)
		return
	}

	for _, tx := range ctx.Block.GetTransactions() {
		if tx.CreateTime > 0 {
			TxAddToBlockCost.Update((time.Now().UnixNano()/1e6 - tx.CreateTime) / 1e3)
		}

		if tx.IsChangeProducter() {

			bp.bm.SetNewDynastyByString(tx.Vout[0].Contract, tx.Vout[0].PubKeyHash.GenerateAddress().String())
		}
	}

	bp.bm.BroadcastBlock(ctx.Block)
	logger.Info("BlockProducer: Broadcast block")
	bp.bm.CheckDynast(ctx.Block.GetHeight())
}
