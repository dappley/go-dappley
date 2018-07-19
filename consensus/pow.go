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

package consensus

import (
	"fmt"
	"math"
	"math/big"

	"container/heap"

	"github.com/dappley/go-dappley/core"
)

var maxNonce int64 = math.MaxInt64

const targetBits = int64(14)

type ProofOfWork struct {
	target *big.Int

	exitCh           chan bool
	messageCh        chan string
	chain            *core.Blockchain
	newBlockReceived bool
}

func NewProofOfWork(chain *core.Blockchain) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	p := &ProofOfWork{
		target:    target,
		exitCh:    make(chan bool, 1),
		messageCh: make(chan string, 128),
		chain:     chain,
	}
	return p
}

func (pow *ProofOfWork) ProduceBlock(cbAddr, cbData string, prevHash []byte) *core.Block {

	var hashInt big.Int
	var hash core.Hash
	nonce := int64(0)

	//add coinbase transaction to transaction pool

	cbtx := core.NewCoinbaseTX(cbAddr, cbData)
	h := core.GetTxnPoolInstance()

	heap.Init(h)
	heap.Push(core.GetTxnPoolInstance(), cbtx)

	parentBlockEncoded, err := pow.chain.DB.Get(prevHash)

	//todo: err handling
	if err != nil {
		return nil
	}

	parentBlock := core.Deserialize(parentBlockEncoded)

	//prepare the new block (without the correct nonce value)
	blk := core.NewBlock(core.GetTxnPoolInstance().GetSortedTransactions(), parentBlock)

	//find the nonce value
	for nonce < maxNonce {
		hash = blk.CalculateHashWithNonce(nonce)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.target) == -1 {
			break
		}

		nonce++
	}

	//complete the block
	blk.SetHash(hash)
	blk.SetNonce(nonce)

	return blk
}

func (pow *ProofOfWork) Validate(blk *core.Block) bool {
	var hashInt big.Int

	hash := blk.CalculateHash()
	hashInt.SetBytes(hash)

	isValid := hashInt.Cmp(pow.target) == -1

	if !isValid {
		return isValid
	}

	isValid = blk.VerifyHash()

	return isValid
}

func (pow *ProofOfWork) Stop() {
	pow.exitCh <- true
}

func (pow *ProofOfWork) Feed(msg string) {
	pow.messageCh <- msg
}

func (pow *ProofOfWork) Start() {
	for {
		select {
		case msg := <-pow.messageCh:
			fmt.Println(msg)
		case block := <-pow.chain.BlockPool().BlockReceivedCh():
			pow.newBlockReceived = true
			fmt.Println(block)
		case <-pow.exitCh:
			return
		}
	}
}
