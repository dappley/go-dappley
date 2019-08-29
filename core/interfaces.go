// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package core

import (
	"github.com/dappley/go-dappley/common/pubsub"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/network/network_model"
	"github.com/golang/protobuf/proto"
)

type Consensus interface {
	Validate(block *Block) bool

	Setup(NetService, string, *BlockChainManager)
	GetProducerAddress() string

	SetKey(string)

	// Start runs the consensus algorithm and begins to produce blocks
	Start()

	// Stop ceases the consensus algorithm and block production
	Stop()

	// IsProducingBlock returns true if this node itself is currently producing a block
	IsProducingBlock() bool
	// Produced returns true iff the underlying block producer of the consensus algorithm produced the specified block
	Produced(block *Block) bool

	// TODO: Should separate the concept of producers from PoW
	AddProducer(string) error
	GetProducers() []string
	//Return the lib block and new block whether pass lib policy
	CheckLibPolicy(b *Block) (*Block, bool)
}

type NetService interface {
	GetHostPeerInfo() network_model.PeerInfo
	UnicastNormalPriorityCommand(commandName string, message proto.Message, destination network_model.PeerInfo)
	UnicastHighProrityCommand(commandName string, message proto.Message, destination network_model.PeerInfo)
	BroadcastNormalPriorityCommand(commandName string, message proto.Message)
	BroadcastHighProrityCommand(commandName string, message proto.Message)
	Listen(subscriber pubsub.Subscriber)
	Relay(dappCmd *network_model.DappCmd, destination network_model.PeerInfo, priority network_model.DappCmdPriority)
}

type ScEngineManager interface {
	CreateEngine() ScEngine
	RunScheduledEvents(contractUtxo []*UTXO, scStorage *ScState, blkHeight uint64, seed int64)
}

type ScEngine interface {
	DestroyEngine()
	ImportSourceCode(source string)
	ImportLocalStorage(state *ScState)
	ImportContractAddr(contractAddr account.Address)
	ImportSourceTXID(txid []byte)
	ImportUTXOs(utxos []*UTXO)
	ImportRewardStorage(rewards map[string]string)
	ImportTransaction(tx *Transaction)
	ImportContractCreateUTXO(utxo *UTXO)
	ImportPrevUtxos(utxos []*UTXO)
	ImportCurrBlockHeight(currBlkHeight uint64)
	ImportSeed(seed int64)
	ImportNodeAddress(addr account.Address)
	GetGeneratedTXs() []*Transaction
	Execute(function, args string) (string, error)
	SetExecutionLimits(uint64, uint64) error
	ExecutionInstructions() uint64
	CheckContactSyntax(source string) error
}
