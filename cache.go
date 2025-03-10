package inmem

import (
	"bytes"
	"container/heap"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	defaultSyncInterval = 5 * time.Minute
)

// make sure cache implements the Cache interface
var _ Cache = (*cache)(nil)

// cache is an in-memory cache that implements the Cache interface
type cache struct {
	items map[string]Item

	mu        *sync.Mutex
	finalizer func(string, any)

	pq    PriorityQueue
	cond  *sync.Cond
	timer *time.Timer

	txType  TransactionType
	txStage map[uint64]*txStore

	sync         bool
	syncFilePath string
	syncInterval time.Duration
}

// New returns a new Cache instance.
func New(ctx context.Context, opt Options) Cache {
	gob.Register(Item{})

	c := &cache{
		items:        make(map[string]Item),
		mu:           &sync.Mutex{},
		finalizer:    opt.Finalizer,
		pq:           make(PriorityQueue, 0),
		timer:        time.NewTimer(time.Hour),
		txType:       opt.TransactionType,
		txStage:      make(map[uint64]*txStore),
		sync:         opt.Sync,
		syncFilePath: opt.SyncFilePath,
		syncInterval: opt.SyncInterval,
	}

	if c.finalizer == nil {
		Warn("no finalizer provided, using default finalizer")

		c.finalizer = func(k string, v any) {}
	}

	if c.txType == "" {
		Warn("no transaction type provided, using optimistic transactions")

		c.txType = TransactionTypeOptimistic
	}

	c.cond = sync.NewCond(c.mu)

	if c.sync {
		if c.syncFilePath == "" {
			c.syncFilePath = filepath.Join(os.TempDir(), "cache.gob")

			Warnf("no sync file path provided, using default path: %s", c.syncFilePath)
		}

		if c.syncInterval == 0 {
			c.syncInterval = defaultSyncInterval

			Warnf("no sync interval provided, using default interval: %s", c.syncInterval)
		}

		if err := c.load(c.syncFilePath); err != nil {
			Errorf("loading cache from file: %v", err)
		} else {
			Debugf("cache data loaded from file: %s store size: %d", c.syncFilePath, c.Size())
		}

		// start the disk sync processgo c.diskSync(ctx)
		go c.diskSync(ctx)
	}

	// start the garbage collector for key eviction
	go c.garbageCollector(ctx)

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

	Debugf("cache cleared")
}

// garbageCollector is a background process that evicts expired items from the cache.
func (c *cache) garbageCollector(ctx context.Context) {
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

		select {
		case <-c.timer.C:
		case <-ctx.Done():
			return
		}
	}
}

// diskSync is a background process that saves the cache data to disk at regular intervals.
func (c *cache) diskSync(ctx context.Context) {
	ticker := time.NewTicker(c.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.SaveFile(c.syncFilePath); err != nil {
				Errorf("saving cache data to file: %v", err)
			} else {
				Debugf("cache data saved to file: %s store size:%d", c.syncFilePath, c.Size())
			}

		case <-ctx.Done():
			return

		}
	}
}

// Set sets a key in the cache with a value and a time-to-live (TTL) in seconds.
func (c *cache) Set(k string, v any, ttl int64) {
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

		Debugf("set key '%s' in transaction", k)
	} else {
		c.setItem(k, item)

		Debugf("set key '%s' in store", k)
	}
}

func (c *cache) setItem(k string, item Item) {
	c.items[k] = item

	if item.Expiration == 0 {
		return
	}

	top := ""
	if c.pq.Len() > 0 {
		top = c.pq[0].key // pq top before we add the keys
	}

	heap.Push(&c.pq, &pqItem{
		key:       k,
		expiresAt: item.Expiration,
	})

	c.cond.Signal()

	// if the pq top has changed then reset the timer.
	if c.pq.Len() > 0 && top != c.pq[0].key {
		c.timer.Reset(time.Until(time.Unix(0, c.pq[0].expiresAt)))
	}
}

// Get returns a value from the cache given a key.
func (c *cache) Get(k string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.InTransaction() {
		stage := c.txStage[c.getStageID()]

		if item, ok := stage.changes[k]; ok {
			Debugf("get key '%s' from transaction", k)

			return item.Object, true
		}

		if _, ok := stage.deletes[k]; ok {
			Debugf("key '%s' deleted in transaction", k)

			return nil, false
		}
	}

	item, ok := c.items[k]
	if !ok {
		Debugf("key '%s' not found in store", k)

		return nil, false
	}

	Debugf("get key '%s' from store", k)

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

		Debugf("delete key '%s' in transaction", k)
	} else {
		c.deleteItem(k)

		Debugf("delete key '%s' from store", k)
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

	Debugf("transaction started. stage- %d", c.getStageID())

	return nil
}

// Commit commits the current transaction.
func (c *cache) Commit() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.InTransaction() {
		return fmt.Errorf("no transaction in progress")
	}

	stageID := c.getStageID()

	stage := c.txStage[stageID]

	// Apply changes
	for k, item := range stage.changes {
		c.setItem(k, item)
	}

	// Apply deletions
	for k := range stage.deletes {
		c.deleteItem(k)
	}

	delete(c.txStage, stageID)

	Debugf("transaction committed. stage- %d", stageID)

	return nil
}

// Rollback rolls back the current transaction.
func (c *cache) Rollback() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.InTransaction() {
		return fmt.Errorf("no transaction in progress")
	}

	stageID := c.getStageID()

	delete(c.txStage, stageID)

	Debugf("transaction rolled back. stage- %d", stageID)

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

// Write the cache's items (using Gob) to an io.Writer.
//
// NOTE: This method is deprecated in favor of c.Items() and NewFrom() (see the
// documentation for NewFrom().)
func (c *cache) Save(w io.Writer) (err error) {
	enc := gob.NewEncoder(w)

	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("error registering item types with Gob library")
		}
	}()

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, v := range c.items {
		gob.Register(v.Object)
	}

	err = enc.Encode(&c.items)

	return
}

// Save the cache's items to the given filename, creating the file if it
// doesn't exist, and overwriting it if it does.
//
// NOTE: This method is deprecated in favor of c.Items() and NewFrom() (see the
// documentation for NewFrom().)
func (c *cache) SaveFile(fname string) error {
	fp, err := os.Create(fname)
	if err != nil {
		return err
	}

	if err = c.Save(fp); err != nil {
		fp.Close()

		return err
	}

	return fp.Close()
}

func (c *cache) load(fname string) error {
	fp, err := os.Open(fname)
	if err != nil {
		return err
	}

	if err = c.Load(fp); err != nil {
		fp.Close()

		return err
	}

	return fp.Close()
}

func (c *cache) Load(r io.Reader) error {
	dec := gob.NewDecoder(r)

	items := map[string]Item{}

	if err := dec.Decode(&items); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	for k, v := range items {
		if !v.Expired(now) {
			c.setItem(k, v)
		}
	}

	return nil
}
