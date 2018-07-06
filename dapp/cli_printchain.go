package main

import (
	"fmt"
	"strconv"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/consensus"
)

func (cli *CLI) printChain() {
	bc, _ := core.GetBlockchain()
	defer bc.DB.Close()

	bci := bc.Iterator()
	for {
		block, err := bci.Next()
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("============ Block %x ============\n", block.GetHash())
		fmt.Printf("Prev. block: %x\n", block.GetPrevHash())
		pow := consensus.NewProofOfWork("")
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate(block)))
		for _, tx := range block.GetTransactions() {
			fmt.Println(tx)
		}
		fmt.Printf("\n\n")

		if len(block.GetPrevHash()) == 0 {
			break
		}
	}
}
