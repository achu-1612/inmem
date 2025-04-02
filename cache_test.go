package inmem

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestGarbageCollector(t *testing.T) {
	finalizerCalled := false
	finalizer := func(k string, v any) {
		finalizerCalled = true
	}

	c := &cache{
		items:     make(map[string]Item),
		mu:        &sync.RWMutex{},
		finalizer: finalizer,
		pq:        make(PriorityQueue, 0),
		timer:     time.NewTimer(time.Hour),
		txStage:   make(map[uint64]*txStore),
		l:         newLogger("", true, false),
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
	go c.garbageCollector(context.Background())

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

func TestClear(t *testing.T) {
	finalizerCalled := false
	finalizer := func(k string, v any) {
		finalizerCalled = true
	}

	c := &cache{
		items:     make(map[string]Item),
		mu:        &sync.RWMutex{},
		finalizer: finalizer,
		pq:        make(PriorityQueue, 0),
		timer:     time.NewTimer(time.Hour),
		txStage:   make(map[uint64]*txStore),
		l:         newLogger("", true, false),
		e:         &NilEviction{},
	}
	c.cond = sync.NewCond(c.mu)

	// Add items to the cache
	c.Set("key1", "value1", 10)
	c.Set("key2", "value2", 20)

	// Clear the cache
	c.Clear()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the cache is empty
	if len(c.items) != 0 {
		t.Errorf("expected cache to be empty, got %d items", len(c.items))
	}

	// Check if the priority queue is empty
	if len(c.pq) != 0 {
		t.Errorf("expected priority queue to be empty, got %d items", len(c.pq))
	}

	// Check if the finalizer was called
	if finalizerCalled {
		t.Errorf("expected finalizer not to be called")
	}
}
func Test_inTransaction(t *testing.T) {
	c := &cache{
		items:     make(map[string]Item),
		mu:        &sync.RWMutex{},
		finalizer: func(k string, v any) {},
		pq:        make(PriorityQueue, 0),
		timer:     time.NewTimer(time.Hour),
		txStage:   make(map[uint64]*txStore),
		l:         newLogger("", true, false),
	}
	c.cond = sync.NewCond(c.mu)

	// Test when no transaction is in progress
	if c.inTransaction() {
		t.Errorf("expected no transaction in progress")
	}

	// Start a transaction
	err := c.Begin()
	if err != nil {
		t.Fatalf("unexpected error starting transaction: %v", err)
	}

	// Test when a transaction is in progress
	if !c.inTransaction() {
		t.Errorf("expected transaction to be in progress")
	}

	// Commit the transaction
	err = c.Commit()
	if err != nil {
		t.Fatalf("unexpected error committing transaction: %v", err)
	}

	// Test when no transaction is in progress after commit
	if c.inTransaction() {
		t.Errorf("expected no transaction in progress after commit")
	}

	// Start another transaction
	err = c.Begin()
	if err != nil {
		t.Fatalf("unexpected error starting transaction: %v", err)
	}

	// Test when a transaction is in progress
	if !c.inTransaction() {
		t.Errorf("expected transaction to be in progress")
	}

	// Rollback the transaction
	err = c.Rollback()
	if err != nil {
		t.Fatalf("unexpected error rolling back transaction: %v", err)
	}

	// Test when no transaction is in progress after rollback
	if c.inTransaction() {
		t.Errorf("expected no transaction in progress after rollback")
	}
}
func TestGetStageID(t *testing.T) {
	tests := []struct {
		name     string
		txType   TransactionType
		expected uint64
	}{
		{
			name:     "Atomic Transaction",
			txType:   TransactionTypeAtomic,
			expected: 0,
		},
		{
			name:     "Optimistic Transaction",
			txType:   TransactionTypeOptimistic,
			expected: getGoroutineID(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cache{
				txType: tt.txType,
				l:      newLogger("", true, false),
			}

			stageID := c.getStageID()
			if tt.txType == TransactionTypeOptimistic {
				if stageID == 0 {
					t.Errorf("expected non-zero stage ID for optimistic transaction, got %d", stageID)
				}
			} else {
				if stageID != tt.expected {
					t.Errorf("expected stage ID %d, got %d", tt.expected, stageID)
				}
			}
		})
	}
}
func TestSetItem(t *testing.T) {
	// finalizerCalled := false
	finalizer := func(k string, v any) {
		// finalizerCalled = true
	}

	c := &cache{
		items:     make(map[string]Item),
		mu:        &sync.RWMutex{},
		finalizer: finalizer,
		pq:        make(PriorityQueue, 0),
		timer:     time.NewTimer(time.Hour),
		txStage:   make(map[uint64]*txStore),
		l:         newLogger("", true, false),
		e:         &NilEviction{},
	}

	c.cond = sync.NewCond(c.mu)

	tests := []struct {
		name       string
		key        string
		value      any
		ttl        int64
		expectSize int
	}{
		{
			name:       "Set item without expiration",
			key:        "key1",
			value:      "value1",
			ttl:        0,
			expectSize: 1,
		},
		{
			name:       "Set item with expiration",
			key:        "key2",
			value:      "value2",
			ttl:        10,
			expectSize: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := Item{
				Object: tt.value,
			}

			if tt.ttl > 0 {
				item.Expiration = time.Now().UnixNano() + tt.ttl*int64(time.Second)
			}

			c.setItem(tt.key, item)

			c.mu.Lock()
			defer c.mu.Unlock()

			if len(c.items) != tt.expectSize {
				t.Errorf("expected cache size %d, got %d", tt.expectSize, len(c.items))
			}

			if _, ok := c.items[tt.key]; !ok {
				t.Errorf("expected item with key %s to be present", tt.key)
			}

			if tt.ttl > 0 {
				if len(c.pq) != 1 {
					t.Errorf("expected priority queue size %d, got %d", 1, len(c.pq))
				}
			} else {
				if len(c.pq) != 0 {
					t.Errorf("expected priority queue size %d, got %d", 0, len(c.pq))
				}
			}
		})
	}
}
func TestDeleteItem(t *testing.T) {
	finalizerCalled := false
	finalizer := func(k string, v any) {
		finalizerCalled = true
	}

	c := &cache{
		items:     make(map[string]Item),
		mu:        &sync.RWMutex{},
		finalizer: finalizer,
		pq:        make(PriorityQueue, 0),
		timer:     time.NewTimer(time.Hour),
		txStage:   make(map[uint64]*txStore),
		l:         newLogger("", true, false),
		e:         &NilEviction{},
	}
	c.cond = sync.NewCond(c.mu)

	tests := []struct {
		name       string
		key        string
		value      any
		ttl        int64
		expectSize int
	}{
		{
			name:       "Delete existing item without expiration",
			key:        "key1",
			value:      "value1",
			ttl:        0,
			expectSize: 0,
		},
		{
			name:       "Delete existing item with expiration",
			key:        "key2",
			value:      "value2",
			ttl:        10,
			expectSize: 0,
		},
		{
			name:       "Delete non-existing item",
			key:        "key3",
			value:      "value3",
			ttl:        0,
			expectSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := Item{
				Object: tt.value,
			}

			if tt.ttl > 0 {
				item.Expiration = time.Now().UnixNano() + tt.ttl*int64(time.Second)
			}

			c.setItem(tt.key, item)

			c.deleteItem(tt.key)

			c.mu.Lock()
			defer c.mu.Unlock()

			if len(c.items) != tt.expectSize {
				t.Errorf("expected cache size %d, got %d", tt.expectSize, len(c.items))
			}

			if _, ok := c.items[tt.key]; ok {
				t.Errorf("expected item with key %s to be deleted", tt.key)
			}

			if tt.ttl > 0 {
				if len(c.pq) != 0 {
					t.Errorf("expected priority queue size %d, got %d", 0, len(c.pq))
				}
			}

			if tt.name != "Delete non-existing item" && !finalizerCalled {
				t.Errorf("expected finalizer to be called")
			}
		})
	}
}
func TestBegin(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(c *cache)
		expectError bool
	}{
		{
			name: "Begin transaction when no transaction in progress",
			setup: func(c *cache) {
				// No setup needed
			},
			expectError: false,
		},
		{
			name: "Begin transaction when a transaction is already in progress",
			setup: func(c *cache) {
				_ = c.Begin() // Start a transaction
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cache{
				items:     make(map[string]Item),
				mu:        &sync.RWMutex{},
				finalizer: func(k string, v any) {},
				pq:        make(PriorityQueue, 0),
				timer:     time.NewTimer(time.Hour),
				txStage:   make(map[uint64]*txStore),
				l:         newLogger("", true, false),
			}
			c.cond = sync.NewCond(c.mu)

			tt.setup(c)

			err := c.Begin()
			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if !tt.expectError && !c.inTransaction() {
				t.Errorf("expected transaction to be in progress")
			}
		})
	}
}
func TestCommit(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(c *cache)
		expectError   bool
		expectedItems map[string]any
		queueSize     int
	}{
		{
			name: "Commit transaction with changes",
			setup: func(c *cache) {
				_ = c.Begin()
				c.Set("key1", "value1", 0)
				c.Set("key2", "value2", 10)
			},
			expectError: false,
			expectedItems: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			queueSize: 1,
		},
		{
			name: "Commit transaction with deletions",
			setup: func(c *cache) {
				c.Set("key1", "value1", 0)
				c.Set("key2", "value2", 10)
				_ = c.Begin()
				c.Delete("key1")
			},
			expectError: false,
			expectedItems: map[string]any{
				"key2": "value2",
			},
			queueSize: 1,
		},
		{
			name: "Commit without transaction",
			setup: func(c *cache) {
				// No setup needed
			},
			expectError:   true,
			expectedItems: map[string]any{
				// No items expected
			},
			queueSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cache{
				items:     make(map[string]Item),
				mu:        &sync.RWMutex{},
				finalizer: func(k string, v any) {},
				pq:        make(PriorityQueue, 0),
				timer:     time.NewTimer(time.Hour),
				txStage:   make(map[uint64]*txStore),
				l:         newLogger("", true, false),
				e:         &NilEviction{},
			}
			c.cond = sync.NewCond(c.mu)

			tt.setup(c)

			err := c.Commit()
			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			c.mu.Lock()
			defer c.mu.Unlock()

			for k, v := range tt.expectedItems {
				item, ok := c.items[k]
				if !ok {
					t.Errorf("expected item with key %s to be present", k)
					continue
				}
				if item.Object != v {
					t.Errorf("expected item with key %s to have value %v, got %v", k, v, item.Object)
				}
			}

			if len(c.items) != len(tt.expectedItems) {
				t.Errorf("expected cache size %d, got %d", len(tt.expectedItems), len(c.items))
			}

			if len(c.pq) != tt.queueSize {
				t.Errorf("expected priority queue size %d, got %d", tt.queueSize, len(c.pq))
			}
		})
	}
}
func TestRollback(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(c *cache)
		expectError   bool
		expectedItems map[string]any
	}{
		{
			name: "Rollback transaction with changes",
			setup: func(c *cache) {
				_ = c.Begin()
				c.Set("key1", "value1", 0)
				c.Set("key2", "value2", 10)
			},
			expectError:   false,
			expectedItems: map[string]any{
				// No items expected after rollback
			},
		},
		{
			name: "Rollback transaction with deletions",
			setup: func(c *cache) {
				c.Set("key1", "value1", 0)
				c.Set("key2", "value2", 10)
				_ = c.Begin()
				c.Delete("key1")
			},
			expectError: false,
			expectedItems: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "Rollback without transaction",
			setup: func(c *cache) {
				// No setup needed
			},
			expectError:   true,
			expectedItems: map[string]any{
				// No items expected
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cache{
				items:     make(map[string]Item),
				mu:        &sync.RWMutex{},
				finalizer: func(k string, v any) {},
				pq:        make(PriorityQueue, 0),
				timer:     time.NewTimer(time.Hour),
				txStage:   make(map[uint64]*txStore),
				l:         newLogger("", true, false),
				e:         &NilEviction{},
			}
			c.cond = sync.NewCond(c.mu)

			tt.setup(c)

			err := c.Rollback()
			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if len(c.txStage) != 0 {
				t.Errorf("expected transaction stage to be empty, got %d items", len(c.txStage))
			}

			c.mu.Lock()
			defer c.mu.Unlock()

			for k, v := range tt.expectedItems {
				item, ok := c.items[k]
				if !ok {
					t.Errorf("expected item with key %s to be present", k)
					continue
				}
				if item.Object != v {
					t.Errorf("expected item with key %s to have value %v, got %v", k, v, item.Object)
				}
			}

			if len(c.items) != len(tt.expectedItems) {
				t.Errorf("expected cache size %d, got %d", len(tt.expectedItems), len(c.items))
			}
		})
	}
}
func TestDelete_WithoutTx(t *testing.T) {
	finalizer := func(k string, v any) {}

	c := &cache{
		items:     make(map[string]Item),
		mu:        &sync.RWMutex{},
		finalizer: finalizer,
		pq:        make(PriorityQueue, 0),
		timer:     time.NewTimer(time.Hour),
		txStage:   make(map[uint64]*txStore),
		l:         newLogger("", true, false),
		e:         &NilEviction{},
	}
	c.cond = sync.NewCond(c.mu)

	tests := []struct {
		name       string
		setup      func()
		key        string
		expectSize int
	}{
		{
			name: "Delete existing item",
			setup: func() {
				c.Set("key1", "value1", 0)
			},
			key:        "key1",
			expectSize: 0,
		},
		{
			name: "Delete non-existing item",
			setup: func() {
				// No setup needed
			},
			key:        "key2",
			expectSize: 0,
		},
		{
			name: "Delete item with expiration",
			setup: func() {
				c.Set("key3", "value3", 10)
			},
			key:        "key3",
			expectSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			c.Delete(tt.key)

			c.mu.Lock()
			defer c.mu.Unlock()

			if len(c.items) != tt.expectSize {
				t.Errorf("expected cache size %d, got %d", tt.expectSize, len(c.items))
			}

			if _, ok := c.items[tt.key]; ok {
				t.Errorf("expected item with key %s to be deleted", tt.key)
			}
		})
	}
}

