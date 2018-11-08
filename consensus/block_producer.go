package consensus

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

// process defines the procedure to produce a valid block modified from a raw (unhashed/unsigned) block
type process func(block *core.Block)

type BlockProducer struct {
	bc          *core.Blockchain
	beneficiary string
	newBlock    *core.Block
	process     process
	idle        bool
}

func NewBlockProducer() *BlockProducer {
	return &BlockProducer{
		bc:          nil,
		beneficiary: "",
		process:     nil,
		newBlock:    nil,
		idle:        true,
	}
}

// Setup tells the producer to give rewards to beneficiaryAddr and return the new block through newBlockCh
func (bp *BlockProducer) Setup(bc *core.Blockchain, beneficiaryAddr string) {
	bp.bc = bc
	bp.beneficiary = beneficiaryAddr
}

// Beneficiary returns the address which receives rewards
func (bp *BlockProducer) Beneficiary() string {
	return bp.beneficiary
}

// SetProcess tells the producer to follow the given process to produce a valid block
func (bp *BlockProducer) SetProcess(process process) {
	bp.process = process
}

// ProduceBlock produces a block by preparing its raw contents and applying the predefined process to it
func (bp *BlockProducer) ProduceBlock() *core.Block {

	bp.idle = false
	bp.prepareBlock()
	if bp.process != nil {
		bp.process(bp.newBlock)
	}
	bp.idle = true
	return bp.newBlock
}

func (bp *BlockProducer) IsIdle() bool {
	return bp.idle
}

func (bp *BlockProducer) prepareBlock() {
	parentBlock, err := bp.bc.GetTailBlock()
	if err != nil {
		logger.Error(err)
	}

	// Retrieve all valid transactions from tx pool
	utxoIndex := core.LoadUTXOIndex(bp.bc.GetDb())
	validTxs := bp.bc.GetTxPool().GetValidTxs(utxoIndex)

	// update UTXO set
	for i, tx := range validTxs {
		// remove transaction if utxo set cannot be updated
		if !utxoIndex.UpdateUtxo(tx) {
			validTxs = append(validTxs[:i], validTxs[i+1:]...)
		}
	}

	totalTips := common.NewAmount(0)
	for _, tx := range validTxs {
		totalTips = totalTips.Add(common.NewAmount(tx.Tip))
	}

	cbtx := core.NewCoinbaseTX(bp.beneficiary, "", bp.bc.GetMaxHeight()+1, totalTips)
	validTxs = append(validTxs, &cbtx)

	bp.newBlock = core.NewBlock(validTxs, parentBlock)
}
