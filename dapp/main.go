package main

import (
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"log"
	"github.com/dappley/go-dappley/consensus"
)

func mining(db storage.LevelDB, signal chan bool)  {
	walletAddr, err := logic.CreateWallet()
	if err != nil {
		log.Panic(err)
	}
	blockchain, err := logic.CreateBlockchain(walletAddr, db)
	if err != nil {
		log.Panic(err)
	}
	miner := consensus.NewMiner(blockchain, walletAddr, consensus.NewProofOfWork(blockchain))
	miner.StartMining(signal)
}

func main() {
	cli := CLI{}
	signal :=make(chan bool)
	var db = storage.OpenDatabase(core.BlockchainDbFile)
	defer db.Close()

	go mining(*db, signal)
	cli.Run(*db)
}
