package inmem

import (
	"context"
	"fmt"
	"testing"
)

// Just to represnt give the same key and number of shards, the shard index should be same
func TestGetShardIndex(t *testing.T) {
	tests := []struct {
		numShards uint32
		key       string
		expected  uint32
	}{
		{numShards: 5, key: "key1", expected: getShardIndex(5, "key1")},
		{numShards: 10, key: "key2", expected: getShardIndex(10, "key2")},
		{numShards: 3, key: "key3", expected: getShardIndex(3, "key3")},
		{numShards: 7, key: "key4", expected: getShardIndex(7, "key4")},
		{numShards: 1, key: "key5", expected: getShardIndex(1, "key5")},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("numShards=%d,key=%s", tt.numShards, tt.key), func(t *testing.T) {
			result := getShardIndex(tt.numShards, tt.key)
			if result != tt.expected {
				t.Errorf("getShardIndex(%d, %q) = %d; want %d",
					tt.numShards, tt.key, result, tt.expected)
			}
		})
	}
}

func TestShardedCacheSize(t *testing.T) {
	tests := []struct {
		numShards uint32
		items     map[string]any
		expected  int
	}{
		{
			numShards: 3,
			items: map[string]any{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			expected: 3,
		},
		{
			numShards: 5,
			items: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			expected: 2,
		},
		{
			numShards: 2,
			items:     map[string]any{},
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("numShards=%d", tt.numShards), func(t *testing.T) {
			ctx := context.Background()
			opt := Options{ShardCount: tt.numShards}

			cache, err := NewShardedCache(ctx, opt)
			if err != nil {
				t.Fatalf("failed to create sharded cache: %v", err)
			}

			for key, value := range tt.items {
				cache.Set(key, value, 0)
			}

			if size := cache.Size(); size != tt.expected {
				t.Errorf("Size() = %d; want %d", size, tt.expected)
			}
		})
	}
}
func TestShardedCacheGet(t *testing.T) {
	tests := []struct {
		numShards uint32
		items     map[string]any
		key       string
		expected  any
		found     bool
	}{
		{
			numShards: 3,
			items: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			key:      "key1",
			expected: "value1",
			found:    true,
		},
		{
			numShards: 5,
			items: map[string]any{
				"key3": "value3",
				"key4": "value4",
			},
			key:      "key5",
			expected: nil,
			found:    false,
		},
		{
			numShards: 2,
			items:     map[string]any{},
			key:       "key6",
			expected:  nil,
			found:     false,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("numShards=%d,key=%s", tt.numShards, tt.key), func(t *testing.T) {
			ctx := context.Background()
			opt := Options{ShardCount: tt.numShards}

			cache, err := NewShardedCache(ctx, opt)
			if err != nil {
				t.Fatalf("failed to create sharded cache: %v", err)
			}

			for key, value := range tt.items {
				cache.Set(key, value, 0)
			}

			result, found := cache.Get(tt.key)
			if result != tt.expected || found != tt.found {
				t.Errorf("Get(%q) = (%v, %v); want (%v, %v)",
					tt.key, result, found, tt.expected, tt.found)
			}
		})
	}
}

func TestShardedCacheSet(t *testing.T) {
	tests := []struct {
		numShards uint32
		items     map[string]any
		key       string
		value     any
		ttl       int64
		expected  any
		found     bool
	}{
		{
			numShards: 3,
			items: map[string]any{
				"key1": "value1",
			},
			key:      "key2",
			value:    "value2",
			ttl:      0,
			expected: "value2",
			found:    true,
		},
		{
			numShards: 5,
			items: map[string]any{
				"key3": "value3",
			},
			key:      "key3",
			value:    "newValue3",
			ttl:      0,
			expected: "newValue3",
			found:    true,
		},
		{
			numShards: 2,
			items:     map[string]any{},
			key:       "key4",
			value:     "value4",
			ttl:       0,
			expected:  "value4",
			found:     true,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("numShards=%d,key=%s", tt.numShards, tt.key), func(t *testing.T) {
			ctx := context.Background()
			opt := Options{ShardCount: tt.numShards}

			cache, err := NewShardedCache(ctx, opt)
			if err != nil {
				t.Fatalf("failed to create sharded cache: %v", err)
			}

			for key, value := range tt.items {
				cache.Set(key, value, 0)
			}

			cache.Set(tt.key, tt.value, tt.ttl)

			result, found := cache.Get(tt.key)
			if result != tt.expected || found != tt.found {
				t.Errorf("Set(%q, %v, %d); Get(%q) = (%v, %v); want (%v, %v)",
					tt.key, tt.value, tt.ttl, tt.key, result, found, tt.expected, tt.found)
			}
		})
	}
}

func TestShardedCacheDelete(t *testing.T) {
	tests := []struct {
		numShards uint32
		items     map[string]any
		key       string
		expected  map[string]any
	}{
		{
			numShards: 3,
			items: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			key: "key1",
			expected: map[string]any{
				"key2": "value2",
			},
		},
		{
			numShards: 5,
			items: map[string]any{
				"key3": "value3",
				"key4": "value4",
			},
			key: "key5", // Non-existent key
			expected: map[string]any{
				"key3": "value3",
				"key4": "value4",
			},
		},
		{
			numShards: 2,
			items: map[string]any{
				"key6": "value6",
			},
			key:      "key6",
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("numShards=%d,key=%s", tt.numShards, tt.key), func(t *testing.T) {
			ctx := context.Background()
			opt := Options{ShardCount: tt.numShards}

			cache, err := NewShardedCache(ctx, opt)
			if err != nil {
				t.Fatalf("failed to create sharded cache: %v", err)
			}

			for key, value := range tt.items {
				cache.Set(key, value, 0)
			}

			cache.Delete(tt.key)

			for key, expectedValue := range tt.expected {
				result, found := cache.Get(key)
				if !found || result != expectedValue {
					t.Errorf("After Delete(%q), Get(%q) = (%v, %v); want (%v, true)",
						tt.key, key, result, found, expectedValue)
				}
			}

			for key := range tt.items {
				if _, exists := tt.expected[key]; !exists {
					_, found := cache.Get(key)
					if found {
						t.Errorf("After Delete(%q), Get(%q) should not exist", tt.key, key)
					}
				}
			}
		})
	}
}

func TestShardedCacheClear(t *testing.T) {
	tests := []struct {
		numShards uint32
		items     map[string]any
	}{
		{
			numShards: 3,
			items: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			numShards: 5,
			items: map[string]any{
				"key3": "value3",
				"key4": "value4",
			},
		},
		{
			numShards: 2,
			items: map[string]any{
				"key5": "value5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("numShards=%d", tt.numShards), func(t *testing.T) {
			ctx := context.Background()
			opt := Options{ShardCount: tt.numShards}

			cache, err := NewShardedCache(ctx, opt)
			if err != nil {
				t.Fatalf("failed to create sharded cache: %v", err)
			}

			for key, value := range tt.items {
				cache.Set(key, value, 0)
			}

			cache.Clear()

			for key := range tt.items {
				_, found := cache.Get(key)
				if found {
					t.Errorf("After Clear(), Get(%q) should not exist", key)
				}
			}

			if size := cache.Size(); size != 0 {
				t.Errorf("After Clear(), Size() = %d; want 0", size)
			}
		})
	}
}
func TestShardedCacheTransactionType(t *testing.T) {
	tests := []struct {
		numShards       uint32
		transactionType TransactionType
	}{
		{
			numShards:       3,
			transactionType: TransactionTypeAtomic,
		},
		{
			numShards:       5,
			transactionType: TransactionTypeOptimistic,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("numShards=%d", tt.numShards), func(t *testing.T) {
			ctx := context.Background()

			opt := Options{
				ShardCount:      tt.numShards,
				TransactionType: tt.transactionType,
			}

			cache, err := NewShardedCache(ctx, opt)
			if err != nil {
				t.Fatalf("failed to create sharded cache: %v", err)
			}

			if transactionType := cache.TransactionType(); transactionType != tt.transactionType {
				t.Errorf("TransactionType() = %v; want %v", transactionType, tt.transactionType)
			}
		})
	}
}
func TestShardedCacheBegin(t *testing.T) {
	tests := []struct {
		numShards uint32
	}{
		{numShards: 3},
		{numShards: 5},
		{numShards: 7},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("numShards=%d", tt.numShards), func(t *testing.T) {
			ctx := context.Background()
			opt := Options{ShardCount: tt.numShards}

			sc, err := NewShardedCache(ctx, opt)
			if err != nil {
				t.Fatalf("failed to create sharded cache: %v", err)
			}

			if err := sc.Begin(); err != nil {
				t.Errorf("Begin() returned error: %v; want nil", err)
			}

			scImpl := sc.(*shardedCache)
			for _, shard := range scImpl.shards {
				c := shard.(*cache)
				if !c.inTransaction() {
					t.Errorf("shard %v is not in transaction", c)
				}
			}
		})
	}
}

