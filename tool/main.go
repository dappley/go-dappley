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
		//enter filename
		fmt.Printf("Enter file name for blockchain%d: \n", i+1)
		text, _ := reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		db := storage.OpenDatabase(text)
		defer db.Close()
		files[i].db = db
		//enter blockchain height
		fmt.Printf("Enter max height for blockchain%d: \n", i+1)
		height, _ := reader.ReadString('\n')
		height = strings.TrimSuffix(height, "\n")
		iheight, _ := strconv.Atoi(height)
		files[i].height = iheight
		//enter height of blockchain have different with other block (0 means no different)
		fmt.Printf("Enter a different starting height for blockchain%d(0 for no different): \n", i+1)
		different, _ := reader.ReadString('\n')
		different = strings.TrimSuffix(different, "\n")
		idifferent, _ := strconv.Atoi(different)
		if iheight <= idifferent || idifferent < 1 {
			files[i].differentFrom = iheight
		} else {
			files[i].differentFrom = idifferent
		}

	}

	generateNewBlockChain(files)
}

func generateNewBlockChain(files []fileInfo) {
	bcs := make([]*core.Blockchain, len(files))
	addr := core.NewAddress(genesisAddr)
	for i := 0; i < len(files); i++ {
		bc := core.CreateBlockchain(addr, files[i].db, nil, 128)
		bcs[i] = bc
	}
	max, index := getMaxHeightOfDifferentStart(files)
	for i := 0; i < max; i++ {
		tailBlk, _ := bcs[index].GetTailBlock()
		b := core.NewBlock([]*core.Transaction{core.MockTransaction()}, tailBlk)
		b.SetHash(b.CalculateHash())
		for idx := 0; idx < len(files); idx++ {
			if files[idx].differentFrom >= i {
				bcs[idx].AddBlockToTail(b)
			}
		}
	}

	for i := 0; i < len(files); i++ {
		makeBlockChainToSize(bcs[i], files[i].height)
		fmt.Println(bcs[i].GetMaxHeight())
	}

}

func getMaxHeightOfDifferentStart(files []fileInfo) (int, int) {
	max := 0
	index := 0
	for i := 0; i < len(files); i++ {
		if max < files[i].differentFrom {
			max = files[i].differentFrom
			index = i
		}
	}
	return max, index
}

func makeBlockChainToSize(bc *core.Blockchain, size int) {
	for bc.GetMaxHeight() < uint64(size) {
		tailBlk, _ := bc.GetTailBlock()
		b := core.NewBlock([]*core.Transaction{core.MockTransaction()}, tailBlk)
		b.SetHash(b.CalculateHash())
		bc.AddBlockToTail(b)
	}
}
