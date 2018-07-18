package consensus

import (
	"testing"

	"time"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/stretchr/testify/assert"
	"fmt"
)

var sendAmount = int(7)
var sendAmount2 = int(6)
var mineReward = int(10)
var tip = int64(5)


//mine multiple transactions
func TestMiner_SingleValidTx(t *testing.T) {

	//create new wallet
	wallets, err := client.NewWallets()
	assert.Nil(t, err)
	assert.NotNil(t, wallets)

	wallet1 := wallets.CreateWallet()
	assert.NotNil(t, wallet1)

	wallet2 := wallets.CreateWallet()
	assert.NotNil(t, wallet2)

	wallet := wallets.GetKeyPairByAddress(wallet1.GetAddress())

	//create a blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	bc, err := core.CreateBlockchain(wallet1.GetAddress(), db)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//create a transaction
	tx, err := core.NewUTXOTransaction(wallet1.GetAddress(), wallet2.GetAddress(), sendAmount, wallet, bc, 0)
	assert.Nil(t, err)

	//push the transaction to transaction pool
	core.GetTxnPoolInstance().Push(tx)

	//start a miner
	miner := NewMiner(bc, wallet1.GetAddress().Address, NewProofOfWork(bc))
	signal := make(chan bool)
	go miner.Start(signal)

	//Make sure there are blocks have been mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 2 {
		time.Sleep(time.Millisecond*500)
		count = GetNumberOfBlocks(t, bc.Iterator())
	}
	miner.Stop()
	signal <- true
	//get the number of blocks
	count = GetNumberOfBlocks(t, bc.Iterator())
	//set the expected wallet value for all wallets
	var expectedVal = map[core.Address]int{
		wallet1.GetAddress()	:mineReward*count-sendAmount,  	//balance should be all mining rewards minus sendAmount
		wallet2.GetAddress()	:sendAmount,					//balance should be the amount rcved from wallet1
	}

	//check balance
	checkBalance(t,bc, expectedVal)
}

//mine empty blocks
func TestMiner_MineEmptyBlock(t *testing.T) {

	//create new wallet
	wallets, _ := client.NewWallets()
	assert.NotNil(t, wallets)

	cbWallet := wallets.CreateWallet()
	assert.NotNil(t, cbWallet)

	//Create Blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	bc, err := core.CreateBlockchain(cbWallet.GetAddress(), db)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//start a miner
	miner := NewMiner(bc, cbWallet.GetAddress().Address, NewProofOfWork(bc))
	signal := make(chan bool)
	go miner.Start(signal)

	//Make sure at least 5 blocks mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 5 {
		count = GetNumberOfBlocks(t, bc.Iterator())
		time.Sleep(time.Second)
	}
	miner.Stop()
	count = GetNumberOfBlocks(t, bc.Iterator())

	//set expected mining rewarded
	var expectedVal = map[core.Address]int{
		cbWallet.GetAddress()	: count * mineReward,
	}

	//check balance
	checkBalance(t,bc, expectedVal)

}


//mine multiple transactions
func TestMiner_MultipleValidTx(t *testing.T) {

	//create new wallet
	wallets, err := client.NewWallets()
	assert.Nil(t, err)
	assert.NotNil(t, wallets)

	wallet1 := wallets.CreateWallet()
	assert.NotNil(t, wallet1)

	wallet2 := wallets.CreateWallet()
	assert.NotNil(t, wallet2)

	wallet := wallets.GetKeyPairByAddress(wallet1.GetAddress())

	//create a blockchain
	db := storage.NewRamStorage()
	defer db.Close()

	bc, err := core.CreateBlockchain(wallet1.GetAddress(), db)
	assert.Nil(t, err)
	assert.NotNil(t, bc)

	//create a transaction
	tx, err := core.NewUTXOTransaction(wallet1.GetAddress(), wallet2.GetAddress(), sendAmount, wallet, bc, 0)
	assert.Nil(t, err)

	//push the transaction to transaction pool
	core.GetTxnPoolInstance().Push(tx)

	//start a miner
	miner := NewMiner(bc, wallet1.GetAddress().Address, NewProofOfWork(bc))
	signal := make(chan bool)
	go miner.Start(signal)

	//Make sure there are blocks have been mined
	count := GetNumberOfBlocks(t, bc.Iterator())
	for count < 2 {
		time.Sleep(time.Millisecond*500)
		count = GetNumberOfBlocks(t, bc.Iterator())
	}
	//printBalances(bc,[]core.Address{wallet1.GetAddress(),wallet2.GetAddress()})

	//add second transation
	tx2, err := core.NewUTXOTransaction(wallet1.GetAddress(), wallet2.GetAddress(), sendAmount2, wallet, bc, 0)
	assert.Nil(t, err)
	core.GetTxnPoolInstance().Push(tx2)

	//Make sure there are blocks have been mined
	currCount := GetNumberOfBlocks(t, bc.Iterator())
	//fmt.Println("currCount:",currCount)
	//printBalances(bc,[]core.Address{wallet1.GetAddress(),wallet2.GetAddress()})
	for count < currCount + 2 {
		time.Sleep(time.Millisecond*500)
		count = GetNumberOfBlocks(t, bc.Iterator())
		//printBalances(bc,[]core.Address{wallet1.GetAddress(),wallet2.GetAddress()})
	}

	//stop mining
	miner.Stop()

	//get the number of blocks
	count = GetNumberOfBlocks(t, bc.Iterator())
	//set the expected wallet value for all wallets
	var expectedVal = map[core.Address]int{
		wallet1.GetAddress()	:mineReward*count-sendAmount-sendAmount2,  	//balance should be all mining rewards minus sendAmount
		wallet2.GetAddress()	:sendAmount+sendAmount2,					//balance should be the amount rcved from wallet1
	}

	//fmt.Println(bc.String())
	//getBalancePrint(bc, wallet1.GetAddress().Address)
	//check balance
	checkBalance(t,bc, expectedVal)


}

func GetNumberOfBlocks(t *testing.T, i *core.Blockchain) int{
	//find how many blocks have been mined
	numOfBlocksMined := 1
	blk, err := i.Next()
	assert.Nil(t, err)
	for blk.GetPrevHash()!=nil {
		numOfBlocksMined++
		blk, err = i.Next()
	}
	return numOfBlocksMined
}

//TODO: test mining with invalid transactions
func TestMiner_InvalidTransactions(t *testing.T) {

}

func printBalances(bc *core.Blockchain, addrs []core.Address) {
	for _, addr := range addrs{
		b, _ := getBalance(bc, addr.Address)
		fmt.Println("addr", addr, ":", b)
	}
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

//balance
func getBalancePrint(bc *core.Blockchain, addr string) (int, error) {

	balance := 0
	pubKeyHash := util.Base58Decode([]byte(addr))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs, err := bc.FindUTXO(pubKeyHash)
	if err != nil {
		return 0, err
	}
	fmt.Println(UTXOs)

	for _, out := range UTXOs {
		balance += out.Value
	}
	return balance, nil
}

func checkBalance(t *testing.T, bc *core.Blockchain, addrBals map[core.Address]int) {
	for addr, bal := range addrBals{
		b, err := getBalance(bc, addr.Address)
		assert.Nil(t, err)
		assert.Equal(t, bal, b)
	}
}