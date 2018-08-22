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
}

type NodeConfig struct{
	port 		uint32
	seed 		string
	dbPath 		string
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
func (nodeConfig *NodeConfig)GetListeningPort() uint32{return nodeConfig.port}
func (nodeConfig *NodeConfig)GetSeed() string{return nodeConfig.seed}
func (nodeConfig *NodeConfig)GetDbPath() string{return nodeConfig.dbPath}