package tool

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
)

const (
	genesisAddr           = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
	genesisFilePath       = "conf/genesis.conf"
	defaultPassword       = "password"
	defaultTimeBetweenBlk = 3
)

type FileInfo struct {
	Height        int
	DifferentFrom int
	Db            *storage.LevelDB
}

type Key struct {
	Key     string `json:"key"`
	Address string `json:"address"`
}

type Keys struct {
	Keys []Key `json:"keys"`
}

func GenerateNewBlockChain(files []FileInfo, d *consensus.Dynasty, keys Keys) {
	bcs := make([]*core.Blockchain, len(files))
	addr := core.NewAddress(genesisAddr)
	for i := 0; i < len(files); i++ {
		bc := core.CreateBlockchain(addr, files[i].Db, nil, 20)
		bcs[i] = bc
	}
	var time int64
	time = 1532392928
	max, index := GetMaxHeightOfDifferentStart(files)
	for i := 0; i < max; i++ {
		time = time + defaultTimeBetweenBlk
		b := generateBlock(bcs[index], time, d, keys)

		for idx := 0; idx < len(files); idx++ {
			if files[idx].DifferentFrom >= i {
				bcs[idx].AddBlockToTail(b)
			}
		}
	}

	for i := 0; i < len(files); i++ {
		makeBlockChainToSize(bcs[i], files[i].Height, time, d, keys)
		fmt.Println(bcs[i].GetMaxHeight())
	}

}

func GetMaxHeightOfDifferentStart(files []FileInfo) (int, int) {
	max := 0
	index := 0
	for i := 0; i < len(files); i++ {
		if max < files[i].DifferentFrom {
			max = files[i].DifferentFrom
			index = i
		}
	}
	return max, index
}

func makeBlockChainToSize(bc *core.Blockchain, size int, time int64, d *consensus.Dynasty, keys Keys) {

	for bc.GetMaxHeight() < uint64(size) {
		time = time + defaultTimeBetweenBlk
		b := generateBlock(bc, time, d, keys)
		bc.AddBlockToTail(b)
	}
}

func generateBlock(bc *core.Blockchain, time int64, d *consensus.Dynasty, keys Keys) *core.Block {
	producer := core.NewAddress(d.ProducerAtATime(time))
	key := keys.getPrivateKeyByAddress(producer)
	tailBlk, _ := bc.GetTailBlock()
	cbtx := core.NewCoinbaseTX(producer, "", bc.GetMaxHeight()+1, common.NewAmount(0))
	b := core.NewBlockWithTimestamp([]*core.Transaction{&cbtx}, tailBlk, time)
	hash := b.CalculateHashWithNonce(0)
	b.SetHash(hash)
	b.SetNonce(0)
	b.SignBlock(key, hash)
	return b
}

func LoadPrivateKey() Keys {
	jsonFile, err := os.Open("conf/key.json")
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var keys Keys

	json.Unmarshal(byteValue, &keys)

	return keys
}

func (k Keys) getPrivateKeyByAddress(address core.Address) string {
	for _, key := range k.Keys {
		if key.Address == address.Address {
			return key.Key
		}
	}
	return ""
}
