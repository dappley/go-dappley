package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

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
	height        int
	differentFrom int
	db            *storage.LevelDB
}

func main() {
	// var filePath string

	numberBuffer := flag.Int("number", 1, "an int")

	flag.Parse()

	number := *numberBuffer
	files := make([]fileInfo, number)

	for i := 0; i < number; i++ {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Enter file name for %d: \n", i+1)
		text, _ := reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		db := storage.OpenDatabase(text)
		defer db.Close()
		files[i].db = db
		fmt.Printf("Enter max height for %d: \n", i+1)
		height, _ := reader.ReadString('\n')
		height = strings.TrimSuffix(height, "\n")
		iheight, _ := strconv.Atoi(height)
		files[i].height = iheight
		fmt.Printf("Enter a different starting height for %d: \n", i+1)
		different, _ := reader.ReadString('\n')
		different = strings.TrimSuffix(different, "\n")
		idifferent, _ := strconv.Atoi(different)
		files[i].differentFrom = idifferent
	}

	generateNewBlockChain(files)
}

func generateNewBlockChain(files []fileInfo) {
	bcs := make([]*core.Blockchain, len(files))
	addr := core.NewAddress(genesisAddr)
	for i := 0; i < len(files); i++ {
		bc := core.CreateBlockchain(addr, files[i].db, nil)
		bcs[i] = bc
	}

	// for i := 0; i < files[0].height; i++ {
	// 	tailBlk, _ := bcs[0].GetTailBlock()
	// 	b := core.NewBlock([]*core.Transaction{core.MockTransaction()}, tailBlk)
	// 	b.SetHash(b.CalculateHash())
	// 	bc.AddBlockToTail(b)
	// }
}

func getMaxHeight(files []fileInfo) int {
	max := 0
	for i := 0; i < len(files); i++ {
		if max < files[i].height {
			max = files[i].height
		}
	}
	return max
}
