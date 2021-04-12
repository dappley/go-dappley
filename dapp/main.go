// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package main

import (
	"flag"

	"github.com/dappley/go-dappley/core/transaction"

	"github.com/dappley/go-dappley/core/blockchain"
	"github.com/dappley/go-dappley/core/blockproducerinfo"
	"github.com/dappley/go-dappley/logic/blockproducer"
	"github.com/dappley/go-dappley/logic/lblockchain"
	"github.com/dappley/go-dappley/logic/transactionpool"

	"github.com/dappley/go-dappley/common/log"
	"github.com/dappley/go-dappley/logic/downloadmanager"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/config"
	configpb "github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"

	"github.com/dappley/go-dappley/logic"

	"net/http"
	_ "net/http/pprof"

	"github.com/dappley/go-dappley/metrics/logMetrics"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/rpc"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/vm"
	"github.com/spf13/viper"
)

const (
	genesisAddr     = "121yKAXeG4cw6uaGCBYjWk9yTWmMkhcoDD"
	configFilePath  = "conf/default.conf"
	genesisFilePath = "conf/genesis.conf"
	peerFilePath    = "conf/peer.conf"
	peerConfDirPath = "../dapp/"
	defaultPassword = "password"
	size1kB         = 1024
	version         = "v0.4.1"
)

func main() {
	defer log.CrashHandler()

	viper.AddConfigPath(".")
	viper.SetConfigFile("conf/dappley.yaml")
	if err := viper.ReadInConfig(); err != nil {
		logger.Errorf("Cannot load dappley configurations from file!  error： %v", err.Error())
		return
	}

	log.BuildLogAndInit()
	var filePath string
	flag.StringVar(&filePath, "f", configFilePath, "Configuration File Path. Default to conf/default.conf")

	var genesisPath string
	flag.StringVar(&genesisPath, "g", genesisFilePath, "Genesis Configuration File Path. Default to conf/genesis.conf")
	//flag.Parse()

	var peerinfoPath string
	flag.StringVar(&peerinfoPath, "p", peerFilePath, "Peer info configuration file Path. Default to conf/peer_default.conf")
	var ver bool
	flag.BoolVar(&ver, "v", false, "display version")
	flag.Parse()

	if ver {
		println(version)
		return
	}

	logger.Infof("Genesis conf file is %v,node conf file is %v,peer info conf file is %v", genesisPath, filePath, peerinfoPath)

	// logger.Infof("Genesis conf file is %v,node conf file is %v", genesisPath, filePath)
	//load genesis file information
	genesisConf := &configpb.DynastyConfig{}
	config.LoadConfig(genesisPath, genesisConf)

	if genesisConf == nil {
		logger.Error("Cannot load genesis configurations from file! Exiting...")
		return
	}

	//load config file information
	conf := &configpb.Config{}
	config.LoadConfig(filePath, conf)
	if conf == nil {
		logger.Error("Cannot load configurations from file! Exiting...")
		return
	}
	peerinfoPath = peerConfDirPath + peerinfoPath
	peerinfoConf := storage.NewFileLoader(peerinfoPath)

	//setup
	db := storage.OpenDatabase(conf.GetNodeConfig().GetDbPath())
	defer db.Close()
	node, err := initNode(conf, peerinfoConf)
	if err != nil {
		return
	} else {
		defer node.Stop()
	}

	//create blockchain
	conss, _ := initConsensus(genesisConf, conf)
	txPoolLimit := conf.GetNodeConfig().GetTxPoolLimit() * size1kB
	nodeAddr := conf.GetNodeConfig().GetNodeAddress()
	blkSizeLimit := conf.GetNodeConfig().GetBlkSizeLimit() * size1kB
	scManager := vm.NewV8EngineManager(account.NewAddress(nodeAddr))
	txPool := transactionpool.NewTransactionPool(node, txPoolLimit)
	//utxo.NewPool()
	bc, err := lblockchain.GetBlockchain(db, conss, txPool, scManager, int(blkSizeLimit))

	var LIBBlk *block.Block = nil
	if err != nil {
		bc, err = logic.CreateBlockchain(account.NewAddress(genesisAddr), db, conss, txPool, scManager, int(blkSizeLimit))
		if err != nil {
			logger.Panic(err)
		}
	} else {
		LIBBlk, _ = bc.GetLIB()
	}
	bc.SetState(blockchain.BlockchainInit)

	bm := lblockchain.NewBlockchainManager(bc, blockchain.NewBlockPool(LIBBlk), node, conss)

	if err != nil {
		logger.WithError(err).Error("Failed to initialize the node! Exiting...")
		return
	}

	//start mining
	logic.SetLockAccount() //lock the account
	logic.SetMinerKeyPair(conf.GetConsensusConfig().GetPrivateKey())

	//start rpc server
	nodeConf := conf.GetNodeConfig()
	server := rpc.NewGrpcServerWithMetrics(node, bm, defaultPassword, conss, &rpc.MetricsServiceConfig{
		PollingInterval: nodeConf.GetMetricsPollingInterval(), TimeSeriesInterval: nodeConf.GetMetricsInterval()})

	server.Start(conf.GetNodeConfig().GetRpcPort())
	defer server.Stop()

	producer := blockproducerinfo.NewBlockProducerInfo(conf.GetConsensusConfig().GetMinerAddress())
	blockProducer := blockproducer.NewBlockProducer(bm, conss, producer)

	downloadManager := downloadmanager.NewDownloadManager(node, bm, len(conss.GetProducers()), blockProducer)
	downloadManager.Start()

	bm.Getblockchain().SetState(blockchain.BlockchainReady)
	bm.RequestDownloadBlockchain()

	minerSubsidy := viper.GetInt("log.minerSubsidy")
	if minerSubsidy == 0 {
		minerSubsidy = 1000000000
	}
	transaction.SetSubsidy(minerSubsidy)

	if viper.GetBool("metrics.open") {
		logMetrics.LogMetricsInfo(bm.Getblockchain())
	}
	if viper.GetBool("pprof.open") {
		go func() {
			http.ListenAndServe(":60001", nil)
		}()
	}
	select {}
}

func initConsensus(conf *configpb.DynastyConfig, generalConf *configpb.Config) (*consensus.DPOS, *consensus.Dynasty) {
	//set up consensus
	conss := consensus.NewDPOS(blockproducerinfo.NewBlockProducerInfo(generalConf.GetConsensusConfig().GetMinerAddress()))
	dynasty := consensus.NewDynastyWithConfigProducers(conf.GetProducers(), (int)(conf.GetMaxProducers()))
	conss.SetDynasty(dynasty)
	conss.SetKey(generalConf.GetConsensusConfig().GetPrivateKey())
	logger.WithFields(logger.Fields{
		"miner_address": generalConf.GetConsensusConfig().GetMinerAddress(),
	}).Info("Consensus is configured.")
	return conss, dynasty
}

func initNode(conf *configpb.Config, peerinfoConf *storage.FileLoader) (*network.Node, error) {

	nodeConfig := conf.GetNodeConfig()
	seeds := nodeConfig.GetSeed()
	port := nodeConfig.GetPort()
	key := nodeConfig.GetKey()

	node := network.NewNode(peerinfoConf, seeds)
	err := node.Start(int(port), key)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	return node, nil
}
