package main

import (
	"fmt"

	"github.com/dappworks/go-dappworks/client"
)

func (cli *CLI) createWallet() {
	wallets, _ := client.NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()

	fmt.Printf("Your new address: %s\n", address)
}
