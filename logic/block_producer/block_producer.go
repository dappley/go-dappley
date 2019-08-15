package block_producer

import (
	"encoding/hex"
	"time"

	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/logic/block_logic"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/block_producer_info"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	"github.com/dappley/go-dappley/logic/transaction_logic"
	"github.com/dappley/go-dappley/logic/utxo_logic"
	"github.com/dappley/go-dappley/vm"
	logger "github.com/sirupsen/logrus"
)

const (
	maxMintingTimeInMs = 2000
	NanoSecsInMilliSec = 1000000
)

type BlockProducer struct {
	bm       *blockchain_logic.BlockchainManager
	con      Consensus
	producer *block_producer_info.BlockProducerInfo
	stopCh   chan bool
}

func NewBlockProducer(bm *blockchain_logic.BlockchainManager, con Consensus, producer *block_producer_info.BlockProducerInfo) *BlockProducer {
	return &BlockProducer{
		bm:       bm,
		con:      con,
		producer: producer,
		stopCh:   make(chan bool, 1),
	}
}

func (bp *BlockProducer) Start() {
	go func() {
		logger.Info("BlockProducer Starts...")
		bp.con.Start()

		for {
			select {
			case <-bp.stopCh:
				bp.con.Stop()
				return
			case <-bp.con.GetBlockProduceNotifier():
				bp.produceBlock()
			}
		}
	}()
}

func (bp *BlockProducer) Stop() {
	logger.Info("Miner stops...")
	bp.stopCh <- true
}

func (bp *BlockProducer) IsProducingBlock() bool {
	return !bp.producer.IsIdle()
}

func (bp *BlockProducer) Getblockchain() *blockchain_logic.Blockchain {
	if bp.bm == nil {
		return nil
	}
	return bp.bm.Getblockchain()
}

func (bp *BlockProducer) produceBlock() {

	deadlineInMs := time.Now().UnixNano()/NanoSecsInMilliSec + maxMintingTimeInMs

	logger.Infof("BlockProducerer: producing block... ***time is %v***", time.Now().Unix())

	// Do not produce block if block pool is syncing
	if bp.bm.Getblockchain().GetState() != blockchain.BlockchainReady {
		logger.Info("BlockProducer: block producer paused because block pool is syncing.")
		return
	}

	bp.producer.BlockProduceStart()
	defer bp.producer.BlockProduceFinish()

	ctx := bp.generateBlock(deadlineInMs)
	if ctx == nil || !bp.con.Validate(ctx.Block) {
		logger.Error("BlockProducer: produced an invalid block!")
		return
	}

	bp.addBlockToBlockchain(ctx)
}

// generateBlock produces a block by preparing its raw contents and applying the predefined Process to it.
// deadlineInMs = 0 means no deadline
func (bp *BlockProducer) generateBlock(deadlineInMs int64) *blockchain_logic.BlockContext {
	logger.Info("BlockProducerInfo: started producing new block...")
	bp.producer.BlockProduceStart()
	ctx := bp.prepareBlock(deadlineInMs)
	processBlkFunc := bp.con.GetProcess()
	if ctx != nil && processBlkFunc != nil {
		processBlkFunc(ctx.Block)
	}
	return ctx
}

func (bp *BlockProducer) prepareBlock(deadlineInMs int64) *blockchain_logic.BlockContext {
	parentBlock, err := bp.bm.Getblockchain().GetTailBlock()
	if err != nil {
		logger.WithError(err).Error("BlockProducerInfo: cannot get the current tail block!")
		return nil
	}

	// Retrieve all valid transactions from tx pool
	utxoIndex := utxo_logic.NewUTXOIndex(bp.bm.Getblockchain().GetUtxoCache())

	validTxs, state := bp.collectTransactions(utxoIndex, parentBlock, deadlineInMs)

	cbtx := bp.calculateTips(validTxs)
	validTxs = append(validTxs, cbtx)
	utxoIndex.UpdateUtxo(cbtx)

	logger.WithFields(logger.Fields{
		"valid_txs": len(validTxs),
	}).Info("BlockProducer: prepared a block.")

	ctx := blockchain_logic.BlockContext{Block: block.NewBlock(validTxs, parentBlock, bp.producer.Beneficiary()), UtxoIndex: utxoIndex, State: state}
	return &ctx
}

