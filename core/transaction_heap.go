package core

// An TransactionHeap is a max-heap of Transactions.
type TransactionHeap []Transaction

func (heap TransactionHeap) Len() int { return len(heap) }
//Compares Transaction Tips
func (heap TransactionHeap) Less(i, j int) bool { return heap[i].Tip > heap[j].Tip }
func (heap TransactionHeap) Swap(i, j int)      { heap[i], heap[j] = heap[j], heap[i] }

func (heap *TransactionHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*heap = append(*heap, x.(Transaction))
}

func (heap *TransactionHeap) Pop() interface{} {
	old := *heap
	length := len(old)
	last := old[length-1]
	*heap = old[0 : length-1]
	return last
}


