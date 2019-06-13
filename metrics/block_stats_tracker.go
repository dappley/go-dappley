package dapmetrics

import (
	"encoding/json"
	"sync"

	"github.com/dappley/go-dappley/common"
)

type BlockStats struct {
	NumTransactions uint64
	Height          uint64
}

type BlockStatsTracker struct {
	Stats    *common.EvictingQueue
	capacity int
	mutex    *sync.RWMutex
}

func NewBlockStatsTracker(capacity int) *BlockStatsTracker {
	return &BlockStatsTracker{
		Stats:    common.NewEvictingQueue(capacity),
		capacity: capacity,
		mutex:    &sync.RWMutex{},
	}
}

func (b *BlockStatsTracker) Update(numTx uint64, blockHeight uint64) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.Stats.Push(&BlockStats{NumTransactions: numTx, Height: blockHeight})
}

func (b *BlockStatsTracker) Filter(isDPOSConsensus bool, isProducer bool) *BlockStatsTracker {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if isDPOSConsensus && !isProducer {
		b.Stats.ForEach(func(element common.Element) {
			element.(*BlockStats).NumTransactions = 0
		})
	}
	return b
}

func (b *BlockStatsTracker) MarshalJSON() ([]byte, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return json.Marshal(b.Stats)
}
