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

package config

import (
	"io/ioutil"
	"github.com/dappley/go-dappley/config/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"
	"errors"
)

type Config struct{
	dynastyConfig	DynastyConfig
	consensusConfig ConsensusConfig
	nodeConfig 		NodeConfig
}

type DynastyConfig struct{
	producers []string
}

type ConsensusConfig struct{
	minerAddr 	string
	privKey string
}

type NodeConfig struct{
	port 		uint32
	seed 		string
	dbPath 		string
	rpcPort 	uint32
}

type BlockchainConfig struct{
	blockchainDBFile 		string
	transactionPoolLimit 	int
}

func LoadConfigFromFile(filename string) *Config{
	bytes, err:=ioutil.ReadFile(filename)
	if err != nil {
		logrus.Warn(errors.New("Could Not Read Config File"))
		logrus.Warn(err)
		return nil
	}

	pb := &configpb.Config{}
	err = proto.UnmarshalText(string(bytes), pb)
	if err != nil {
		logrus.Warn(errors.New("Could Not Parse Config File"))
		logrus.Warn(err)
		return nil
	}

	dynastyConfig := DynastyConfig{}
	if pb.DynastyConfig != nil{
		dynastyConfig.producers = pb.DynastyConfig.Producers
	}

	consensusConfig := ConsensusConfig{}
	if pb.ConsensusConfig != nil{
		consensusConfig.minerAddr = pb.ConsensusConfig.MinerAddr
	}

	nodeConfig := NodeConfig{}
	if pb.NodeConfig != nil{
		nodeConfig.port = pb.NodeConfig.Port
		nodeConfig.seed = pb.NodeConfig.Seed
		nodeConfig.dbPath = pb.NodeConfig.DbPath
		nodeConfig.rpcPort = pb.NodeConfig.RpcPort
	}

	return &Config{
		dynastyConfig,
		consensusConfig,
		nodeConfig,
	}
}

func (config *Config) GetDynastyConfig() *DynastyConfig{return &config.dynastyConfig}
func (config *Config) GetConsensusConfig() *ConsensusConfig{return &config.consensusConfig}
func (config *Config) GetNodeConfig() *NodeConfig{return &config.nodeConfig}

func (dynastyConfig *DynastyConfig)GetProducers() []string{return dynastyConfig.producers}
func (consensusConfig *ConsensusConfig)GetMinerAddr() string{return consensusConfig.minerAddr}
func (consensusConfig *ConsensusConfig)GetMinerPrivKey() string{return consensusConfig.privKey}
func (nodeConfig *NodeConfig)GetListeningPort() uint32{return nodeConfig.port}
func (nodeConfig *NodeConfig)GetSeed() string{return nodeConfig.seed}
func (nodeConfig *NodeConfig)GetDbPath() string{return nodeConfig.dbPath}
func (nodeConfig *NodeConfig)GetRpcPort() uint32{return nodeConfig.rpcPort}