package tool

import (
	"encoding/json"
	"fmt"
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/logic"
	logger "github.com/sirupsen/logrus"
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
	defaultTimeBetweenBlk = 5
)

var (
	password             = "testpassword"
	maxWallet            = 4
	currBalance          = make(map[string]uint64)
	numOfTx				 = 100
	time 				 = int64(1532392928)
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
	for i := range files{
		bc := core.CreateBlockchain(addr, files[i].Db, nil, 2000, nil)
		bcs[i] = bc
	}

	for i, p := range d.GetProducers(){
		logger.WithFields(logger.Fields{
			"producer" : p,
		}).Info("Producer:",i)
	}

	wm, err := logic.GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		logger.Panic("Cannot get wallet manager.")
	}
	addrs := CreateWallet(wm)
	producer := core.NewAddress(d.ProducerAtATime(time))
	key := keys.getPrivateKeyByAddress(producer)
	logic.SetMinerKeyPair(key)

	//max, index := GetMaxHeightOfDifferentStart(files)
	//fund every miner
	for i := 0; i < len(d.GetProducers()); i++ {
		for idx := range files {
			b := generateBlock(bcs[idx], d, keys)
			bcs[idx].AddBlockToTail(b)
		}
	}

	//fund from miner
	fundingBlock := generateFundingBlock(bcs[0], d, keys, addrs[0])
	for idx := range files {
		bcs[idx].AddBlockToTail(fundingBlock)
	}


	for i, file := range files {
		makeBlockChainToSize(bcs[i], file.Height, d, keys, addrs, wm)
	}

}

func GetMaxHeightOfDifferentStart(files []FileInfo) (int, int) {
	max := 0
	index := 0
	for i, file := range files {
		if max < file.DifferentFrom {
			max = file.DifferentFrom
			index = i
		}
	}
	return max, index
}

func makeBlockChainToSize(bc *core.Blockchain, size int, d *consensus.Dynasty, keys Keys, addrs []core.Address, wm *client.WalletManager) {

	for bc.GetMaxHeight() < uint64(size) {
		generateTransactions(addrs, bc, wm)
		b := generateBlock(bc, d, keys)
		bc.AddBlockToTail(b)
	}
}

func generateBlock(bc *core.Blockchain, d *consensus.Dynasty, keys Keys) *core.Block {
	producer := core.NewAddress(d.ProducerAtATime(time))
	logger.WithFields(logger.Fields{
		"producer" : producer.String(),
		"timestamp": time,
	}).Info("Tool:Generating Block...")
	key := keys.getPrivateKeyByAddress(producer)
	tailBlk, _ := bc.GetTailBlock()
	cbtx := core.NewCoinbaseTX(producer, "", bc.GetMaxHeight()+1, common.NewAmount(0))
	utxoIndex := core.LoadUTXOIndex(bc.GetDb())
	txs := bc.GetTxPool().GetFilteredTransactions(utxoIndex, bc.GetMaxHeight()+1)
	txs = append(txs, &cbtx)
	b := core.NewBlockWithTimestamp(txs, tailBlk, time)
	hash := b.CalculateHashWithNonce(0)
	b.SetHash(hash)
	b.SetNonce(0)
	b.SignBlock(key, hash)
	time = time + defaultTimeBetweenBlk
	return b
}

func generateFundingBlock(bc *core.Blockchain, d *consensus.Dynasty, keys Keys, fundAddr core.Address) *core.Block{
	logger.Info("generateFundingBlock")
	initFund := uint64(100000)
	logic.SendFromMiner(fundAddr, common.NewAmount(initFund), bc, nil)
	currBalance[fundAddr.String()] = initFund
	return generateBlock(bc, d, keys)
}

func generateTransactions(addrs []core.Address, bc *core.Blockchain,wm *client.WalletManager){
	for i:=0;i< numOfTx;i++{
		generateTransaction(addrs, bc, wm)
	}
}

func generateTransaction(addrs []core.Address, bc *core.Blockchain,wm *client.WalletManager){
	sender, receiver := getSenderAndReceiver(addrs)

	senderWallet := wm.GetWalletByAddress(sender)
	if senderWallet == nil || len(senderWallet.Addresses) == 0 {
		logger.Panic("Can not find sender wallet")
	}

	logic.Send(senderWallet, receiver, common.NewAmount(1), common.NewAmount(0), "", bc, nil)
	currBalance[sender.String()] -= 1
	currBalance[receiver.String()] += 1
}

func getSenderAndReceiver(addrs []core.Address) (sender,receiver core.Address){
	for i, addr := range addrs{
		if currBalance[addr.String()] > 1000 {
			sender = addr
			if i == len(addrs)-1 {
				receiver = addrs[0]
			}else{
				receiver = addrs[i+1]
			}
			return
		}
	}
	logger.Panic("getSenderAndReceiver failed")
	return
}

func CreateRandomTransactions([]core.Address) []*core.Transaction{
	return nil
}

func CreateWallet(wm *client.WalletManager) []core.Address {

	addresses := wm.GetAddresses()
	numOfWallets := len(addresses)
	for i := numOfWallets; i < maxWallet; i++ {
		_, err := logic.CreateWalletWithpassphrase(password)
		if err != nil {
			logger.WithError(err).Panic("Cannot create new wallet.")
		}
	}

	addresses = wm.GetAddresses()
	logger.WithFields(logger.Fields{
		"addresses": addresses,
	}).Info("Wallets are created")
	return addresses
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
