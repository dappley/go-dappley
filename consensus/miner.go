package consensus

import (
	"math/big"
	"github.com/dappley/go-dappley/core"
	logger "github.com/sirupsen/logrus"
)

const defaulttargetBits = 14

type State int

const (
	prepareBlockState   State = iota
	mineBlockState
)

type MinedBlock struct{
	block 		*core.Block
	isValid 	bool
}

type Miner struct{
	target    	*big.Int
	exitCh    	chan bool
	bc        	*core.Blockchain
	nextState 	State
	cbAddr    	string
	newBlock  	*MinedBlock
	nonce	   	int64
	retChan 	chan(*MinedBlock)
}

func NewMiner() *Miner{
	m := &Miner{
		target: 		nil,
		exitCh: 		make(chan bool, 1),
		bc:     		nil,
		nextState:		prepareBlockState,
		cbAddr: 		"",
		newBlock:		nil,
		nonce:			0,
	}
	m.SetTargetBit(defaulttargetBits)
	return m
}

func (miner *Miner) SetTargetBit(bit int){
	if bit <= 0 || bit > 256 {
		return
	}
	target := big.NewInt(1)
	miner.target = target.Lsh(target, uint(256-bit))
}

func (miner *Miner) Setup(bc *core.Blockchain, cbAddr string, retChan chan(*MinedBlock)){
	miner.bc = bc
	miner.cbAddr = cbAddr
	miner.retChan = retChan
}

func (miner *Miner) Start() {
	go func() {
		logger.Debug("Miner: Start Mining A Block...")
		miner.nextState = prepareBlockState
		for {
			select {
			case <-miner.exitCh:
				logger.Debug("Miner: Mining Ends...")

				return
			default:
				miner.runNextState()
			}
		}
	}()
}

func (miner *Miner) Stop() {
	miner.exitCh <- true
	if !miner.newBlock.isValid {
		miner.newBlock.block.Rollback()
	}else{
		miner.retChan <- miner.newBlock
	}
}

func (miner *Miner) Validate(blk *core.Block) bool {
	var hashInt big.Int

	hash := blk.GetHash()
	hashInt.SetBytes(hash)

	isValid := hashInt.Cmp(miner.target) == -1

	return isValid
}


func (miner *Miner) runNextState(){
	switch miner.nextState {
	case prepareBlockState:
		miner.newBlock = &MinedBlock{}
		miner.newBlock.block = miner.prepareBlock()
		miner.newBlock.isValid = false
		miner.nextState = mineBlockState
	case mineBlockState:
		if miner.nonce < maxNonce {
			if ok := miner.mineBlock(); ok {
				miner.Stop()
			}
		}else{
			miner.Stop()
		}
	}
}

func (miner *Miner) prepareBlock() *core.Block{

	parentBlock,err := miner.bc.GetTailBlock()
	if err!=nil {
		logger.Error(err)
	}

	//verify all transactions
	miner.verifyTransactions()
	//get all transactions
	txs := core.GetTxnPoolInstance().GetSortedTransactions()
	//add coinbase transaction to transaction pool
	cbtx := core.NewCoinbaseTX(miner.cbAddr, "")
	txs = append(txs, &cbtx)

	miner.nonce = 0
	//prepare the new block (without the correct nonce value)
	return core.NewBlock(txs, parentBlock)
}

//returns true if a block is mined; returns false if the nonce value does not satisfy the difficulty requirement
func (miner *Miner) mineBlock() bool{
	hash, ok := miner.verifyNonce(miner.nonce, miner.newBlock.block)
	if ok {
		miner.newBlock.block.SetHash(hash)
		miner.newBlock.block.SetNonce(miner.nonce)
		miner.newBlock.isValid = true
	}else{
		miner.nonce ++
	}
	return ok
}

func (miner *Miner) verifyNonce(nonce int64, blk *core.Block) (core.Hash, bool){
	var hashInt big.Int
	var hash core.Hash

	hash = blk.CalculateHashWithNonce(nonce)
	hashInt.SetBytes(hash[:])

	return hash, hashInt.Cmp(miner.target) == -1
}

//verify transactions and remove invalid transactions
func (miner *Miner) verifyTransactions() {
	utxoPool := core.GetStoredUtxoMap(miner.bc.DB, core.UtxoMapKey)
	txPool := core.GetTxnPoolInstance()
	txPool.FilterAllTransactions(utxoPool)
}