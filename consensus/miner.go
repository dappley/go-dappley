package consensus

import (
	"github.com/dappworks/go-dappworks/core"
)

type state int

const (
	prepareTxPoolState state = iota
	mineState
	updateNewBlock
	cleanUpState
)

type Miner struct {
	bc           *core.Blockchain
	txPool       []*core.Transaction
	newBlock     *core.Block
	coinBaseAddr string
	nextState    state
}

//create a new instance
func NewMiner(txs []*core.Transaction, bc *core.Blockchain, coinBaseAddr string) *Miner {

	return &Miner{
		bc,
		txs,
		nil,
		coinBaseAddr,
		prepareTxPoolState,
	}
}

//start mining
func (pd *Miner) Start() {
	pd.run()
}

func (pd *Miner) UpdateTxPool(txs []*core.Transaction) {
	pd.txPool = txs
}

//start the state machine
func (pd *Miner) run() {

Loop:
	for {
		switch pd.nextState {
		case prepareTxPoolState:
			pd.prepareTxPool()
			pd.nextState = mineState

		case mineState:
			pd.mine()
			pd.nextState = updateNewBlock
		case updateNewBlock:
			pd.updateNewBlock()
			pd.nextState = cleanUpState
		case cleanUpState:
			pd.cleanUp()
			break Loop
		}
	}
}

//prepare transaction pool
func (pd *Miner) prepareTxPool() {
	// verify all transactions
	pd.verifyTransactions()

	// add coinbase transaction
	cbtx := core.NewCoinbaseTX(pd.coinBaseAddr, "")
	pd.txPool = append([]*core.Transaction{cbtx}, pd.txPool...)

}

//start proof of work process
func (pd *Miner) mine() {

	//get the hash of last newBlock
	lastHash, err := pd.bc.GetLastHash()
	if err != nil {
		//TODU
	}

	//create a new newBlock with the transaction pool and last hasth
	pd.newBlock = core.NewBlock(pd.txPool, lastHash)
	pow := core.NewProofOfWork(pd.newBlock)
	nonce, hash := pow.Run()
	pd.newBlock.SetHash(hash[:])
	pd.newBlock.SetNonce(nonce)
}

//update the blockchain with the new block
func (pd *Miner) updateNewBlock() {

	pd.txPool = nil
	pd.bc.UpdateNewBlock(pd.newBlock)

}

func (pd *Miner) cleanUp() {
	pd.txPool = nil
	pd.nextState = prepareTxPoolState
}

//verify transactions and remove invalid transactions
func (pd *Miner) verifyTransactions() {
	for i, tx := range pd.txPool {
		if pd.bc.VerifyTransaction(tx) != true {
			//Remove transaction from transaction pool if the transaction is not verified
			pd.txPool = append(pd.txPool[0:i], pd.txPool[i+1:len(pd.txPool)]...)
		}
	}
}
