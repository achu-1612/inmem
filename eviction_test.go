package inmem

import (
	"testing"
)

// shardIndexEntry implements the LFUResource interface,
// so that it can be used with the LFUCache.
type testLFUEntry struct {
	key       string
	shardIdx  int
	frequency int
}

func (e *testLFUEntry) Value() any {
	return e.shardIdx
}

func (e *testLFUEntry) Set(value any) {
	e.shardIdx = value.(int)
}

func (e *testLFUEntry) Frequency() int {
	return e.frequency
}

func (e *testLFUEntry) IncrementFrequency() {
	e.frequency++
}

func (e *testLFUEntry) Key() string {
	return e.key
}

func TestLFUCache_Put(t *testing.T) {
	tests := []struct {
		name     string
		maxSize  int
		actions  []func(c Eviction)
		expected map[string]int
	}{
		{
			name:    "Put single item",
			maxSize: 1,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
			},
			expected: map[string]int{"key1": 1},
		},
		{
			name:    "Put multiple items within capacity",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key2", 2) },
			},
			expected: map[string]int{"key1": 1, "key2": 2},
		},
		{
			name:    "Put items exceeding capacity",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key2", 2) },
				func(c Eviction) { c.Put("key3", 3) },
			},
			expected: map[string]int{"key2": 2, "key3": 3},
		},
		{
			name:    "Update existing item",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key1", 10) },
			},
			expected: map[string]int{"key1": 10},
		},
		{
			name:    "Evict least frequently used item",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key2", 2) },
				func(c Eviction) { c.Get("key1") },
				func(c Eviction) { c.Put("key3", 3) },
			},
			expected: map[string]int{"key1": 1, "key3": 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewEviction(EvictionOptions{
				Policy:    EvictionPolicyLFU,
				MaxSize:   tt.maxSize,
				Allocator: func(s string) LFUResource { return &testLFUEntry{key: s} },
				Finalizer: func(s string, a any) {},
			})

			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			for _, action := range tt.actions {
				action(cache)
			}

			for key, expectedShardIdx := range tt.expected {
				if shardIdx, ok := cache.Get(key); !ok {
					t.Errorf("expected key %s to be in cache", key)
				} else if shardIdx.(int) != expectedShardIdx {
					t.Errorf("expected key %s to have shardIdx %d, got %d", key, expectedShardIdx, shardIdx)
				}

			}
		})
	}
}
