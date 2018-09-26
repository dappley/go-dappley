package main

import (
	"flag"
	"fmt"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
)

const (
	genesisAddr     = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
	configFilePath  = "conf/default.conf"
	genesisFilePath = "conf/genesis.conf"
	defaultPassword = "password"
)

func main() {
	var filePath string
	wordPtr := flag.String("word", "foo", "a string")
	flag.StringVar(&filePath, "filePath", "default.db", "a string var")
	maxHeightBuffer := flag.Int("maxHeight", 100, "an int")
	maxHeight := *maxHeightBuffer
	fmt.Println(filePath)
	fmt.Println(maxHeight)
	fmt.Println(*wordPtr)
	db := storage.OpenDatabase(filePath)
	defer db.Close()
	generateNewBlockChain(maxHeight, db)
}

func generateNewBlockChain(size int, db storage.Storage) *core.Blockchain {
	s := db
	addr := core.NewAddress(genesisAddr)
	bc := core.CreateBlockchain(addr, s, nil)

	for i := 0; i < size; i++ {
		tailBlk, _ := bc.GetTailBlock()
		b := core.NewBlock([]*core.Transaction{core.MockTransaction()}, tailBlk)
		b.SetHash(b.CalculateHash())
		bc.AddBlockToTail(b)
	}
	return bc
}
