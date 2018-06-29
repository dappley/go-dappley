package main

import (
	"fmt"
	"log"

	"github.com/dappworks/go-dappworks/client"
)

func (cli *CLI) listAddresses() {
	wallets, err := client.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}
