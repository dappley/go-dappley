package config

import (
	"errors"

	"github.com/dappley/go-dappley/config/pb"
	"github.com/gogo/protobuf/proto"
	logger "github.com/sirupsen/logrus"
	"io/ioutil"
)

type CliConfig struct {
	port     uint32
	password string
}

func (cliConfig *CliConfig) GetRpcPort() uint32       { return cliConfig.port }
func (cliConfig *CliConfig) GetAdminPassword() string { return cliConfig.password }

func LoadCliConfigFromFile(filename string) *CliConfig {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.Warn(errors.New("Could Not Read CLI Config File"))
		logger.Warn(err)
		return nil
	}

	pb := &configpb.CliConfig{}
	err = proto.UnmarshalText(string(bytes), pb)
	if err != nil {
		logger.Warn(errors.New("Could Not Parse CLI Config File"))
		logger.Warn(err)
		return nil
	}

	return &CliConfig{
		port:     pb.Port,
		password: pb.Password,
	}
}
