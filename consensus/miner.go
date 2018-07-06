package consensus

import (
	"log"
	"github.com/dappworks/go-dappworks/core"
	"container/heap"
)

type state int


const (
	prepareTxPoolState state = iota
	mineState
	updateNewBlock
	cleanUpState
)


type Miner struct{
	bc 		  		*core.Blockchain
	newBlock 		*core.Block
	coinBaseAddr 	string
	nextState 		state
}

//create a new instance
func NewMiner(bc *core.Blockchain,coinBaseAddr string) *Miner{

	return &Miner{
		bc,
		nil,
		coinBaseAddr,
		prepareTxPoolState,
	}
}

//start mining
func (pd *Miner) Start(){
	pd.run()
}

func UpdateTxPool(txs core.TransactionPool){
	core.TransactionPoolSingleton = txs
}

//start the state machine
func (pd *Miner) run(){

	Loop:
	for{
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
func (pd *Miner) prepareTxPool(){
	// verify all transactions
	pd.verifyTransactions()

	// add coinbase transaction
	cbtx := core.NewCoinbaseTX(pd.coinBaseAddr,"")
	h := &core.TransactionPool{}
	heap.Init(h)
	heap.Push(&core.TransactionPoolSingleton, cbtx)

}

//start proof of work process
func (pd *Miner) mine(){

	//get the hash of last newBlock
	lastHash, err := pd.bc.GetLastHash()
	if err != nil {
		log.Panic(err)
	}

	//create a new newBlock with the transaction pool and last hasth
	pd.newBlock = core.NewBlock(lastHash)
	pow := core.NewProofOfWork(pd.newBlock)
	nonce, hash := pow.Run()
	pd.newBlock.SetHash(hash[:])
	pd.newBlock.SetNonce(nonce)
}

//update the blockchain with the new block
func (pd *Miner) updateNewBlock(){

	err := pd.bc.UpdateNewBlock(pd.newBlock)
	if err != nil {
		log.Panic(err)
	}
}

func (pd *Miner) cleanUp(){

	pd.nextState = prepareTxPoolState
}

//verify transactions and remove invalid transactions
func (pd *Miner) verifyTransactions() {
	//for TransactionPool.Len() > 0 {
	//
	//	var txn = heap.Pop(&TransactionPool).(core.Transaction)
	//
	//	//if pd.bc.VerifyTransaction(txn) != true {
	//	//	//Remove transaction from transaction pool if the transaction is not verified
	//	//	pd.txPool = append(pd.txPool[0:i],pd.txPool[i+1:len(pd.txPool)]...)
	//	//}
	//}
	//}
}