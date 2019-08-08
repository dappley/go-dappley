package blockchain

import (
	"github.com/dappley/go-dappley/common/hash"
)

type BlockchainState int

const (
	BlockchainInit BlockchainState = iota
	BlockchainDownloading
	BlockchainSync
	BlockchainReady
)

type Blockchain struct {
	tailBlockHash hash.Hash
	libHash       hash.Hash
	state         BlockchainState
}

func NewBlockchain(tailBlockHash hash.Hash, libBlockHash hash.Hash) *Blockchain {
	return &Blockchain{
		tailBlockHash,
		libBlockHash,
		BlockchainReady,
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
	bc.state = state
}

func (bc *Blockchain) SetStateToBlockchainReady() {
	bc.state = BlockchainReady
}

func (bc *Blockchain) SetStateToBlockchainDownloading() {
	bc.state = BlockchainDownloading
}

func (bc *Blockchain) SetStateToBlockchainSyncing() {
	bc.state = BlockchainSync
}

func (bc *Blockchain) SetStateToBlockchainInit() {
	bc.state = BlockchainInit
}

func (bc *Blockchain) GetState() BlockchainState {
	return bc.state
}

func (bc *Blockchain) IsBlockchainInitializing() bool {
	return bc.state == BlockchainInit
}

func (bc *Blockchain) IsBlockchainDownloading() bool {
	return bc.state == BlockchainDownloading
}

func (bc *Blockchain) IsBlockchainSyncing() bool {
	return bc.state == BlockchainSync
}

func (bc *Blockchain) IsBlockchainReady() bool {
	return bc.state == BlockchainReady
}
