package client

import (
	"io/ioutil"
	"github.com/dappley/go-dappley/client/pb"
	"github.com/gogo/protobuf/proto"

	"errors"
	logger "github.com/sirupsen/logrus"
)

type WalletConfig struct{
	filePath string
}

func (wc *WalletConfig) GetFilePath() string{return wc.filePath}

func LoadWalletConfigFromFile(filename string) *WalletConfig {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.Warn(errors.New("Could Not Read Config File"))
		logger.Warn(err)
		return nil
	}

	pb := &walletpb.WalletConfig{}
	err = proto.UnmarshalText(string(bytes), pb)

	return &WalletConfig{
		pb.FilePath,
	}
}
