package main

import (
	"log"
	"sync"

	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
)

const (
	listeningPort = 12321
)

type Dep struct {
	db *storage.LevelDB
	bc *core.Blockchain
}

func setup(db *storage.LevelDB) (string, *core.Blockchain) {
	wallet, err := logic.CreateWallet()
	if err != nil {
		log.Panic(err)
	}
	walletAddr := wallet.GetAddress()
	blockchain, err := logic.CreateBlockchain(walletAddr, db)
	if err != nil {
		log.Panic(err)
	}
	return walletAddr.Address, blockchain
}

func startNetwork(bc *core.Blockchain) *network.Node {
	//start network
	node := network.NewNode(bc)
	err := node.Start(listeningPort)
	if err != nil {
		log.Panic(err)
	}
	return node
}

func main() {
	cli := CLI{}
	var waitGroup sync.WaitGroup
	//set to debug level
	logger.SetLevel(logger.InfoLevel)

	//setup
	db := storage.OpenDatabase(core.BlockchainDbFile)
	defer db.Close()
	addr, bc := setup(db)

	input := &Dep{
		db,
		bc,
	}

	waitGroup.Add(1)
	pow := consensus.NewProofOfWork()
	pow.Setup(bc, addr)
	miner := consensus.NewMiner(pow)
	go func() {
		miner.Start()
		waitGroup.Done()
	}()

	cli.Run(input, miner, waitGroup)
	waitGroup.Wait()
}
