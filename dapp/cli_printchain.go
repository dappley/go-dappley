package main

import (
	"fmt"
	"strconv"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/storage"
)

func (cli *CLI) printChain(db storage.Storage) {
//	db := storage.OpenDatabase(core.BlockchainDbFile)
	bc, _ := core.GetBlockchain(db)

	bci := bc.Iterator()
	for {
		block, err := bci.Next()
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("============ Block %x ============\n", block.GetHash())
		fmt.Printf("Prev. block: %x\n", block.GetPrevHash())
		pow := consensus.NewProofOfWork(bc)
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
