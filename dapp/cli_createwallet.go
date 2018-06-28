package main

import (
	"fmt"

	"github.com/dappworks/go-dappworks/core"
)

func (cli *CLI) createWallet() {
	wallets, _ := core.NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()

	fmt.Printf("Your new address: %s\n", address)
}