func (bp *BlockProducer) collectTransactions(utxoIndex *utxo_logic.UTXOIndex, parentBlk *block.Block, deadlineInMs int64) ([]*transaction.Transaction, *scState.ScState) {
	var validTxs []*transaction.Transaction
	totalSize := 0

	scStorage := scState.LoadScStateFromDatabase(bp.bm.Getblockchain().GetDb())
	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	rewards := make(map[string]string)
	currBlkHeight := parentBlk.GetHeight() + 1

	for totalSize < bp.bm.Getblockchain().GetBlockSizeLimit() && bp.bm.Getblockchain().GetTxPool().GetNumOfTxInPool() > 0 && !isExceedingDeadline(deadlineInMs) {

		txNode := bp.bm.Getblockchain().GetTxPool().PopTransactionWithMostTips(utxoIndex)
		if txNode == nil {
			break
		}
		totalSize += txNode.Size

		ctx := txNode.Value.ToContractTx()
		minerAddr := account.NewAddress(bp.producer.Beneficiary())
		if ctx != nil {
			prevUtxos, err := utxo_logic.FindVinUtxosInUtxoPool(*utxoIndex, ctx.Transaction)
			if err != nil {
				logger.WithError(err).WithFields(logger.Fields{
					"txid": hex.EncodeToString(ctx.ID),
				}).Warn("BlockProducer: cannot find vin while executing smart contract")
				return nil, nil
			}
			isSCUTXO := (*utxoIndex).GetAllUTXOsByPubKeyHash([]byte(ctx.Vout[0].PubKeyHash)).Size() == 0

			validTxs = append(validTxs, txNode.Value)
			utxoIndex.UpdateUtxo(txNode.Value)

			gasCount, generatedTxs, err := transaction_logic.Execute(ctx, prevUtxos, isSCUTXO, *utxoIndex, scStorage, rewards, engine, currBlkHeight, parentBlk)

			// record gas used
			if err != nil {
				// add utxo from txs into utxoIndex
				logger.WithError(err).Error("executeSmartContract error.")
			}
			if gasCount > 0 {
				grtx, err := transaction.NewGasRewardTx(minerAddr, currBlkHeight, common.NewAmount(gasCount), ctx.GasPrice)
				if err == nil {
					generatedTxs = append(generatedTxs, &grtx)
				}
			}
			gctx, err := transaction.NewGasChangeTx(ctx.GetDefaultFromPubKeyHash().GenerateAddress(), currBlkHeight, common.NewAmount(gasCount), ctx.GasLimit, ctx.GasPrice)
			if err == nil {
				generatedTxs = append(generatedTxs, &gctx)
			}
			validTxs = append(validTxs, generatedTxs...)
			utxoIndex.UpdateUtxoState(generatedTxs)
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

func (bp *BlockProducer) calculateTips(txs []*transaction.Transaction) *transaction.Transaction {
	//calculate tips
	totalTips := common.NewAmount(0)
	for _, tx := range txs {
		totalTips = totalTips.Add(tx.Tip)
	}
	cbtx := transaction.NewCoinbaseTX(account.NewAddress(bp.producer.Beneficiary()), "", bp.bm.Getblockchain().GetMaxHeight()+1, totalTips)
	return &cbtx
}

//executeSmartContract executes all smart contracts
func (bp *BlockProducer) executeSmartContract(utxoIndex *utxo_logic.UTXOIndex,
	txs []*transaction.Transaction, currBlkHeight uint64, parentBlk *block.Block) ([]*transaction.Transaction, *scState.ScState) {
	//start a new smart contract engine

	scStorage := scState.LoadScStateFromDatabase(bp.bm.Getblockchain().GetDb())
	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	var generatedTXs []*transaction.Transaction
	rewards := make(map[string]string)

	minerAddr := account.NewAddress(bp.producer.Beneficiary())

	for _, tx := range txs {
		ctx := tx.ToContractTx()
		if ctx == nil {
			// add utxo from txs into utxoIndex
			utxoIndex.UpdateUtxo(tx)
			continue
		}
		prevUtxos, err := utxo_logic.FindVinUtxosInUtxoPool(*utxoIndex, ctx.Transaction)
		if err != nil {
			logger.WithError(err).WithFields(logger.Fields{
				"txid": hex.EncodeToString(ctx.ID),
			}).Warn("BlockProducer: cannot find vin while executing smart contract")
			return nil, nil
		}
		isSCUTXO := (*utxoIndex).GetAllUTXOsByPubKeyHash([]byte(ctx.Vout[0].PubKeyHash)).Size() == 0
		gasCount, newTxs, err := transaction_logic.Execute(ctx, prevUtxos, isSCUTXO, *utxoIndex, scStorage, rewards, engine, currBlkHeight, parentBlk)
		generatedTXs = append(generatedTXs, newTxs...)
		// record gas used
		if err != nil {
			// add utxo from txs into utxoIndex
			logger.WithFields(logger.Fields{
				"err": err,
			}).Error("executeSmartContract error.")
		}
		if gasCount > 0 {
			grtx, err := transaction.NewGasRewardTx(minerAddr, currBlkHeight, common.NewAmount(gasCount), ctx.GasPrice)
			if err == nil {
				generatedTXs = append(generatedTXs, &grtx)
			}
		}
		gctx, err := transaction.NewGasChangeTx(ctx.GetDefaultFromPubKeyHash().GenerateAddress(), currBlkHeight, common.NewAmount(gasCount), ctx.GasLimit, ctx.GasPrice)
		if err == nil {

			generatedTXs = append(generatedTXs, &gctx)
		}

		// add utxo from txs into utxoIndex
		utxoIndex.UpdateUtxo(tx)
	}
	// append reward transaction
	if len(rewards) > 0 {
		rtx := transaction.NewRewardTx(currBlkHeight, rewards)
		generatedTXs = append(generatedTXs, &rtx)
	}
	return generatedTXs, scStorage
}

func (bp *BlockProducer) addBlockToBlockchain(ctx *blockchain_logic.BlockContext) {
	logger.WithFields(logger.Fields{
		"height": ctx.Block.GetHeight(),
		"hash":   ctx.Block.GetHash().String(),
	}).Info("BlockProducer: produced a new block.")
	if !block_logic.VerifyHash(ctx.Block) {
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
	bp.bm.BroadcastBlock(ctx.Block)
}

func isExceedingDeadline(deadlineInMs int64) bool {
	return deadlineInMs > 0 && time.Now().UnixNano()/1000000 >= deadlineInMs
}
