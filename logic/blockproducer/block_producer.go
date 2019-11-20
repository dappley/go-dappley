package blockproducer

import (
	"encoding/hex"
	"github.com/dappley/go-dappley/common/deadline"
	"github.com/dappley/go-dappley/core/blockchain"
	"time"

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
	bm       *lblockchain.BlockchainManager
	con      Consensus
	producer *blockproducerinfo.BlockProducerInfo
	stopCh   chan bool
}

//NewBlockProducer returns a new block producer instance
func NewBlockProducer(bm *lblockchain.BlockchainManager, con Consensus, producer *blockproducerinfo.BlockProducerInfo) *BlockProducer {
	return &BlockProducer{
		bm:       bm,
		con:      con,
		producer: producer,
		stopCh:   make(chan bool, 1),
	}
}

//Start starts the block producing process
func (bp *BlockProducer) Start() {
	go func() {
		logger.Info("BlockProducer Starts...")
		for {
			select {
			case <-bp.stopCh:
				return
			default:
				bp.con.ProduceBlock(bp.produceBlock)
			}
		}
	}()
}

//Stop stops the block producing process
func (bp *BlockProducer) Stop() {
	logger.Info("Miner stops...")
	bp.stopCh <- true
}

//IsProducingBlock returns if the local producer is producing a block
func (bp *BlockProducer) IsProducingBlock() bool {
	return !bp.producer.IsIdle()
}

//produceBlock produces a new block and add it to blockchain
func (bp *BlockProducer) produceBlock(processFunc func(*block.Block), deadline deadline.Deadline) {
	// Do not produce block if block pool is syncing
	if bp.bm.Getblockchain().GetState() != blockchain.BlockchainReady {
		logger.Info("BlockProducer: block producer paused because block pool is syncing.")
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

//prepareBlock generates a new block
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
	cbtx := transaction.NewCoinbaseTX(account.NewAddress(bp.producer.Beneficiary()), "", bp.bm.Getblockchain().GetMaxHeight()+1, totalTips)
	validTxs = append(validTxs, &cbtx)
	utxoIndex.UpdateUtxo(&cbtx)

	logger.WithFields(logger.Fields{
		"valid_txs": len(validTxs),
	}).Info("BlockProducer: prepared a block.")

	ctx := lblockchain.BlockContext{Block: block.NewBlock(validTxs, parentBlock, bp.producer.Beneficiary()), UtxoIndex: utxoIndex, State: state}
	return &ctx
}

//collectTransactions pack transactions from transaction pool to a new block
func (bp *BlockProducer) collectTransactions(utxoIndex *lutxo.UTXOIndex, parentBlk *block.Block, deadline deadline.Deadline) ([]*transaction.Transaction, *scState.ScState) {

	var validTxs []*transaction.Transaction
	totalSize := 0
	count := 0

	scStorage := scState.LoadScStateFromDatabase(bp.bm.Getblockchain().GetDb())
	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	rewards := make(map[string]string)
	currBlkHeight := parentBlk.GetHeight() + 1

	for totalSize < bp.bm.Getblockchain().GetBlockSizeLimit() && bp.bm.Getblockchain().GetTxPool().GetNumOfTxInPool() > 0 && !deadline.IsPassed() {

		txNode := bp.bm.Getblockchain().GetTxPool().PopTransactionWithMostTips(utxoIndex)
		if txNode == nil {
			break
		}
		totalSize += txNode.Size
		count++

		ctx := txNode.Value.ToContractTx()
		minerAddr := account.NewAddress(bp.producer.Beneficiary())
		minerTA := account.NewContractAccountByAddress(minerAddr)
		if ctx != nil {
			prevUtxos, err := lutxo.FindVinUtxosInUtxoPool(*utxoIndex, ctx.Transaction)
			if err != nil {
				logger.WithError(err).WithFields(logger.Fields{
					"txid": hex.EncodeToString(ctx.ID),
				}).Warn("BlockProducer: cannot find vin while executing smart contract")
				return nil, nil
			}
			isContractDeployed := ltransaction.IsContractDeployed(utxoIndex, ctx)
			validTxs = append(validTxs, txNode.Value)
			utxoIndex.UpdateUtxo(txNode.Value)

			gasCount, generatedTxs, err := ltransaction.Execute(ctx, prevUtxos, isContractDeployed, *utxoIndex, scStorage, rewards, engine, currBlkHeight, parentBlk)

			if err != nil {
				logger.WithError(err).Error("BlockProducer: executeSmartContract error.")
			}
			// record gas used
			if gasCount > 0 {
				grtx, err := transaction.NewGasRewardTx(minerTA, currBlkHeight, common.NewAmount(gasCount), ctx.GasPrice, count)
				if err == nil {
					generatedTxs = append(generatedTxs, &grtx)
				}
			}
			gctx, err := transaction.NewGasChangeTx(ctx.GetDefaultFromTransactionAccount(), currBlkHeight, common.NewAmount(gasCount), ctx.GasLimit, ctx.GasPrice, count)
			if err == nil {
				generatedTxs = append(generatedTxs, &gctx)
			}
			validTxs = append(validTxs, generatedTxs...)
			utxoIndex.UpdateUtxos(generatedTxs)
		} else {
			validTxs = append(validTxs, txNode.Value)
			utxoIndex.UpdateUtxo(txNode.Value)
		}
	}

	// append reward transaction
	if len(rewards) > 0 {
		rtx := transaction.NewRewardTx(currBlkHeight, rewards)
		validTxs = append(validTxs, &rtx)
		utxoIndex.UpdateUtxo(&rtx)
	}

	return validTxs, scStorage
}

//calculateTips calculate how much tips are earned from the input transactions
func (bp *BlockProducer) calculateTips(txs []*transaction.Transaction) *common.Amount {
	//calculate tips
	totalTips := common.NewAmount(0)
	for _, tx := range txs {
		totalTips = totalTips.Add(tx.Tip)
	}
	return totalTips
}

//addBlockToBlockchain adds the new block to blockchain
func (bp *BlockProducer) addBlockToBlockchain(ctx *lblockchain.BlockContext) {
	logger.WithFields(logger.Fields{
		"height": ctx.Block.GetHeight(),
		"hash":   ctx.Block.GetHash().String(),
	}).Info("BlockProducer: produced a new block.")
	if !lblock.VerifyHash(ctx.Block) {
		logger.Warn("BlockProducer: hash of the new block is invalid.")
		return
	}

	if !bp.bm.Getblockchain().CheckLibPolicy(ctx.Block) {
		logger.Warn("BlockProducer: the number of producers is not enough.")
		tailBlock, _ := bp.bm.Getblockchain().GetTailBlock()
		bp.bm.BroadcastBlock(tailBlock)
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
	}

	bp.bm.BroadcastBlock(ctx.Block)
	logger.Info("BlockProducer: Broadcast block")
}
