package inmem

import (
	"container/heap"
	"fmt"
	"sync"
	"time"
)

// make sure cache implements the Cache interface
var _ Cache = (*cache)(nil)

// cache is an in-memory cache that implements the Cache interface
type cache struct {
	items     map[string]Item
	mu        *sync.Mutex
	finalizer func(string, interface{})
	pq        PriorityQueue
	cond      *sync.Cond
	timer     *time.Timer
}

// New returns a new Cache instance.
func New(opt Options) Cache {
	c := &cache{
		items:     make(map[string]Item),
		mu:        &sync.Mutex{},
		finalizer: opt.Finalizer,
		pq:        make(PriorityQueue, 0),
		timer:     time.NewTimer(time.Hour),
	}

	if c.finalizer == nil {
		c.finalizer = func(k string, v interface{}) {}
	}

	c.cond = sync.NewCond(c.mu)

	// start the garbage collector for key eviction
	go c.garbageCollector()

	return c
}

// Len returns the number of items in the cache.
func (c *cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.items)
}

// Clear clears all items from the cache.
func (c *cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]Item)
	c.pq = make(PriorityQueue, 0)
	c.timer.Stop()
	c.timer.Reset(time.Hour)
}

// garbageCollector is a background process that evicts expired items from the cache.
func (c *cache) garbageCollector() {
	for {
		c.mu.Lock()
		for len(c.pq) == 0 {
			c.cond.Wait() // release the lock and wait until an item is added
		}

		now := time.Now().UnixNano()
		for len(c.pq) > 0 && c.pq[0].expiresAt <= now {
			fmt.Println("Deleting key", c.pq[0].key, c.pq[0].expiresAt, now)
			expired := heap.Pop(&c.pq).(*pqItem)
			delete(c.items, expired.key)
			fmt.Println("Deleted key", expired.key)
			c.finalizer(expired.key, c.items[expired.key].Object)
		}

		// Sleep until the next item's expiration
		if len(c.pq) > 0 {
			sleepDuration := time.Until(time.Unix(0, c.pq[0].expiresAt))
			c.timer.Reset(sleepDuration)
		}
		c.mu.Unlock()

		<-c.timer.C
	}
}

// Set sets a key in the cache with a value and a time-to-live (TTL) in seconds.
func (c *cache) Set(k string, v interface{}, ttl int64) {
	item := Item{
		Object: v,
	}

	if ttl > 0 {
		item.Expiration = time.Now().UnixNano() + ttl*int64(time.Second)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[k] = item

	if ttl > 0 {
		heap.Push(&c.pq, &pqItem{
			key:       k,
			expiresAt: item.Expiration,
		})

		c.cond.Signal()

		if c.pq[0].key == k {
			c.timer.Reset(time.Until(time.Unix(0, item.Expiration)))
		}
	}
}

// Get returns a value from the cache given a key.
func (c *cache) Get(k string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[k]
	if !ok {
		return nil, false
	}

	return item.Object, true
}

// Delete deletes a key from the cache.
func (c *cache) Delete(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[k]
	if !ok {
		return
	}

	delete(c.items, k)

	c.finalizer(k, item.Object)
}
