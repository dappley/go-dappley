package consensus

import (
	"math"
	"math/big"

	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

const defaulttargetBits = 14

type State int

var maxNonce int64 = math.MaxInt64

type MinedBlock struct {
	block   *core.Block
	isValid bool
}

type Miner struct {
	target   *big.Int
	exitCh   chan bool
	bc       *core.Blockchain
	cbAddr   string
	newBlock *MinedBlock
	nonce    int64
	retChan  chan (*MinedBlock)
}

func NewMiner() *Miner {
	m := &Miner{
		target:   nil,
		exitCh:   make(chan bool, 1),
		bc:       nil,
		cbAddr:   "",
		newBlock: &MinedBlock{nil, false},
		nonce:    0,
	}
	m.SetTargetBit(defaulttargetBits)
	return m
}

func (miner *Miner) SetTargetBit(bit int) {
	if bit <= 0 || bit > 256 {
		return
	}
	target := big.NewInt(1)
	miner.target = target.Lsh(target, uint(256-bit))
}

func (miner *Miner) Setup(bc *core.Blockchain, cbAddr string, retChan chan (*MinedBlock)) {
	miner.bc = bc
	miner.cbAddr = cbAddr
	miner.retChan = retChan
}

func (miner *Miner) Start() {
	go func() {
		logger.Info("Miner: Start Mining A Block...")
		miner.resetExitCh()
		miner.prepare()
		nonce := int64(0)
	hashLoop:
		for {
			select {
			case <-miner.exitCh:
				break hashLoop
			default:
				if nonce < maxNonce {
					if ok := miner.mineBlock(nonce); ok {
						break hashLoop
					} else {
						nonce++
					}
				} else {
					break hashLoop
				}
			}
		}
		miner.returnBlk()
		logger.Info("Miner: Mining Ends...")
	}()
}

func (miner *Miner) Stop() {
	if len(miner.exitCh) == 0 {
		miner.exitCh <- true
	}
}

func (miner *Miner) Validate(blk *core.Block) bool {
	var hashInt big.Int

	hash := blk.GetHash()
	hashInt.SetBytes(hash)

	isValid := hashInt.Cmp(miner.target) == -1

	return isValid
}

func (miner *Miner) prepare() {
	miner.newBlock = miner.prepareBlock()
}

func (miner *Miner) returnBlk() {
	if !miner.newBlock.isValid {
		miner.newBlock.block.Rollback(miner.bc.TxnPool())
	}
	miner.retChan <- miner.newBlock
}

func (miner *Miner) resetExitCh() {
	if len(miner.exitCh) > 0 {
		<-miner.exitCh
	}
}

func (miner *Miner) prepareBlock() *MinedBlock {

	parentBlock, err := miner.bc.GetTailBlock()
	if err != nil {
		logger.Error(err)
	}

	//verify all transactions
	//miner.verifyTransactions()
	//get all transactions
	txs := miner.bc.TxnPool().GetSortedTransactions()
	//add coinbase transaction to transaction pool
	cbtx := core.NewCoinbaseTX(miner.cbAddr, "")
	txs = append(txs, &cbtx)

	miner.nonce = 0
	//prepare the new block (without the correct nonce value)
	return &MinedBlock{core.NewBlock(txs, parentBlock), false}
}

//returns true if a block is mined; returns false if the nonce value does not satisfy the difficulty requirement
func (miner *Miner) mineBlock(nonce int64) bool {
	hash, ok := miner.verifyNonce(nonce, miner.newBlock.block)
	if ok {
		miner.newBlock.block.SetHash(hash)
		miner.newBlock.block.SetNonce(nonce)
		miner.newBlock.isValid = true
	}
	return ok
}

func (miner *Miner) verifyNonce(nonce int64, blk *core.Block) (core.Hash, bool) {
	var hashInt big.Int
	var hash core.Hash

	hash = blk.CalculateHashWithNonce(nonce)
	hashInt.SetBytes(hash[:])

	return hash, hashInt.Cmp(miner.target) == -1
}

//verify transactions and remove invalid transactions
func (miner *Miner) verifyTransactions() {
	utxoPool := core.GetStoredUtxoMap(miner.bc.DB, core.UtxoMapKey)
	txPool := miner.bc.TxnPool()
	txPool.FilterAllTransactions(utxoPool)
}
