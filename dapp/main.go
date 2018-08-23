package main

import (
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/network"
	logger "github.com/sirupsen/logrus"
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/config"
	"flag"
	"github.com/dappley/go-dappley/storage"
	"log"
	"github.com/dappley/go-dappley/logic"
)

const (
    genesisAddr = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
	configFilePath 	= "conf/default.conf"
)

func main() {

	var filePath string
	flag.StringVar(&filePath, "f", configFilePath, "Configuration File Path. Default to conf/default.conf")
	flag.Parse()

	cli := CLI{}
	//set to debug level
	logger.SetLevel(logger.InfoLevel)
	conf := config.LoadConfigFromFile(filePath)
	if conf== nil{
		logger.Error("ERROR: Cannot load configurations from file!Exiting...")
		return
	}

	//setup
	db := storage.OpenDatabase(conf.GetNodeConfig().GetDbPath())
	defer db.Close()

	//creat blockchain
	conss, dynasty := initConsensus(conf)
	bc,err := core.GetBlockchain(db,conss)
	if err !=nil {
		bc, err = logic.CreateBlockchain(core.Address{genesisAddr}, db, conss)
		if err != nil {
			log.Panic(err)
		}
	}

	node, err := initNode(conf,bc)
	if err!= nil{
		return
	}

	//create wallets
	wallets, err := client.NewWallets()

	//start mining
	minerAddr := conf.GetConsensusConfig().GetMinerAddr()
	conss.Setup(node, minerAddr)
	logger.Info("Miner Address is ", minerAddr)

	conss.Start()
	defer conss.Stop()

	cli.Run(bc, node, wallets, dynasty)
}

func initConsensus(conf *config.Config) (core.Consensus, *consensus.Dynasty){
	//set up consensus
	conss := consensus.NewDpos()
	dynasty := consensus.NewDynastyWithProducers(conf.GetDynastyConfig().GetProducers())
	conss.SetDynasty(dynasty)
	conss.SetTargetBit(18)
	return conss, dynasty
}

func initNode(conf *config.Config,bc *core.Blockchain) (*network.Node, error){
	//create node
	node := network.NewNode(bc)
	nodeConfig := conf.GetNodeConfig()
	port := nodeConfig.GetListeningPort()
	err := node.Start(int(port))
	if err!=nil {
		logger.Error(err)
		logger.Error("ERROR: Invalid Port!Exiting...")
		return nil, err
	}
	seed := nodeConfig.GetSeed()
	if seed != "" {
		node.AddStreamByString(seed)
	}
	return node,nil
}