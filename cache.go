package inmem

import (
	"bytes"
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

	"github.com/achu-1612/inmem/eviction"
	"github.com/achu-1612/inmem/log"
)

const (
	// defaultSyncInterval is the default interval for disk sync of the cache data.
	defaultSyncInterval = 5 * time.Minute
)

// make sure cache implements the Cache interface
var _ Cache = (*cache)(nil)

// Item represents a cache item.
type Item struct {
	// Object is the actual data to be stored in the cache.
	Object any

	// Expiration is the unix time when the entry will get expired/evicted from the cache.
	Expiration int64
}

// Expired returns true if the item has expired.
func (i *Item) Expired(now time.Time) bool {
	if i.Expiration == 0 {
		return false
	}

	return now.UnixNano() > i.Expiration
}

type txStore struct {
	changes map[string]Item
	deletes map[string]struct{}
}

// cache is an in-memory cache that implements the Cache interface
type cache struct {
	items map[string]Item

	mu        *sync.RWMutex
	finalizer func(string, any)

	ttl   eviction.TTL
	cond  *sync.Cond
	timer *time.Timer

	txType  TransactionType
	txStage map[uint64]*txStore

	sync           bool
	syncFolderPath string
	syncInterval   time.Duration

	storeIndex int

	l log.Logger // logger specific to cache instance.

	e eviction.Eviction // Eviction policy implemetation
}

// New returns a new Cache instance.
func NewCache(ctx context.Context, opt Options, index int) (Cache, error) {
	c := &cache{
		items:          make(map[string]Item),
		mu:             &sync.RWMutex{},
		finalizer:      opt.Finalizer,
		ttl:            eviction.NewTTL(),
		timer:          time.NewTimer(time.Hour),
		txType:         opt.TransactionType,
		txStage:        make(map[uint64]*txStore),
		sync:           opt.Sync,
		syncFolderPath: opt.SyncFolderPath,
		syncInterval:   opt.SyncInterval,
		storeIndex:     index,
		l:              log.New("store-"+fmt.Sprint(index), opt.SupressLog, opt.DebugLogs),
	}

	// we want key eviction happening at the eviction policy level to reflect in the cache store.
	// That is why the eviction finalizer is set to delete the item from the cache store.
	// The delete finalizer is set to do nothing as,
	// we internally delete on the eviction will be called when the item is deleted from the cache store.
	c.e = eviction.New(eviction.Options{
		Policy:   opt.EvictionPolicy,
		Capacity: opt.MaxSize,
		EvictFinalizer: func(s string, a any) {
			c.deleteItem(s)
		},
		DeleteFinalizer: func(s string, a any) {},
	})

	if c.finalizer == nil {
		c.l.Warn("no finalizer provided, using default finalizer")

		c.finalizer = func(k string, v any) {}
	}

	if c.txType == "" {
		c.l.Warn("no transaction type provided, using optimistic transactions")

		c.txType = TransactionTypeOptimistic
	}

	c.cond = sync.NewCond(c.mu)

	if c.sync {
		if c.syncFolderPath == "" {
			c.syncFolderPath = filepath.Join(os.TempDir(), "inmem")

			c.l.Warnf("no sync folder path provided, using default path: %s", c.syncFolderPath)
		}

		if c.syncInterval == 0 {
			c.syncInterval = defaultSyncInterval

			c.l.Warnf("no sync interval provided, using default interval: %s", c.syncInterval)
		}

		if err := c.load(); err != nil {
			c.l.Errorf("loading cache from file: %v", err)
		} else {
			c.l.Debugf("cache data loaded from folder: %s store size: %d", c.syncFolderPath, c.Size())
		}

		// start the disk sync process
		go c.diskSync(ctx)
	}

	// start the garbage collector for key eviction
	go c.garbageCollector(ctx)

	return c, nil
}

