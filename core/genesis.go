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
package core

import (
	"time"
)

const genesisCoinbaseData = "Hell world"


func NewGenesisBlock(address string) *Block {
	//return consensus.ProduceBlock(address, genesisCoinbaseData,[]byte{})
	txs := []*Transaction{}
	tx := NewCoinbaseTX(address,genesisCoinbaseData)
	txs = append(txs,&tx)

	header := &BlockHeader{
		hash: []byte{000},
		prevHash: []byte{},
		nonce:     0,
		timestamp: time.Now().Unix(),
	}
	b := &Block{
		header: header,
		transactions: txs,
	}
	return b
}
