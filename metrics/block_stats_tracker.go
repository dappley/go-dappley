package dapmetrics

import (
	"encoding/json"
	"sync"
)

type BlockStatsTracker struct {
	NumTxPerBlock []uint64
	BlockHeights  []uint64
	capacity      int
	mutex         *sync.RWMutex
}

func NewBlockStatsTracker(capacity int) *BlockStatsTracker {
	return &BlockStatsTracker{
		NumTxPerBlock: make([]uint64, 0, capacity),
		BlockHeights:  make([]uint64, 0, capacity),
		capacity:      capacity,
		mutex:         &sync.RWMutex{},
	}
}

func (b *BlockStatsTracker) Update(numTx uint64, blockHeight uint64) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if len(b.NumTxPerBlock)+1 > b.capacity {
		b.NumTxPerBlock = b.NumTxPerBlock[1:]
		b.BlockHeights = b.BlockHeights[1:]
	}
	b.NumTxPerBlock = append(b.NumTxPerBlock, numTx)
	b.BlockHeights = append(b.BlockHeights, blockHeight)
}

func (b *BlockStatsTracker) Filter(isDPOSConsensus bool, isProducer bool) *BlockStatsTracker {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if isDPOSConsensus && !isProducer {
		b.NumTxPerBlock = make([]uint64, len(b.BlockHeights))
	}
	return b
}

type blockStatsTrackerAlias BlockStatsTracker /* prevents json.Marshall infinite recursion */

func (b *BlockStatsTracker) MarshalJSON() ([]byte, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return json.Marshal(blockStatsTrackerAlias(*b))
}
