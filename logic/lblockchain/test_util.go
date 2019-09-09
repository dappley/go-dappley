package lblockchain

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/scState"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/logic/lblock"
	"github.com/dappley/go-dappley/logic/lblockchain/mocks"
	"github.com/dappley/go-dappley/logic/lutxo"
	"github.com/dappley/go-dappley/logic/transactionpool"
	"github.com/dappley/go-dappley/storage"
	"github.com/stretchr/testify/mock"
)

func PrepareBlockContext(bc *Blockchain, blk *block.Block) *BlockContext {
	state := scState.LoadScStateFromDatabase(bc.GetDb())
	utxoIndex := lutxo.NewUTXOIndex(bc.GetUtxoCache())
	utxoIndex.UpdateUtxoState(blk.GetTransactions())
	ctx := BlockContext{Block: blk, UtxoIndex: utxoIndex, State: state}
	return &ctx
}

func GenerateMockBlockchainWithCoinbaseTxOnly(size int) *Blockchain {
	//create a new block chain
	s := storage.NewRamStorage()
	addr := account.NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
	consensus := &mocks.Consensus{}
	consensus.On("GetProducers").Return(nil)
	consensus.On("GetLibProducerNum").Return(6)
	consensus.On("Validate", mock.Anything).Return(true)
	consensus.On("IsBypassingLibCheck").Return(true)
	bc := CreateBlockchain(addr, s, consensus, transactionpool.NewTransactionPool(nil, 128000), nil, 100000)

	for i := 0; i < size; i++ {
		tailBlk, _ := bc.GetTailBlock()
		cbtx := transaction.NewCoinbaseTX(addr, "", bc.GetMaxHeight()+1, common.NewAmount(0))
		b := block.NewBlock([]*transaction.Transaction{&cbtx}, tailBlk, "16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
		b.SetHash(lblock.CalculateHash(b))
		bc.AddBlockContextToTail(PrepareBlockContext(bc, b))
	}
	return bc
}

func AddBlockToGeneratedBlockchain(bc *Blockchain, numOfBlks int) {
	for i := 0; i < numOfBlks; i++ {
		addr := account.NewAddress("16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
		tailBlk, _ := bc.GetTailBlock()
		cbtx := transaction.NewCoinbaseTX(addr, "", bc.GetMaxHeight()+1, common.NewAmount(0))
		b := block.NewBlock([]*transaction.Transaction{&cbtx}, tailBlk, "16PencPNnF8CiSx2EBGEd1axhf7vuHCouj")
		b.SetHash(lblock.CalculateHash(b))
		bc.AddBlockContextToTail(PrepareBlockContext(bc, b))
	}
}
