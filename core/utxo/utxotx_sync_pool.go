package utxo

import (
	"sync"
)

var DEFAULT_SYNC_POOL *SyncPool

func NewPool() *SyncPool {
	DEFAULT_SYNC_POOL = NewSyncPool(
		5,
		30000,
		2,
	)
	return DEFAULT_SYNC_POOL
}

func Alloc(size int) *UTXOTx {
	return DEFAULT_SYNC_POOL.Alloc(size)
}

func Free(utxotx *UTXOTx) {
	DEFAULT_SYNC_POOL.Free(utxotx)
}

// SyncPool is a sync.Pool base slab allocation memory pool
type SyncPool struct {
	classes     []sync.Pool
	classesSize []int
	minSize     int
	maxSize     int

}

func NewSyncPool(minSize, maxSize, factor int) *SyncPool {
	n := 0
	for chunkSize := minSize; chunkSize <= maxSize; chunkSize *= factor {
		n++
	}
	pool := &SyncPool{
		make([]sync.Pool, n),
		make([]int, n),
		minSize, maxSize,
	}
	n = 0
	for chunkSize := minSize; chunkSize <= maxSize; chunkSize *= factor {
		pool.classesSize[n] = chunkSize
		pool.classes[n].New = func(size int) func() interface{} {
			return func() interface{} {
				buf := UTXOTx{Indices: make(map[string]*UTXO, size)}
				return &buf
			}
		}(chunkSize)
		n++
	}
	return pool
}

func (pool *SyncPool) Alloc(size int) *UTXOTx {
	if size <= pool.maxSize {
		for i := 0; i < len(pool.classesSize); i++ {
			if pool.classesSize[i] >= size {
				utxotx := pool.classes[i].Get().(*UTXOTx)
				return utxotx
			}
		}
	}
	return &UTXOTx{Indices: make(map[string]*UTXO, size)}
}

func (pool *SyncPool) Free(utxotx *UTXOTx) {
	if size := len(utxotx.Indices); size <= pool.maxSize {
		for i := 0; i < len(pool.classesSize); i++ {
			if pool.classesSize[i] >= size {
				for k,_:= range utxotx.Indices{
					utxotx.Indices[k] = nil
					delete(utxotx.Indices,k)
				}
				pool.classes[i].Put(utxotx)
				return
			}
		}
	}
}