package consensus

import (
	"testing"
	"github.com/dappworks/go-dappworks/client"
	"github.com/dappworks/go-dappworks/core"
	"os"
	"github.com/dappworks/go-dappworks/storage"
	"github.com/stretchr/testify/assert"
	"github.com/dappworks/go-dappworks/util"
)

var sendAmount = int(5)
var mineAward = int(10)


//mine one transaction
func TestMiner_SingleValidTx(t *testing.T) {

	setup()

	//create new wallet
	wallets, _ := client.NewWallets()
	assert.NotNil(t,wallets)

	addr1 := wallets.CreateWallet()
	assert.NotNil(t, addr1)

	addr2:= wallets.CreateWallet()
	assert.NotNil(t, addr2)

	wallet := wallets.GetWallet(addr1)

	//create a blockchain
	assert.Equal(t,true,core.ValidateAddress(addr1))
	bc := core.CreateBlockchain(addr1)

	assert.NotNil(t, bc)
	defer bc.DB.Close()

	//check balance
	checkBalance(t, addr1, addr2, bc, mineAward, 0)

	//create 2 transactions and start mining
	tx, err := core.NewUTXOTransaction(addr1, addr2, sendAmount, wallet, bc)
	assert.Nil(t, err)
	txs := []*core.Transaction{tx}

	miner := NewMiner(txs, bc, addr1)
	miner.Start()

	checkBalance(t, addr1, addr2, bc, mineAward*2-sendAmount, sendAmount)

	teardown()
}

//mine empty blocks
func TestMiner_MineEmptyBlock(t *testing.T) {

	setup()

	//create new wallet
	wallets, _ := client.NewWallets()
	assert.NotNil(t,wallets)

	addr1 := wallets.CreateWallet()
	assert.NotNil(t, addr1)

	addr2:= wallets.CreateWallet()
	assert.NotNil(t, addr2)

	//create a blockchain
	assert.Equal(t,true,core.ValidateAddress(addr1))
	bc := core.CreateBlockchain(addr1)
	assert.NotNil(t, bc)

	defer bc.DB.Close()

	//check balance
	checkBalance(t, addr1, addr2, bc, mineAward, 0)

	//create 2 transactions and start mining
	txs := []*core.Transaction{}

	miner := NewMiner(txs, bc, addr1)
	miner.Start()

	checkBalance(t, addr1, addr2, bc, mineAward*2, 0)

	teardown()
}

//mine multiple transactions
func TestMiner_MultipleValidTx(t *testing.T) {

	setup()

	//create new wallet
	wallets, _ := client.NewWallets()
	assert.NotNil(t,wallets)

	addr1 := wallets.CreateWallet()
	assert.NotNil(t, addr1)

	addr2:= wallets.CreateWallet()
	assert.NotNil(t, addr2)

	wallet := wallets.GetWallet(addr1)

	//create a blockchain
	assert.Equal(t,true,core.ValidateAddress(addr1))
	bc := core.CreateBlockchain(addr1)
	assert.NotNil(t, bc)

	defer bc.DB.Close()

	//check balance
	checkBalance(t, addr1, addr2, bc, mineAward, 0)

	//create 2 transactions and start mining
	tx, err := core.NewUTXOTransaction(addr1, addr2, sendAmount, wallet, bc)
	assert.Nil(t, err)
	//duplicated transactions. The second transaction will be ignored
	txs := []*core.Transaction{tx,tx}

	miner := NewMiner(txs, bc, addr1)
	miner.Start()

	checkBalance(t, addr1, addr2, bc, mineAward*3-sendAmount*2, sendAmount*2)

	teardown()

}

//update tx pool
func TestMiner_UpdateTxPool(t *testing.T) {

	setup()

	//create new wallet
	wallets, _ := client.NewWallets()
	assert.NotNil(t,wallets)

	addr1 := wallets.CreateWallet()
	assert.NotNil(t, addr1)

	addr2:= wallets.CreateWallet()
	assert.NotNil(t, addr2)

	wallet := wallets.GetWallet(addr1)

	//create a blockchain
	assert.Equal(t,true,core.ValidateAddress(addr1))
	bc := core.CreateBlockchain(addr1)
	assert.NotNil(t, bc)

	defer bc.DB.Close()

	//check balance
	checkBalance(t, addr1, addr2, bc, mineAward, 0)

	//create 2 transactions and start mining
	tx, err := core.NewUTXOTransaction(addr1, addr2, sendAmount, wallet, bc)
	assert.Nil(t, err)
	//duplicated transactions. The second transaction will be ignored
	txs := []*core.Transaction{tx,tx}

	miner := NewMiner(txs, bc, addr1)
	miner.Start()

	checkBalance(t, addr1, addr2, bc, mineAward*3-sendAmount*2, sendAmount*2)

	tx1, err := core.NewUTXOTransaction(addr1, addr2, sendAmount, wallet, bc)
	txs = []*core.Transaction{tx1,tx1}
	miner.UpdateTxPool(txs)
	miner.Start()

	checkBalance(t, addr1, addr2, bc, mineAward*5-sendAmount*4, sendAmount*4)

	teardown()
}

//TODO: test mining with invalid transactions
func TestMiner_InvalidTransactions(t *testing.T){

}

//balance
func getBalance(bc *core.Blockchain, addr string) (int, error){

	balance := 0
	pubKeyHash := util.Base58Decode([]byte(addr))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := bc.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}
	return balance, nil
}

func setup() {
	cleanUpDatabase()
}

func teardown() {
	cleanUpDatabase()
}

func cleanUpDatabase() {
	os.RemoveAll(storage.DefaultDbFile)
	os.RemoveAll(client.WalletFile)
}

func checkBalance(t *testing.T, addr1, addr2 string,bc *core.Blockchain,addr1v,addr2v int){
	//check balance after transaction
	balance1, err := getBalance(bc, addr1)
	assert.Nil(t, err)
	assert.Equal(t, addr1v, balance1)

	balance2, err := getBalance(bc, addr2)
	assert.Nil(t, err)
	assert.Equal(t, addr2v, balance2)
}