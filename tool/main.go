package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
)

const (
	genesisAddr     = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
	configFilePath  = "conf/default.conf"
	genesisFilePath = "conf/genesis.conf"
	defaultPassword = "password"
)

type fileInfo struct {
	path          string
	maxHight      int
	differentFrom int
}

func main() {
	var filePath string

	numberBuffer := flag.Int("number", 1, "an int")

	flag.Parse()

	number := *numberBuffer

	for i := 1; i <= number; i++ {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		fmt.Println(text)
	}

	db := storage.OpenDatabase(filePath)
	defer db.Close()
	//generateNewBlockChain(maxHeight, db)
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
