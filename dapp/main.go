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
	"github.com/dappley/go-dappley/common/log"
	"github.com/dappley/go-dappley/logic/download_manager"
	logger "github.com/sirupsen/logrus"

	"github.com/dappley/go-dappley/config"
	configpb "github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
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
	defaultPassword = "password"
	size1kB         = 1024
)

func main() {
	viper.AddConfigPath(".")
	viper.SetConfigFile("conf/dappley.yaml")
	if err := viper.ReadInConfig(); err != nil {
		logger.Errorf("Cannot load dappley configurations from file!  error： %v", err.Error())
		return
	}

	log.BuildLogAndInit()
	logger.Debugf("Debug mode open!")
	var filePath string
	flag.StringVar(&filePath, "f", configFilePath, "Configuration File Path. Default to conf/default.conf")

	var genesisPath string
	flag.StringVar(&genesisPath, "g", genesisFilePath, "Genesis Configuration File Path. Default to conf/genesis.conf")
	flag.Parse()

	logger.Infof("Genesis conf file is %v,node conf file is %v", genesisPath, filePath)
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

	//setup
	db := storage.OpenDatabase(conf.GetNodeConfig().GetDbPath())
	defer db.Close()
	node, err := initNode(conf, db)
	defer node.Stop()

	//create blockchain
	conss, _ := initConsensus(genesisConf)
	txPoolLimit := conf.GetNodeConfig().GetTxPoolLimit() * size1kB
	nodeAddr := conf.GetNodeConfig().GetNodeAddress()
	blkSizeLimit := conf.GetNodeConfig().GetBlkSizeLimit() * size1kB
	scManager := vm.NewV8EngineManager(core.NewAddress(nodeAddr))
	txPool := core.NewTransactionPool(node, txPoolLimit)
	bc, err := core.GetBlockchain(db, conss, txPool, scManager, int(blkSizeLimit))
	if err != nil {
		bc, err = logic.CreateBlockchain(core.NewAddress(genesisAddr), db, conss, txPool, scManager, int(blkSizeLimit))
		if err != nil {
			logger.Panic(err)
		}
	}
	bc.SetState(core.BlockchainInit)

	bm := core.NewBlockChainManager(bc, core.NewBlockPool(0), node)

	if err != nil {
		logger.WithError(err).Error("Failed to initialize the node! Exiting...")
		return
	}

	downloadManager := download_manager.NewDownloadManager(node, bm)
	downloadManager.Start()
	bm.SetDownloadRequestCh(downloadManager.GetDownloadRequestCh())

	minerAddr := conf.GetConsensusConfig().GetMinerAddress()
	conss.Setup(node, minerAddr, bm)
	conss.SetKey(conf.GetConsensusConfig().GetPrivateKey())
	logger.WithFields(logger.Fields{
		"miner_address": minerAddr,
	}).Info("Consensus is configured.")

	bm.Getblockchain().SetState(core.BlockchainReady)

	//start rpc server
	nodeConf := conf.GetNodeConfig()
	server := rpc.NewGrpcServerWithMetrics(node, bm, defaultPassword, &rpc.MetricsServiceConfig{
		PollingInterval: nodeConf.GetMetricsPollingInterval(), TimeSeriesInterval: nodeConf.GetMetricsInterval()})

	server.Start(conf.GetNodeConfig().GetRpcPort())
	defer server.Stop()

	//start mining
	logic.SetLockWallet() //lock the wallet
	logic.SetMinerKeyPair(conf.GetConsensusConfig().GetPrivateKey())
	conss.Start()
	defer conss.Stop()

	bm.RequestDownloadBlockchain()

	select {}
}

func initConsensus(conf *configpb.DynastyConfig) (core.Consensus, *consensus.Dynasty) {
	//set up consensus
	conss := consensus.NewDPOS()
	dynasty := consensus.NewDynastyWithConfigProducers(conf.GetProducers(), (int)(conf.GetMaxProducers()))
	conss.SetDynasty(dynasty)
	return conss, dynasty
}

func initNode(conf *configpb.Config, db storage.Storage) (*network.Node, error) {

	nodeConfig := conf.GetNodeConfig()
	seeds := nodeConfig.GetSeed()
	port := nodeConfig.GetPort()
	keyPath := nodeConfig.GetKeyPath()

	node := network.NewNode(db, seeds)
	err := node.Start(int(port), keyPath)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	return node, nil
}
