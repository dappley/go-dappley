package main

import (
	"fmt"
	"log"

	"github.com/dappworks/go-dappworks/core"
	"github.com/dappworks/go-dappworks/client"
)

func (cli *CLI) send(from, to string, amount int) {
	if !core.ValidateAddress(from) {
		log.Panic("ERROR: Sender address is not valid")
	}
	if !core.ValidateAddress(to) {
		log.Panic("ERROR: Recipient address is not valid")
	}

	bc := core.NewBlockchain(from)
	defer bc.DB.Close()


	wallets, err := client.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)
	tx := core.NewUTXOTransaction(from, to, amount, wallet, bc)
	cbTx := core.NewCoinbaseTX(from, "")
	txs := []*core.Transaction{cbTx, tx}

	//TODO: miner should be separated from the sender
	bc.MineBlock(txs)
	fmt.Println("Success!")
}