// Size returns the number of items in the cache.
func (c *cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// TransactionType returns the type of transaction used by the cache.
func (c *cache) TransactionType() TransactionType {
	return c.txType
}

// inTransaction returns true if the cache is in a transaction.
func (c *cache) inTransaction() bool {
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
	c.ttl.Clear()
	c.timer.Stop()
	c.timer.Reset(time.Hour)
	c.e.Clear()
	c.l.Debugf("cache cleared")
}

// garbageCollector is a background process that evicts expired items from the cache.
func (c *cache) garbageCollector(ctx context.Context) {
	for {
		c.mu.Lock()
		for c.ttl.Len() == 0 {
			c.cond.Wait() // release the lock and wait until an item is added
		}

		now := time.Now().UnixNano()

		for c.ttl.Len() > 0 && c.ttl.Top().Value().(int64) <= now {
			expiredKey := c.ttl.Pop()

			item, ok := c.items[expiredKey]
			if ok {
				delete(c.items, expiredKey)

				c.finalizer(expiredKey, item.Object)
			}
		}

		// Sleep until the next item's expiration
		if c.ttl.Len() > 0 {
			sleepDuration := time.Until(time.Unix(0, c.ttl.Top().Value().(int64)))

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
			if err := c.Dump(); err != nil {
				c.l.Errorf("saving cache data to file: %v", err)
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

	if c.inTransaction() {
		stage := c.txStage[c.getStageID()]
		stage.changes[k] = item

		delete(stage.deletes, k)

		c.l.Debugf("set key '%s' in transaction", k)

		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.setItem(k, item)

	c.l.Debugf("set key '%s' in store", k)
}

func (c *cache) setItem(k string, item Item) {
	c.items[k] = item

	c.e.Put(k, nil)

	if item.Expiration == 0 {
		return
	}

	top := ""
	if c.ttl.Len() > 0 {
		top = c.ttl.Top().Key() // pq top before we add the keys
	}

	c.ttl.Put(k, item.Expiration)

	c.cond.Signal()

	newTop := c.ttl.Top()

	// if the pq top has changed then reset the timer.
	if c.ttl.Len() > 0 && top != newTop.Key() {
		c.timer.Reset(time.Until(time.Unix(0, newTop.Value().(int64))))
	}
}

// Get returns a value from the cache given a key.
func (c *cache) Get(k string) (any, bool) {
	if c.inTransaction() {
		stage := c.txStage[c.getStageID()]

		if item, ok := stage.changes[k]; ok {
			c.l.Debugf("get key '%s' from transaction", k)

			return item.Object, true
		}

		if _, ok := stage.deletes[k]; ok {
			c.l.Debugf("key '%s' deleted in transaction", k)

			return nil, false
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Even if the eviction policy is not used, the nil/no-op eviction policy will return true for any key.
	if _, ok := c.e.Get(k); !ok {
		c.l.Debugf("key '%s' not found in eviction policy", k)

		return nil, false
	}

	item, ok := c.items[k]
	if !ok {
		c.l.Debugf("key '%s' not found in store", k)

		return nil, false
	}

	c.l.Debugf("get key '%s' from store", k)

	return item.Object, true
}

// Delete deletes a key from the cache.
func (c *cache) Delete(k string) {
	if c.inTransaction() {
		stage := c.txStage[c.getStageID()]

		delete(stage.changes, k)

		stage.deletes[k] = struct{}{}

		c.l.Debugf("delete key '%s' in transaction", k)

		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.deleteItem(k)

	c.l.Debugf("delete key '%s' from store", k)
}

func (c *cache) deleteItem(k string) {
	item, ok := c.items[k]
	if !ok {
		return
	}

	delete(c.items, k)
	c.finalizer(k, item.Object)

	c.e.Delete(k)

	if item.Expiration == 0 {
		return
	}

	top := ""

	if c.ttl.Len() > 0 {
		top = c.ttl.Top().Key()
	}

	c.ttl.Delete(k)

	newTop := c.ttl.Top()

	// if the top of the pq has changed, reset the timer
	if c.ttl.Len() > 0 && top != newTop.Key() {
		c.timer.Reset(time.Until(time.Unix(0, newTop.Value().(int64))))
	}
}

// Begin starts a new transaction.
func (c *cache) Begin() error {
	if c.inTransaction() {
		return ErrorInTransaction
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.txStage[c.getStageID()] = &txStore{
		changes: make(map[string]Item),
		deletes: make(map[string]struct{}),
	}

	c.l.Debugf("transaction started. stage- %d", c.getStageID())

	return nil
}

// Commit commits the current transaction.
func (c *cache) Commit() error {
	if !c.inTransaction() {
		return ErrorInTransaction
	}

	c.mu.Lock()
	defer c.mu.Unlock()

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

	c.l.Debugf("transaction committed. stage- %d", stageID)

	return nil
}

// Rollback rolls back the current transaction.
func (c *cache) Rollback() error {
	if !c.inTransaction() {
		return ErrNotInTransaction
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	stageID := c.getStageID()

	delete(c.txStage, stageID)

	c.l.Debugf("transaction rolled back. stage- %d", stageID)

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

// save writes cache's items (using Gob) to an io.Writer.
func (c *cache) save(w io.Writer) error {
	enc := gob.NewEncoder(w)

	c.mu.RLock()
	defer c.mu.RUnlock()

	return enc.Encode(&c.items)
}

// getSyncFilePath returns the path to the sync file for the current cache store/shard.
func (c *cache) getSyncFilePath() string {
	return filepath.Join(c.syncFolderPath, fmt.Sprintf("store-%d.gob", c.storeIndex))
}

// Dump the cache's items to the given filename, creating the file if it
// doesn't exist, and overwriting it if it does.
func (c *cache) Dump() error {
	fp, err := os.Create(c.getSyncFilePath())
	if err != nil {
		return fmt.Errorf("cache dump: %v", err)
	}

	defer func() {
		if err := fp.Close(); err != nil {
			c.l.Errorf("closing file %s: %v", c.getSyncFilePath(), err)
		}
	}()

	if err := c.save(fp); err != nil {
		return fmt.Errorf("cache dump: %v", err)
	}

	fi, err := fp.Stat()
	if err != nil {
		c.l.Errorf("getting file info: %v", err)
	}

	c.l.Debugf("cache data saved to file: %s store size: %d file size: %d", c.getSyncFilePath(), c.Size(), fi.Size())

	return nil
}

// load reads and loads a cache dump from the given filename.
func (c *cache) load() error {
	fileName := c.getSyncFilePath()

	fp, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return err
	}

	defer func() {
		if err := fp.Close(); err != nil {
			c.l.Errorf("closing file %s: %v", fileName, err)
		}
	}()

	dec := gob.NewDecoder(fp)

	items := map[string]Item{}

	if err := dec.Decode(&items); err != nil {
		return err
	}

	c.mu.Lock()

	now := time.Now()

	// while loading the data from a dump file,
	// if the key is expired, the finalizer won't be called for the key.
	for k, v := range items {
		if !v.Expired(now) {
			c.setItem(k, v)
		}
	}

	c.mu.Unlock()

	c.l.Debugf("cache data loaded from file: %s store size: %d", fileName, c.Size())
	return nil
}
