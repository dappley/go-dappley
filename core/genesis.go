// Copyright (C) 2018 go-dappworks authors
//
// This file is part of the go-dappworks library.
//
// the go-dappworks library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappworks library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappworks library.  If not, see <http://www.gnu.org/licenses/>.
//
package core

const genesisCoinbaseData = "Hell world"

func NewGenesisBlock(address string) *Block {
	cbtx := NewCoinbaseTX(address, genesisCoinbaseData)
	//add coinbase txn to txn pool
	TransactionPoolSingleton.Push(cbtx)
	block := NewBlock([]byte{})
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.SetHash(hash[:])
	block.SetNonce(nonce)
	return block
}
