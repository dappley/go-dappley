package consensus

import (
	"encoding/hex"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/logic/blockchain_logic"
	"github.com/dappley/go-dappley/logic/transaction_logic"
	"github.com/dappley/go-dappley/logic/utxo_logic"
	"github.com/dappley/go-dappley/vm"
	logger "github.com/sirupsen/logrus"
)

// process defines the procedure to produce a valid block modified from a raw (unhashed/unsigned) block
type process func(ctx *blockchain_logic.BlockContext)

type BlockProducerLogic struct {
	bc       *blockchain_logic.Blockchain
	process  process
	producer *BlockProducerInfo
}

func NewBlockProducerLogic() *BlockProducerLogic {
	return &BlockProducerLogic{
		bc:       nil,
		process:  nil,
		producer: NewBlockProducerInfo(),
	}
}

func (bp *BlockProducerLogic) Setup(bc *blockchain_logic.Blockchain, beneficiaryAddr string) {
	bp.bc = bc
	bp.producer.Setup(beneficiaryAddr)
}

// SetProcess tells the producer to follow the given process to produce a valid block
func (bp *BlockProducerLogic) SetProcess(process process) {
	bp.process = process
}

// ProduceBlock produces a block by preparing its raw contents and applying the predefined process to it.
// deadlineInMs = 0 means no deadline
func (bp *BlockProducerLogic) ProduceBlock(deadlineInMs int64) *blockchain_logic.BlockContext {
	logger.Info("BlockProducerInfo: started producing new block...")
	bp.producer.BlockProduceStart()
	ctx := bp.prepareBlock(deadlineInMs)
	if ctx != nil && bp.process != nil {
		bp.process(ctx)
	}
	return ctx
}

func (bp *BlockProducerLogic) prepareBlock(deadlineInMs int64) *blockchain_logic.BlockContext {
	parentBlock, err := bp.bc.GetTailBlock()
	if err != nil {
		logger.WithError(err).Error("BlockProducerInfo: cannot get the current tail block!")
		return nil
	}

	// Retrieve all valid transactions from tx pool
	utxoIndex := utxo_logic.NewUTXOIndex(bp.bc.GetUtxoCache())

	validTxs, state := bp.collectTransactions(utxoIndex, parentBlock, deadlineInMs)

	cbtx := bp.calculateTips(validTxs)
	validTxs = append(validTxs, cbtx)
	utxoIndex.UpdateUtxo(cbtx)

	logger.WithFields(logger.Fields{
		"valid_txs": len(validTxs),
	}).Info("BlockProducerInfo: prepared a block.")

	ctx := blockchain_logic.BlockContext{Block: block.NewBlock(validTxs, parentBlock, bp.producer.beneficiary), UtxoIndex: utxoIndex, State: state}
	return &ctx
}

func (bp *BlockProducerLogic) collectTransactions(utxoIndex *utxo_logic.UTXOIndex, parentBlk *block.Block, deadlineInMs int64) ([]*transaction.Transaction, *scState.ScState) {
	var validTxs []*transaction.Transaction
	totalSize := 0

	scStorage := scState.LoadScStateFromDatabase(bp.bc.GetDb())
	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	rewards := make(map[string]string)
	currBlkHeight := parentBlk.GetHeight() + 1

	for totalSize < bp.bc.GetBlockSizeLimit() && bp.bc.GetTxPool().GetNumOfTxInPool() > 0 && !isExceedingDeadline(deadlineInMs) {

		txNode := bp.bc.GetTxPool().PopTransactionWithMostTips(utxoIndex)
		if txNode == nil {
			break
		}
		totalSize += txNode.Size

		ctx := txNode.Value.ToContractTx()
		minerAddr := account.NewAddress(bp.producer.beneficiary)
		if ctx != nil {
			prevUtxos, err := utxo_logic.FindVinUtxosInUtxoPool(*utxoIndex, ctx.Transaction)
			if err != nil {
				logger.WithError(err).WithFields(logger.Fields{
					"txid": hex.EncodeToString(ctx.ID),
				}).Warn("Transaction: cannot find vin while executing smart contract")
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

func (bp *BlockProducerLogic) calculateTips(txs []*transaction.Transaction) *transaction.Transaction {
	//calculate tips
	totalTips := common.NewAmount(0)
	for _, tx := range txs {
		totalTips = totalTips.Add(tx.Tip)
	}
	cbtx := transaction_logic.NewCoinbaseTX(account.NewAddress(bp.producer.beneficiary), "", bp.bc.GetMaxHeight()+1, totalTips)
	return &cbtx
}

//executeSmartContract executes all smart contracts
func (bp *BlockProducerLogic) executeSmartContract(utxoIndex *utxo_logic.UTXOIndex,
	txs []*transaction.Transaction, currBlkHeight uint64, parentBlk *block.Block) ([]*transaction.Transaction, *scState.ScState) {
	//start a new smart contract engine

	scStorage := scState.LoadScStateFromDatabase(bp.bc.GetDb())
	engine := vm.NewV8Engine()
	defer engine.DestroyEngine()
	var generatedTXs []*transaction.Transaction
	rewards := make(map[string]string)

	minerAddr := account.NewAddress(bp.producer.beneficiary)

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
			}).Warn("Transaction: cannot find vin while executing smart contract")
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

// Beneficiary returns the address which receives rewards
func (bp *BlockProducerLogic) Beneficiary() string {
	return bp.producer.Beneficiary()
}

func (bp *BlockProducerLogic) BlockProduceFinish() {
	bp.producer.BlockProduceFinish()
}

func (bp *BlockProducerLogic) IsIdle() bool {
	return bp.producer.IsIdle()
}

func (bp *BlockProducerLogic) Produced(blk *block.Block) bool {
	return bp.producer.Produced(blk)
}
