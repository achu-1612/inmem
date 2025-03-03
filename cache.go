package inmem

import (
	"bytes"
	"container/heap"
	"fmt"
	"runtime"
	"strconv"
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
	txType    TransactionType
	txStage   map[uint64]*txStore
}

// New returns a new Cache instance.
func New(opt Options) Cache {
	c := &cache{
		items:     make(map[string]Item),
		mu:        &sync.Mutex{},
		finalizer: opt.Finalizer,
		pq:        make(PriorityQueue, 0),
		timer:     time.NewTimer(time.Hour),
		txType:    opt.TransactionType,
		txStage:   make(map[uint64]*txStore),
	}

	if c.finalizer == nil {
		c.finalizer = func(k string, v interface{}) {}
	}

	if c.txType == "" {
		c.txType = TransactionTypeOptimistic
	}

	c.cond = sync.NewCond(c.mu)

	// start the garbage collector for key eviction
	go c.garbageCollector()

	return c
}

// Size returns the number of items in the cache.
func (c *cache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.items)
}

// TransactionType returns the type of transaction used by the cache.
func (c *cache) TransactionType() TransactionType {
	return c.txType
}

// InTransaction returns true if the cache is in a transaction.
func (c *cache) InTransaction() bool {
	_, ok := c.txStage[c.getStageID()]
	return ok
}

// getStageID returns the stage ID for the current transaction.
func (c *cache) getStageID() uint64 {
	if c.txType == TransactionTypeAtomic {
		return 0
	}
	return getGoroutineID()
}

// Clear clears all items from the cache.
// Note: Clear does not flush out the on-going transaction data.
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
			expired := heap.Pop(&c.pq).(*pqItem)
			item, ok := c.items[expired.key]
			if ok {
				delete(c.items, expired.key)
				c.finalizer(expired.key, item.Object)
			}
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

	if c.InTransaction() {
		stage := c.txStage[c.getStageID()]
		stage.changes[k] = item
		delete(stage.deletes, k)
	} else {
		c.setItem(k, item)
	}
}

func (c *cache) setItem(k string, item Item) {
	c.items[k] = item

	if item.Expiration == 0 {
		return
	}

	top := ""
	if c.pq.Len() > 0 {
		top = c.pq[0].key // pq top before we remove the keys
	}

	heap.Push(&c.pq, &pqItem{
		key:       k,
		expiresAt: item.Expiration,
	})

	c.cond.Signal()

	if c.pq.Len() > 0 && top != c.pq[0].key {
		c.timer.Reset(time.Until(time.Unix(0, c.pq[0].expiresAt)))
	}
}

// Get returns a value from the cache given a key.
func (c *cache) Get(k string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.InTransaction() {
		stage := c.txStage[c.getStageID()]

		if item, ok := stage.changes[k]; ok {
			return item.Object, true
		}

		if _, ok := stage.deletes[k]; ok {
			return nil, false
		}
	}

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

	if c.InTransaction() {
		stage := c.txStage[c.getStageID()]

		delete(stage.changes, k)
		stage.deletes[k] = struct{}{}
	} else {
		c.deleteItem(k)
	}
}

func (c *cache) deleteItem(k string) {
	item, ok := c.items[k]
	if !ok {
		return
	}

	delete(c.items, k)
	c.finalizer(k, item.Object)

	if item.Expiration == 0 {
		return
	}

	top := ""

	if c.pq.Len() > 0 {
		top = c.pq[0].key
	}

	c.pq.Remove(k)

	// if the top of the pq has changed, reset the timer
	if c.pq.Len() > 0 && top != c.pq[0].key {
		c.timer.Reset(time.Until(time.Unix(0, c.pq[0].expiresAt)))
	}
}

// Begin starts a new transaction.
func (c *cache) Begin() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.InTransaction() {
		return fmt.Errorf("transaction already in progress")
	}

	c.txStage[c.getStageID()] = &txStore{
		changes: make(map[string]Item),
		deletes: make(map[string]struct{}),
	}

	return nil
}

// Commit commits the current transaction.
func (c *cache) Commit() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.InTransaction() {
		return fmt.Errorf("no transaction in progress")
	}

	stage := c.txStage[c.getStageID()]

	// Apply changes
	for k, item := range stage.changes {
		c.setItem(k, item)
	}

	// Apply deletions
	for k := range stage.deletes {
		c.deleteItem(k)
	}

	delete(c.txStage, c.getStageID())

	return nil
}

// Rollback rolls back the current transaction.
func (c *cache) Rollback() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.InTransaction() {
		return fmt.Errorf("no transaction in progress")
	}

	delete(c.txStage, c.getStageID())

	return nil
}

func getGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
