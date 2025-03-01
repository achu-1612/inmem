package inmem

import "container/heap"

// make sure PriorityQueue implements heap.Interface
var _ heap.Interface = (*PriorityQueue)(nil)

// pqItem is an item in the priority queue
type pqItem struct {
	key       string
	expiresAt int64
	index     int
}

// PriorityQueue is a priority queue of pqItems
type PriorityQueue []*pqItem

// Len returns the length of the priority queue
func (pq PriorityQueue) Len() int {
	return len(pq)
}

// Less returns true if the item at index i expires before the item at index j
func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].expiresAt < pq[j].expiresAt
}

// Swap swaps the items at index i and j
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index, pq[j].index = i, j
}

// Push pushes an item onto the priority queue
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)

	item := x.(*pqItem)
	item.index = n

	*pq = append(*pq, item)
}

// Pop pops an item from the priority queue
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)

	item := old[n-1]
	item.index = -1

	*pq = old[0 : n-1]

	return item
}