func TestShardedCacheCommit(t *testing.T) {
	tests := []struct {
		numShards uint32
	}{
		{numShards: 3},
		{numShards: 5},
		{numShards: 7},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("numShards=%d", tt.numShards), func(t *testing.T) {
			ctx := context.Background()
			opt := Options{ShardCount: tt.numShards}

			sc, err := NewShardedCache(ctx, opt)
			if err != nil {
				t.Fatalf("failed to create sharded cache: %v", err)
			}

			if err := sc.Begin(); err != nil {
				t.Errorf("Begin() returned error: %v; want nil", err)
			}

			scImpl := sc.(*shardedCache)
			for _, shard := range scImpl.shards {
				c := shard.(*cache)
				if !c.inTransaction() {
					t.Errorf("shard %v is not in transaction", c)
				}
			}

			if err := sc.Commit(); err != nil {
				t.Errorf("Commit() returned error: %v; want nil", err)
			}

			for _, shard := range scImpl.shards {
				c := shard.(*cache)
				if c.inTransaction() {
					t.Errorf("shard %v is in transaction", c)
				}
			}
		})
	}
}

func TestShardedCacheRollback(t *testing.T) {
	tests := []struct {
		numShards uint32
	}{
		{numShards: 3},
		{numShards: 5},
		{numShards: 7},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("numShards=%d", tt.numShards), func(t *testing.T) {
			ctx := context.Background()
			opt := Options{ShardCount: tt.numShards}

			sc, err := NewShardedCache(ctx, opt)
			if err != nil {
				t.Fatalf("failed to create sharded cache: %v", err)
			}

			if err := sc.Begin(); err != nil {
				t.Errorf("Begin() returned error: %v; want nil", err)
			}

			scImpl := sc.(*shardedCache)
			for _, shard := range scImpl.shards {
				c := shard.(*cache)
				if !c.inTransaction() {
					t.Errorf("shard %v is not in transaction", c)
				}
			}

			if err := sc.Rollback(); err != nil {
				t.Errorf("Rollback() returned error: %v; want nil", err)
			}

			for _, shard := range scImpl.shards {
				c := shard.(*cache)
				if c.inTransaction() {
					t.Errorf("shard %v is in transaction", c)
				}
			}
		})
	}
}
