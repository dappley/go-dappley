package main

import (
	"fmt"
	"log"

	"github.com/dappworks/go-dappworks/core"
)

func (cli *CLI) createBlockchain(address string) {
	if !core.ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := core.CreateBlockchain(address)
	bc.DB.Close()
	fmt.Println("Done!")
}
