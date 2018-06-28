package main

import (
	"fmt"
	"log"

	"github.com/dappworks/go-dappworks/core"
)

func (cli *CLI) listAddresses() {
	wallets, err := core.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}
