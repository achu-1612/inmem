package inmem

import (
	"container/heap"
	"sync"
	"testing"
	"time"
)

func TestGarbageCollector(t *testing.T) {
	finalizerCalled := false
	finalizer := func(k string, v interface{}) {
		finalizerCalled = true
	}

	c := &cache{
		items:     make(map[string]Item),
		mu:        &sync.Mutex{},
		finalizer: finalizer,
		pq:        make(PriorityQueue, 0),
		timer:     time.NewTimer(time.Hour),
		txStage:   make(map[uint64]*txStore),
	}
	c.cond = sync.NewCond(c.mu)

	// Add an item with a short TTL
	ttl := int64(1) // 1 second
	item := Item{
		Object:     "testValue",
		Expiration: time.Now().UnixNano() + ttl*int64(time.Second),
	}
	c.items["testKey"] = item
	heap.Push(&c.pq, &pqItem{
		key:       "testKey",
		expiresAt: item.Expiration,
	})

	// Start the garbage collector in a separate goroutine
	go c.garbageCollector()

	// Wait for the item to expire
	time.Sleep(2 * time.Second)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the item has been removed
	if _, ok := c.items["testKey"]; ok {
		t.Errorf("expected item to be removed from cache")
	}

	// Check if the finalizer was called
	if !finalizerCalled {
		t.Errorf("expected finalizer to be called")
	}
}
