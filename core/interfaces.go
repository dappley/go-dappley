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
	"github.com/libp2p/go-libp2p-peer"
)

type Consensus interface {
	Validate(block *Block) bool

	Setup(NetService, string)
	SetKey(string)

	// Start runs the consensus algorithm and begins to produce blocks
	Start()

	// Stop ceases the consensus algorithm and block production
	Stop()

	// IsProducingBlock returns true if this node itself is currently producing a block
	IsProducingBlock() bool

	// TODO: Should separate the concept of producers from PoW
	AddProducer(string) error
	GetProducers() []string
}

type NetService interface {
	BroadcastBlock(block *Block) error
	GetPeerID() peer.ID
	GetBlockchain() *Blockchain
}

type BlockPoolInterface interface {
	SetBlockchain(bc *Blockchain)
	BlockRequestCh() chan BlockRequestPars
	GetBlockchain() *Blockchain
	GetSyncState() int
	SetSyncState(int)
	VerifyTransactions(utxo UTXOIndex, forkBlks []*Block) bool
	Push(block *Block, pid peer.ID)
}
