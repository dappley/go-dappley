package main

import (
	"fmt"
	"strconv"

	"github.com/dappworks/go-dappworks/core"
)

func (cli *CLI) printChain() {
	bc, _ := core.GetBlockchain("")
	defer bc.DB.Close()

	bci := bc.Iterator()

	for {
		block := bci.Next()

		fmt.Printf("============ Block %x ============\n", block.GetHash())
		fmt.Printf("Prev. block: %x\n", block.GetPrevHash())
		pow := core.NewProofOfWork(block)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.GetTransactions() {
			fmt.Println(tx)
		}
		fmt.Printf("\n\n")

		if len(block.GetPrevHash()) == 0 {
			break
		}
	}
}
