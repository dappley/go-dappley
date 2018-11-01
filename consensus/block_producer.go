package consensus

import (
	"math"

	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
)

var maxNonce int64 = math.MaxInt64

type NewBlock struct {
	*core.Block
	IsValid bool
}

// Requirement inspects the given block and returns true if it fulfills the requirement
type Requirement func(block *core.Block) bool

var noRequirement = func(block *core.Block) bool { return true }

type BlockProducer struct {
	bc          *core.Blockchain
	beneficiary string
	key         string
	newBlock    *NewBlock
	nonce       int64
	requirement Requirement
	exitCh      chan bool
	newBlockCh  chan *NewBlock
	idle        bool
}

func NewBlockProducer() *BlockProducer {
	return &BlockProducer{
		exitCh:      make(chan bool, 1),
		bc:          nil,
		beneficiary: "",
		newBlock:    &NewBlock{nil, false},
		requirement: noRequirement,
		idle:        true,
	}
}

// Setup tells the producer to give rewards to beneficiaryAddr and return the new block through newBlockCh
func (bp *BlockProducer) Setup(bc *core.Blockchain, beneficiaryAddr string, newBlockCh chan *NewBlock) {
	bp.bc = bc
	bp.beneficiary = beneficiaryAddr
	bp.newBlockCh = newBlockCh
}

func (bp *BlockProducer) SetPrivateKey(key string) {
	bp.key = key
}

// Beneficiary returns the address which receives rewards
func (bp *BlockProducer) Beneficiary() string {
	return bp.beneficiary
}

// SetRequirement defines the requirement that a new block must fulfill
func (bp *BlockProducer) SetRequirement(requirement Requirement) {
	bp.requirement = requirement
}

// Start commences the block production process in a background thread
func (bp *BlockProducer) Start() {
	go func() {
		if bp.bc.GetBlockPool().GetSyncState() {
			return
		}
		logger.Info("BlockProducer: Start Producing A Block...")
		bp.resetExitCh()
		bp.idle = false
		bp.prepareBlock()
		nonce := int64(0)
	hashLoop:
		for {
			select {
			case <-bp.exitCh:
				break hashLoop
			default:
				if nonce < maxNonce {
					if ok := bp.produceBlock(nonce); ok {
						break hashLoop
					} else {
						nonce++
					}
				} else {
					break hashLoop
				}
			}
		}
		bp.returnBlk()
		bp.idle = true
		logger.Info("BlockProducer: Block Produced")
	}()
}

func (bp *BlockProducer) Stop() {
	if len(bp.exitCh) == 0 {
		bp.exitCh <- true
	}
}

func (bp *BlockProducer) IsIdle() bool {
	return bp.idle
}

func (bp *BlockProducer) returnBlk() {
	if !bp.newBlock.IsValid {
		bp.newBlock.Rollback(bp.bc.GetTxPool())
	}
	bp.newBlockCh <- bp.newBlock
}

func (bp *BlockProducer) resetExitCh() {
	if len(bp.exitCh) > 0 {
		<-bp.exitCh
	}
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
			validTxs = append(validTxs[:i], validTxs[i + 1:]...)
		}
	}

	totalTips := common.NewAmount(0)
	for _, tx := range validTxs {
		totalTips = totalTips.Add(common.NewAmount(tx.Tip))
	}

	cbtx := core.NewCoinbaseTX(bp.beneficiary, "", bp.bc.GetMaxHeight()+1, totalTips)
	validTxs = append(validTxs, &cbtx)

	bp.nonce = 0
	bp.newBlock = &NewBlock{core.NewBlock(validTxs, parentBlock), false}
}

// produceBlock returns true if a block is mined; returns false if the nonce value does not satisfy the difficulty requirement
func (bp *BlockProducer) produceBlock(nonce int64) bool {
	hash := bp.newBlock.CalculateHashWithNonce(nonce)
	bp.newBlock.SetHash(hash)
	bp.newBlock.SetNonce(nonce)
	fulfilled := bp.requirement(bp.newBlock.Block)
	if fulfilled {
		hash = bp.newBlock.CalculateHashWithoutNonce()
		bp.newBlock.SetHash(hash)
		if len(bp.key) > 0 {
			signed := bp.newBlock.SignBlock(bp.key, hash)
			if !signed {
				logger.Warn("Miner Key= ", bp.key)
				return false
			}
		}
		bp.newBlock.IsValid = true
	}
	return fulfilled
}
