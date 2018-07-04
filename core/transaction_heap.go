package core

// An TransactionHeap is a max-heap of Transactions.
type TransactionHeap []Transaction

func (h TransactionHeap) Len() int { return len(h) }

func (h TransactionHeap) Less(i, j int) bool { return h[i].Tip > h[j].Tip }
func (h TransactionHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *TransactionHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(Transaction))
}

func (h *TransactionHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}


