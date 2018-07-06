package core

import "sync"

// An TransactionPool is a max-heap of Transactions.
type TransactionPool []Transaction

var instance *TransactionPool
var once sync.Once

func (pool TransactionPool) Len() int { return len(pool) }
//Compares Transaction Tips
func (pool TransactionPool) Less(i, j int) bool { return pool[i].Tip > pool[j].Tip }
func (pool TransactionPool) Swap(i, j int)      { pool[i], pool[j] = pool[j], pool[i] }

func (pool *TransactionPool) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*pool = append(*pool, x.(Transaction))
}

func (pool *TransactionPool) Pop() interface{} {
	old := *pool
	length := len(old)
	last := old[length-1]
	*pool = old[0 : length-1]
	return last
}

func GetTxnPoolInstance() *TransactionPool {
	once.Do(func() {
		instance = &TransactionPool{}
	})
	return instance
}

func ModifyTxnPoolInstance(pool *TransactionPool) *TransactionPool {
	instance = pool
	return instance
}

