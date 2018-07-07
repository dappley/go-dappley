package consensus

import (
	"os"
	"testing"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
	"github.com/dappley/go-dappley/storage"
)

var sendAmount = int(5)
var mineReward = int(10)
var tip = int64(5)

//mine one transaction
func TestMiner_SingleValidTx(t *testing.T) {

	setup()

	//create new wallet

	wallets, err := client.NewWallets()
	assert.Nil(t, err)
	assert.NotNil(t, wallets)

	addr1 := wallets.CreateWallet()
	assert.NotNil(t, addr1)

	addr2 := wallets.CreateWallet()
	assert.NotNil(t, addr2)

	wallet := wallets.GetWallet(addr1)

	//create a blockchain
	assert.Equal(t, true, core.ValidateAddress(addr1))

	db := storage.OpenDatabase(core.BlockchainDbFile)
	defer db.Close()

	bc, err := core.CreateBlockchain(addr1, NewProofOfWork(), *db)
	assert.Nil(t, err)

	assert.NotNil(t, bc)


	//check balance
	checkBalance(t, addr1, addr2, bc, mineReward, 0)

	//create 2 transactions and start mining
	tx, err := core.NewUTXOTransaction(addr1, addr2, sendAmount, wallet, bc, tip)
	assert.Nil(t, err)

	core.GetTxnPoolInstance().Push(tx)

	miner := NewMiner(bc, addr1, NewProofOfWork())
	miner.Start()

	checkBalance(t, addr1, addr2, bc, mineReward*2-sendAmount, sendAmount)

	teardown()
}

//mine empty blocks
func TestMiner_MineEmptyBlock(t *testing.T) {

	setup()

	//create new wallet
	wallets, _ := client.NewWallets()
	assert.NotNil(t, wallets)

	addr1 := wallets.CreateWallet()
	assert.NotNil(t, addr1)

	addr2 := wallets.CreateWallet()
	assert.NotNil(t, addr2)

	//create a blockchain
	assert.Equal(t, true, core.ValidateAddress(addr1))

	db := storage.OpenDatabase(core.BlockchainDbFile)
	defer db.Close()

	bc, err := core.CreateBlockchain(addr1, NewProofOfWork(),*db)
	assert.Nil(t, err)
	assert.NotNil(t, bc)


	//check balance
	checkBalance(t, addr1, addr2, bc, mineReward, 0)

	//create 2 transactions and start mining

	miner := NewMiner(bc, addr1, NewProofOfWork())
	miner.Start()

	checkBalance(t, addr1, addr2, bc, mineReward*2, 0)

	teardown()
}

//mine multiple transactions
func TestMiner_MultipleValidTx(t *testing.T) {

	setup()

	//create new wallet
	wallets, _ := client.NewWallets()
	assert.NotNil(t, wallets)

	addr1 := wallets.CreateWallet()
	assert.NotNil(t, addr1)

	addr2 := wallets.CreateWallet()
	assert.NotNil(t, addr2)

	wallet := wallets.GetWallet(addr1)

	//create a blockchain
	assert.Equal(t, true, core.ValidateAddress(addr1))

	db := storage.OpenDatabase(core.BlockchainDbFile)
	defer db.Close()
	
	bc, err := core.CreateBlockchain(addr1, NewProofOfWork(), *db)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	

	//check balance ; a:10, b:0
	checkBalance(t, addr1, addr2, bc, mineReward, 0)

	tx, err := core.NewUTXOTransaction(addr1, addr2, sendAmount, wallet, bc, tip)
	assert.Nil(t, err)

	//a:15 b:5
	core.GetTxnPoolInstance().Push(tx)
	//a:20 b:10
	core.GetTxnPoolInstance().Push(tx)

	miner := NewMiner(bc, addr1, NewProofOfWork())
	miner.Start()
	checkBalance(t, addr1, addr2, bc, mineReward*3-sendAmount*2, sendAmount*2)

	teardown()

}

//TODO: test mining with invalid transactions
func TestMiner_InvalidTransactions(t *testing.T) {

}

//balance
func getBalance(bc *core.Blockchain, addr string) (int, error) {

	balance := 0
	pubKeyHash := util.Base58Decode([]byte(addr))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs, err := bc.FindUTXO(pubKeyHash)
	if err != nil {
		return 0, err
	}

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
	os.RemoveAll("../bin/blockchain.DB")
	os.RemoveAll(client.WalletFile)
}

func checkBalance(t *testing.T, addr1, addr2 string, bc *core.Blockchain, addr1v, addr2v int) {
	//check balance after transaction
	balance1, err := getBalance(bc, addr1)
	assert.Nil(t, err)
	assert.Equal(t, addr1v, balance1)

	balance2, err := getBalance(bc, addr2)
	assert.Nil(t, err)
	assert.Equal(t, addr2v, balance2)
}
