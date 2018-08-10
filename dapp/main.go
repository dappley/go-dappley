package main

import (
	"log"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"github.com/dappley/go-dappley/client"
)

const (
	listeningPort = 12321
)

const genesisAddr = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"

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
	//set to debug level
	logger.SetLevel(logger.InfoLevel)

	//setup
	//db := storage.OpenDatabase(core.BlockchainDbFile)
	db := storage.NewRamStorage()
	defer db.Close()

	pow := consensus.NewProofOfWork()
	bc, err := logic.CreateBlockchain(core.Address{genesisAddr}, db, pow)
	if err != nil {
		log.Panic(err)
	}
	//node := network.NewNode(bc)

	//create wallet for mining
	wallets, err := client.NewWallets()
	wallet := wallets.CreateWallet()
	wallets.SaveWalletToFile()

	walletAddr := wallet.GetAddress()

	//start mining
	pow.Setup(nil, walletAddr.Address)
	pow.SetTargetBit(20)
	pow.Start()
	defer pow.Stop()

	cli.Run(bc, nil, wallets)
}