func TestDelete_WithTx(t *testing.T) {
	finalizer := func(k string, v any) {}

	c := &cache{
		items:     make(map[string]Item),
		mu:        &sync.RWMutex{},
		finalizer: finalizer,
		pq:        make(PriorityQueue, 0),
		timer:     time.NewTimer(time.Hour),
		txStage:   make(map[uint64]*txStore),
		l:         newLogger("", true, false),
		e:         &NilEviction{},
	}
	c.cond = sync.NewCond(c.mu)

	tests := []struct {
		name       string
		setup      func()
		key        string
		expectSize int
	}{
		{
			name: "Tx set and delete",
			setup: func() {
				_ = c.Begin()
				c.Set("key1", "value1", 0)
			},
			key:        "key1",
			expectSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			c.Delete(tt.key)

			stage := c.txStage[c.getStageID()]
			if _, ok := stage.changes[tt.key]; ok {
				t.Errorf("expected item with key %s to be deleted from transaction stage", tt.key)
			}

			if _, ok := stage.deletes[tt.key]; !ok {
				t.Errorf("expected item with key %s to be marked for deletion in transaction stage", tt.key)
			}
		})
	}
}
func TestGet(t *testing.T) {
	finalizer := func(k string, v any) {}

	c := &cache{
		items:     make(map[string]Item),
		mu:        &sync.RWMutex{},
		finalizer: finalizer,
		pq:        make(PriorityQueue, 0),
		timer:     time.NewTimer(time.Hour),
		txStage:   make(map[uint64]*txStore),
		l:         newLogger("", true, false),
		e:         &NilEviction{},
	}
	c.cond = sync.NewCond(c.mu)

	tests := []struct {
		name      string
		setup     func()
		key       string
		expectVal any
		expectOk  bool
	}{
		{
			name: "Get existing item without transaction",
			setup: func() {
				c.Set("key1", "value1", 0)
			},
			key:       "key1",
			expectVal: "value1",
			expectOk:  true,
		},
		{
			name: "Get non-existing item without transaction",
			setup: func() {
				// No setup needed
			},
			key:       "key2",
			expectVal: nil,
			expectOk:  false,
		},
		{
			name: "Get existing item with transaction",
			setup: func() {
				_ = c.Begin()
				c.Set("key3", "value3", 0)
			},
			key:       "key3",
			expectVal: "value3",
			expectOk:  true,
		},
		{
			name: "Get deleted item with transaction",
			setup: func() {
				c.Set("key4", "value4", 0)
				_ = c.Begin()
				c.Delete("key4")
			},
			key:       "key4",
			expectVal: nil,
			expectOk:  false,
		},
		{
			name: "Get item modified in transaction",
			setup: func() {
				c.Set("key5", "value5", 0)
				_ = c.Begin()
				c.Set("key5", "newValue5", 0)
			},
			key:       "key5",
			expectVal: "newValue5",
			expectOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			val, ok := c.Get(tt.key)
			if ok != tt.expectOk {
				t.Errorf("expected ok %v, got %v", tt.expectOk, ok)
			}
			if val != tt.expectVal {
				t.Errorf("expected value %v, got %v", tt.expectVal, val)
			}
		})
	}
}
func TestSet(t *testing.T) {
	finalizer := func(k string, v any) {}

	tests := []struct {
		name       string
		setup      func(c *cache)
		key        string
		value      any
		ttl        int64
		expectSize int
		expectVal  any
	}{
		{
			name: "Set item without transaction",
			setup: func(c *cache) {
				// No setup needed
			},
			key:        "key1",
			value:      "value1",
			ttl:        0,
			expectSize: 1,
			expectVal:  "value1",
		},
		{
			name: "Set item with transaction",
			setup: func(c *cache) {
				_ = c.Begin()
			},
			key:        "key2",
			value:      "value2",
			ttl:        0,
			expectSize: 0, // Transaction not committed yet
			expectVal:  nil,
		},
		{
			name: "Set item with expiration",
			setup: func(c *cache) {
				// No setup needed
			},
			key:        "key3",
			value:      "value3",
			ttl:        10,
			expectSize: 1,
			expectVal:  "value3",
		},
		{
			name: "Set item with transaction and commit",
			setup: func(c *cache) {
				_ = c.Begin()
				c.Set("key4", "value4", 0)
				_ = c.Commit()
			},
			key:        "key4",
			value:      "value4",
			ttl:        0,
			expectSize: 1,
			expectVal:  "value4",
		},
		{
			name: "Set item with transaction and rollback",
			setup: func(c *cache) {
				_ = c.Begin()
				c.Set("key5", "value5", 0)
				_ = c.Rollback()
			},
			key:        "key5",
			value:      "value5",
			ttl:        0,
			expectSize: 1, // Transaction rolled back
			expectVal:  "value5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cache{
				items:     make(map[string]Item),
				mu:        &sync.RWMutex{},
				finalizer: finalizer,
				pq:        make(PriorityQueue, 0),
				timer:     time.NewTimer(time.Hour),
				txStage:   make(map[uint64]*txStore),
				l:         newLogger("", true, false),
				e:         &NilEviction{},
			}
			c.cond = sync.NewCond(c.mu)

			tt.setup(c)

			c.Set(tt.key, tt.value, tt.ttl)

			c.mu.Lock()
			defer c.mu.Unlock()

			if len(c.items) != tt.expectSize {
				t.Errorf("expected cache size %d, got %d", tt.expectSize, len(c.items))
			}

			val, ok := c.items[tt.key]
			if tt.expectVal == nil && ok {
				t.Errorf("expected item with key %s to be absent", tt.key)
			} else if tt.expectVal != nil && (!ok || val.Object != tt.expectVal) {
				t.Errorf("expected item with key %s to have value %v, got %v", tt.key, tt.expectVal, val.Object)
			}
		})
	}
}

func TestCacheWithEviction(t *testing.T) {
	c, err := NewCache(context.Background(), Options{
		EvictionPolicy: EvictionPolicyLFU,
		MaxSize:        10,
	}, 0)

	if err != nil {
		t.Fatalf("unexpected error creating cache: %v", err)
	}

	// Add 10 items to the cache
	for i := 0; i < 10; i++ {
		c.Set(fmt.Sprintf("%d", i), i, 0)
	}

	// Add another item to the cache
	c.Set(fmt.Sprintf("%d", 10), 10, 0)

	// Check if the first item has been evicted
	if _, ok := c.Get("0"); ok {
		t.Errorf("expected item to be evicted")
	}
}
