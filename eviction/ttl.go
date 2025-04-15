package eviction

import (
	"container/heap"
)

type TTLItem interface {
	Key() string
	Value() any
}

// pqItem is an item in the priority queue
type pqItem struct {
	key       string
	expiresAt int64
	index     int
}

func (p *pqItem) Key() string {
	return p.key
}

func (p *pqItem) Value() any {
	return p.expiresAt
}

// PriorityQueue is a priority queue of pqItems
type PriorityQueue []*pqItem

// make sure PriorityQueue implements heap.Interface
var _ heap.Interface = (*PriorityQueue)(nil)

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
func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)

	item := x.(*pqItem)
	item.index = n

	*pq = append(*pq, item)
}

// Pop pops an item from the priority queue
func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)

	item := old[n-1]
	item.index = -1

	*pq = old[0 : n-1]

	return item
}

// Remove removes items from the priority queue given the keys
func (pq *PriorityQueue) Remove(key string) {
	for i, item := range *pq {
		if item.key == key {
			heap.Remove(pq, i)
			break
		}
	}
}

type TTL interface {
	Len() int
	Top() TTLItem
	Put(key string, value any)
	Delete(key string)
	Pop() string
	Clear()
}

var _ TTL = (*ttl)(nil)

type ttl struct {
	pq PriorityQueue
}

func NewTTL() TTL {
	return &ttl{
		pq: make(PriorityQueue, 0),
	}
}

func (t *ttl) Len() int {
	return t.pq.Len()
}

func (t *ttl) Pop() string {
	v := heap.Pop(&t.pq).(*pqItem)
	if v == nil {
		return ""
	}

	return v.key
}

func (t *ttl) Delete(key string) {
	t.pq.Remove(key)
}

func (t *ttl) Clear() {
	t.pq = make(PriorityQueue, 0)
}

func (t *ttl) Top() TTLItem {
	if t.pq.Len() == 0 {
		return nil
	}

	return (t.pq)[0]
}

func (t *ttl) Put(key string, value any) {
	heap.Push(&t.pq, &pqItem{
		key:       key,
		expiresAt: value.(int64),
	})
}
