package eviction

import (
	"container/list"
	"testing"
)

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
			cache := &lfuCache{
				maxSize:         tt.maxSize,
				cache:           make(map[string]*list.Element),
				frequency:       make(map[int]*list.List),
				deleteFinalizer: func(key string, value any) {},
				evcitFinalizer:  func(key string, value any) {},
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
