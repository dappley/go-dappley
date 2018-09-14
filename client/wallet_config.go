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
