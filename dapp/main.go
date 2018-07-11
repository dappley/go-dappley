package main

import (
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"log"
	"github.com/dappley/go-dappley/consensus"
	"sync"
	"github.com/dappley/go-dappley/network"
)


const(
	listeningPort = 12321
)

type Dep struct{
	db 			*storage.LevelDB
	bc			*core.Blockchain
}

func setup(db *storage.LevelDB) (string, *core.Blockchain){
	walletAddr, err := logic.CreateWallet()
	if err != nil {
		log.Panic(err)
	}
	blockchain, err := logic.CreateBlockchain(walletAddr, db)
	if err != nil {
		log.Panic(err)
	}
	return walletAddr, blockchain
}

func startNetwork(bc *core.Blockchain) *network.Node{
	//start network
	node := network.NewNode(bc)
	err := node.Start(listeningPort)
	if err!= nil {
		log.Panic(err)
	}
	return node
}

func mining(blockchain *core.Blockchain, walletAddr string, signal chan bool)  {
	miner := consensus.NewMiner(blockchain, walletAddr, consensus.NewProofOfWork(blockchain))
	miner.StartMining(signal)
}

func main() {
	cli := CLI{}
	signal :=make(chan bool)
	var waitGroup sync.WaitGroup

	//setup
	db := storage.OpenDatabase(core.BlockchainDbFile)
	defer db.Close()
	addr,bc:=setup(db)

	input := &Dep{
		db,
		bc,
	}

	waitGroup.Add(1)
	go func() {
		mining(bc,addr, signal)
		waitGroup.Done()
	}()

	cli.Run(input, signal, waitGroup)
	waitGroup.Wait()
}
