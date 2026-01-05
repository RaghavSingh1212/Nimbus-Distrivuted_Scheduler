package queue

import (
	"container/heap"
	"sync"
)

// PriorityQueue implements a priority queue for tasks
type PriorityQueue struct {
	items []*Item
	mu    sync.Mutex
}

type Item struct {
	TaskID   string
	Priority int
	Index    int
}

func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items: make([]*Item, 0),
	}
	heap.Init(pq)
	return pq
}

func (pq *PriorityQueue) PushTask(taskID string, priority int) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	item := &Item{
		TaskID:   taskID,
		Priority: priority,
	}
	heap.Push(pq, item)
}

func (pq *PriorityQueue) PopTask() (string, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if len(pq.items) == 0 {
		return "", false
	}
	item := heap.Pop(pq).(*Item)
	return item.TaskID, true
}

func (pq *PriorityQueue) Len() int {
	// This is called by heap package, assume lock is already held
	return len(pq.items)
}

func (pq *PriorityQueue) lenLocked() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.items)
}

func (pq *PriorityQueue) Remove(taskID string) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	for i, item := range pq.items {
		if item.TaskID == taskID {
			heap.Remove(pq, i)
			return true
		}
	}
	return false
}

// Heap interface implementation
func (pq *PriorityQueue) Less(i, j int) bool {
	// Higher priority first, then FIFO
	if pq.items[i].Priority != pq.items[j].Priority {
		return pq.items[i].Priority > pq.items[j].Priority
	}
	return i < j
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].Index = i
	pq.items[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(pq.items)
	item := x.(*Item)
	item.Index = n
	pq.items = append(pq.items, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	pq.items = old[0 : n-1]
	return item
}

