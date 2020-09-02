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
	"os"
	"testing"

	configpb "github.com/dappley/go-dappley/config/pb"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	logger.SetLevel(logger.ErrorLevel)
	tests := []struct {
		name     string
		content  string
		expected proto.Message
	}{
		{
			name:    "CorrectFileContent",
			content: fakeFileContent(),
			expected: &configpb.Config{
				ConsensusConfig: &configpb.ConsensusConfig{
					MinerAddress: "1BpXBb3uunLa9PL8MmkMtKNd3jzb5DHFkG",
					PrivateKey:   "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e",
				},
				NodeConfig: &configpb.NodeConfig{
					Port:    5,
					Seed:    []string{"/ip4/127.0.0.1/tcp/34836/ipfs/QmPtahvwSvnSHymR5HZiSTpkm9xHymx9QLNkUjJ7mfygGs"},
					DbPath:  "dbPath",
					RpcPort: 200,
				},
			},
		},
		{
			name:    "EmptySeed",
			content: noSeedContent(),
			expected: &configpb.Config{
				ConsensusConfig: &configpb.ConsensusConfig{
					MinerAddress: "1BpXBb3uunLa9PL8MmkMtKNd3jzb5DHFkG",
					PrivateKey:   "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e",
				},
				NodeConfig: &configpb.NodeConfig{
					Port: 5,
				},
			},
		},
		{
			name:     "WrongFileContent",
			content:  "WrongFileContent",
			expected: &configpb.Config{},
		},
		{
			name:     "EmptyFile",
			content:  "",
			expected: &configpb.Config{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.name + "_config.conf"
			ioutil.WriteFile(filename, []byte(tt.content), 0644)
			defer os.Remove(filename)
			config := &configpb.Config{}
			LoadConfig(filename, config)
			//fmt.Println("config.ConsensusConfig.tt.expected:",tt.expected)
			assert.Equal(t, tt.expected, config)
		})
	}
}

func fakeFileContent() string {
	return `
	consensus_config{
					miner_address: "1BpXBb3uunLa9PL8MmkMtKNd3jzb5DHFkG",
					private_key: "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e",
	}
	node_config{
		port: 5
		seed: "/ip4/127.0.0.1/tcp/34836/ipfs/QmPtahvwSvnSHymR5HZiSTpkm9xHymx9QLNkUjJ7mfygGs"
		db_path: "dbPath"
		rpc_port: 200
	}`
}

func noSeedContent() string {
	return `
	consensus_config{
						miner_address: "1BpXBb3uunLa9PL8MmkMtKNd3jzb5DHFkG",
					private_key: "bb23d2ff19f5b16955e8a24dca34dd520980fe3bddca2b3e1b56663f0ec1aa7e",
	}
	node_config{
		port: 5
	}`
}
