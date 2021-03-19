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
	"strconv"

	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
)

func LoadConfig(filename string, pb proto.Message) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.WithError(err).Warn("LoadConfig: cannot read the config file!")
		return
	}

	err = proto.UnmarshalText(string(bytes), pb)
	if err != nil {
		logger.WithError(err).Warn("LoadConfig: cannot parse content of the config file!")
	}
}

func UpdateProducer(filename string, producers []string, height uint64) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.WithError(err).Warn("LoadConfig: cannot read the config file!")
		return
	}
	err = ioutil.WriteFile(filename+strconv.FormatUint(height, 10), bytes, 0644)
	if err != nil {
		logger.WithError(err).Warn("LoadConfig: cannot backup the config file!")
		return
	}

	info := "producers: [\n"
	for i := 0; i < len(producers)-1; i++ {
		info = info + "\"" + producers[i] + "\",\n"
	}

	info = info + "\"" + producers[len(producers)-1] + "\"\n]\n"
	info = info + "max_producers: " + strconv.Itoa(len(producers))
	err = ioutil.WriteFile(filename, []byte(info), 0644)

}
