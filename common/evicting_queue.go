package common

import (
	"container/list"
	"encoding/json"
	"fmt"
	"sync"
)

type EvictingQueue struct {
	capacity int
	queue    *list.List
	mutex    *sync.RWMutex
}

type Element interface{}

// NewEvictingQueue constructor for EvictingQueue with specified maximum capacity
func NewEvictingQueue(capacity int) *EvictingQueue {
	return &EvictingQueue{
		capacity: capacity,
		queue:    list.New(),
		mutex:    &sync.RWMutex{},
	}
}

// Push adds item to the end of the queue evicting any last recently added items to ensure capacity constraint is intact
func (eq *EvictingQueue) Push(item Element) *EvictingQueue {
	eq.mutex.Lock()
	defer eq.mutex.Unlock()
	if eq.queue.Len()+1 > eq.capacity {
		eq.pop()
	}
	eq.queue.PushBack(item)
	return eq
}

// Pop returns the last recently added item, removing it from the queue
func (eq *EvictingQueue) Pop() Element {
	eq.mutex.Lock()
	defer eq.mutex.Unlock()
	return eq.pop()
}

// Peek returns the last recently added item
func (eq *EvictingQueue) Peek() Element {
	eq.mutex.RLock()
	defer eq.mutex.RUnlock()
	if !eq.isEmpty() {
		return eq.queue.Front().Value
	}
	return nil
}

func (eq *EvictingQueue) IsEmpty() bool {
	eq.mutex.RLock()
	defer eq.mutex.RUnlock()
	return eq.isEmpty()
}

func (eq *EvictingQueue) Len() int {
	eq.mutex.RLock()
	defer eq.mutex.RUnlock()
	return eq.queue.Len()
}

// ForEach applies fn to each element of the queue
// warning: any Push/Pop of this EvictingQueue within fn will result in deadlock
func (eq *EvictingQueue) ForEach(fn func(element Element)) *EvictingQueue {
	eq.mutex.RLock()
	defer eq.mutex.RUnlock()
	for front := eq.queue.Front(); front != nil; front = front.Next() {
		fn(front.Value)
	}
	return eq
}

func (eq *EvictingQueue) pop() Element {
	if !eq.isEmpty() {
		return eq.queue.Remove(eq.queue.Front())
	}
	return nil
}

func (eq *EvictingQueue) isEmpty() bool {
	return eq.queue.Len() == 0
}

func (eq *EvictingQueue) toArray() []Element {
	array := make([]Element, eq.queue.Len())
	for i, front := 0, eq.queue.Front(); front != nil; i, front = i+1, front.Next() {
		array[i] = front.Value
	}
	return array
}

func (eq *EvictingQueue) String() string {
	eq.mutex.RLock()
	defer eq.mutex.RUnlock()
	return fmt.Sprint(eq.toArray())
}

func (eq *EvictingQueue) MarshalJSON() ([]byte, error) {
	eq.mutex.RLock()
	defer eq.mutex.RUnlock()
	return json.Marshal(eq.toArray())
}
