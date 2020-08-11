package blockchain

import (
	"github.com/dappley/go-dappley/common/hash"
)

type Blockchain struct {
	tailBlockHash hash.Hash
	libHash       hash.Hash
}

func NewBlockchain(tailBlockHash hash.Hash, libBlockHash hash.Hash) Blockchain {
	setBlockchainState(BlockchainReady)
	return Blockchain{
		tailBlockHash,
		libBlockHash,
	}
}

func (bc *Blockchain) GetTailBlockHash() hash.Hash {
	return bc.tailBlockHash
}

func (bc *Blockchain) GetLIBHash() hash.Hash {
	return bc.libHash
}

func (bc *Blockchain) SetTailBlockHash(tailBlockHash hash.Hash) {
	bc.tailBlockHash = tailBlockHash
}

func (bc *Blockchain) SetLIBHash(libHash hash.Hash) {
	bc.libHash = libHash
}

func (bc *Blockchain) SetState(state BlockchainState) {
	setBlockchainState(state)
}

func (bc *Blockchain) GetState() BlockchainState {
	return getBlockchainState()
}
