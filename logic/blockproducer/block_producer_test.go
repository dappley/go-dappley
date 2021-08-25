package blockproducer

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/blockproducerinfo"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transactionbase"
	"github.com/dappley/go-dappley/logic/lblockchain"
	"github.com/dappley/go-dappley/logic/lblockchain/mocks"
	"github.com/dappley/go-dappley/logic/transactionpool"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewBlockProducer(t *testing.T) {
	bm := &lblockchain.BlockchainManager{}
	producer := blockproducerinfo.NewBlockProducerInfo("test")
	con := consensus.NewDPOS(producer)

	blockProducer := NewBlockProducer(bm, con, producer)

	assert.Equal(t, bm, blockProducer.bm)
	assert.Equal(t, con, blockProducer.con)
	assert.Equal(t, producer, blockProducer.producer)
	assert.Equal(t, 1, cap(blockProducer.stopCh))
	assert.False(t, blockProducer.isRunning)
}

func TestBlockProducer_Start(t *testing.T) {
	producer := blockproducerinfo.NewBlockProducerInfo("test")
	con := consensus.NewDPOS(producer)
	con.SetDynasty(consensus.NewDynasty([]string{"test"}, 1, 1))

	libPolicy := &mocks.LIBPolicy{}
	libPolicy.On("GetProducers").Return(nil)
	libPolicy.On("GetMinConfirmationNum").Return(6)
	libPolicy.On("IsBypassingLibCheck").Return(true)
	acc := account.NewAccount()
	db := storage.NewRamStorage()
	bc := lblockchain.CreateBlockchain(acc.GetAddress(), db, libPolicy, transactionpool.NewTransactionPool(nil, 128000), 100000)
	bm := lblockchain.NewBlockchainManager(bc, nil, nil, con)

	blockProducer := NewBlockProducer(bm, con, producer)
	assert.False(t, blockProducer.isRunning)

	blockProducer.Start()
	time.Sleep(time.Millisecond * 500)
	assert.True(t, blockProducer.isRunning)

	blockProducer.stopCh <- true
	//blockProducer.con.(*consensus.DPOS).Stop()
	util.WaitDoneOrTimeout(func() bool {
		return !blockProducer.isRunning
	}, 5)
	assert.False(t, blockProducer.isRunning)
}

func TestBlockProducer_Stop(t *testing.T) {
	bm := &lblockchain.BlockchainManager{}
	producer := blockproducerinfo.NewBlockProducerInfo("test")
	con := consensus.NewDPOS(producer)

	blockProducer := NewBlockProducer(bm, con, producer)
	blockProducer.Stop()
	stopValue := <-blockProducer.stopCh
	assert.True(t, stopValue)
}

func TestBlockProducer_IsProducingBlock(t *testing.T) {
	bm := &lblockchain.BlockchainManager{}
	producer := blockproducerinfo.NewBlockProducerInfo("test")
	con := consensus.NewDPOS(producer)

	blockProducer := NewBlockProducer(bm, con, producer)
	assert.False(t, blockProducer.IsProducingBlock())
	blockProducer.producer.BlockProduceStart()
	assert.True(t, blockProducer.IsProducingBlock())
	blockProducer.producer.BlockProduceFinish()
	assert.False(t, blockProducer.IsProducingBlock())
}

func TestBlockProducer_GetProduceBlockStatus(t *testing.T) {
	bm := &lblockchain.BlockchainManager{}
	producer := blockproducerinfo.NewBlockProducerInfo("test")
	con := consensus.NewDPOS(producer)

	blockProducer := NewBlockProducer(bm, con, producer)
	assert.False(t, blockProducer.GetProduceBlockStatus())
	blockProducer.isRunning = true
	assert.True(t, blockProducer.GetProduceBlockStatus())
}

func TestCalculateTips(t *testing.T) {
	txs := []*transaction.Transaction{
		{
			ID:       []byte{0x4f, 0xda, 0x27, 0xf8, 0x2c, 0xa, 0x49, 0x9d, 0x8c, 0x19, 0x37, 0x46, 0x2c, 0x19, 0x9, 0xc6, 0x96, 0x54, 0x31, 0x99, 0x9f, 0x1f, 0xc6, 0x84, 0xf5, 0xc0, 0xc5, 0x6b, 0xbd, 0xbd, 0xe, 0xc8},
			Vin:      []transactionbase.TXInput{},
			Vout:     []transactionbase.TXOutput{},
			Tip:      common.NewAmount(5),
			GasLimit: common.NewAmount(1024),
			GasPrice: common.NewAmount(1),
		},
		{
			ID:       []byte{0x4f, 0xda, 0x28, 0xf8, 0x2c, 0xa, 0x49, 0x9d, 0x8c, 0x19, 0x37, 0x46, 0x2c, 0x19, 0x9, 0xc6, 0x96, 0x54, 0x31, 0x99, 0x9f, 0x1f, 0xc6, 0x84, 0xf5, 0xc0, 0xc5, 0x6b, 0xbd, 0xbd, 0xe, 0xc8},
			Vin:      []transactionbase.TXInput{},
			Vout:     []transactionbase.TXOutput{},
			Tip:      common.NewAmount(10),
			GasLimit: common.NewAmount(1024),
			GasPrice: common.NewAmount(1),
		},
		{
			ID:       []byte{0x4f, 0xda, 0x29, 0xf8, 0x2c, 0xa, 0x49, 0x9d, 0x8c, 0x19, 0x37, 0x46, 0x2c, 0x19, 0x9, 0xc6, 0x96, 0x54, 0x31, 0x99, 0x9f, 0x1f, 0xc6, 0x84, 0xf5, 0xc0, 0xc5, 0x6b, 0xbd, 0xbd, 0xe, 0xc8},
			Vin:      []transactionbase.TXInput{},
			Vout:     []transactionbase.TXOutput{},
			Tip:      common.NewAmount(15),
			GasLimit: common.NewAmount(1024),
			GasPrice: common.NewAmount(1),
		},
	}

	bm := &lblockchain.BlockchainManager{}
	producer := blockproducerinfo.NewBlockProducerInfo("test")
	con := consensus.NewDPOS(producer)

	blockProducer := NewBlockProducer(bm, con, producer)

	assert.Equal(t, common.NewAmount(30), blockProducer.calculateTips(txs))
}
