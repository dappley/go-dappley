package config

import (
	"io/ioutil"
	"github.com/dappley/go-dappley/config/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"
	"errors"
)

type Config struct{
	dynastyConfig DynastyConfig
}

type DynastyConfig struct{
	producers []string
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

	return &Config{dynastyConfig}
}

