package logic

import (
	"github.com/dappworks/go-dappworks/core"
	"errors"
	"github.com/dappworks/go-dappworks/client"
	"github.com/dappworks/go-dappworks/util"
)

var (
	ErrInvalidAddress = errors.New("ERROR: Address is invalid")
	ErrInvalidSenderAddress = errors.New("ERROR: Sender address is invalid")
	ErrInvalidRcverAddress = errors.New("ERROR: Receiver address is invalid")
)

//create a blockchain
func CreateBlockchain(address string) (*core.Blockchain, error){
	if !core.ValidateAddress(address) {
		return nil, ErrInvalidAddress
	}
	bc := core.CreateBlockchain(address)
	err := bc.DB.Close()
	return bc,err
}

//create a wallet
func CreateWallet() (string, error) {
	wallets, err := client.NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()
	return address, err
}

//get balance
func GetBalance(address string) (int, error) {
	if !core.ValidateAddress(address) {
		return 0,ErrInvalidAddress
	}
	bc := core.NewBlockchain(address)
	defer bc.DB.Close()

	balance := 0
	pubKeyHash := util.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := bc.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}
	return balance, nil
}

//get all addresses
func GetAllAddresses() ([]string, error) {
	wallets, err := client.NewWallets()
	if err != nil {
		return nil,err
	}
	addresses := wallets.GetAddresses()

	return addresses,err
}

func Send(from, to string, amount int) error{
	if !core.ValidateAddress(from) {
		return ErrInvalidSenderAddress
	}
	if !core.ValidateAddress(to) {
		return ErrInvalidRcverAddress
	}

	bc := core.NewBlockchain(from)
	defer bc.DB.Close()

	wallets, err := client.NewWallets()
	if err != nil {
		return err
	}
	wallet := wallets.GetWallet(from)
	tx := core.NewUTXOTransaction(from, to, amount, wallet, bc)
	cbTx := core.NewCoinbaseTX(from, "")
	txs := []*core.Transaction{cbTx, tx}

	//TODO: miner should be separated from the sender
	bc.MineBlock(txs)
	return err
}