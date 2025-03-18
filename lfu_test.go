package inmem

import (
	"testing"
)

func TestLFUCache_Put(t *testing.T) {
	tests := []struct {
		name     string
		maxSize  int
		actions  []func(c *LFUCache)
		expected map[string]uint32
	}{
		{
			name:    "Put single item",
			maxSize: 1,
			actions: []func(c *LFUCache){
				func(c *LFUCache) { c.Put("key1", 1) },
			},
			expected: map[string]uint32{"key1": 1},
		},
		{
			name:    "Put multiple items within capacity",
			maxSize: 2,
			actions: []func(c *LFUCache){
				func(c *LFUCache) { c.Put("key1", 1) },
				func(c *LFUCache) { c.Put("key2", 2) },
			},
			expected: map[string]uint32{"key1": 1, "key2": 2},
		},
		{
			name:    "Put items exceeding capacity",
			maxSize: 2,
			actions: []func(c *LFUCache){
				func(c *LFUCache) { c.Put("key1", 1) },
				func(c *LFUCache) { c.Put("key2", 2) },
				func(c *LFUCache) { c.Put("key3", 3) },
			},
			expected: map[string]uint32{"key2": 2, "key3": 3},
		},
		{
			name:    "Update existing item",
			maxSize: 2,
			actions: []func(c *LFUCache){
				func(c *LFUCache) { c.Put("key1", 1) },
				func(c *LFUCache) { c.Put("key1", 10) },
			},
			expected: map[string]uint32{"key1": 10},
		},
		{
			name:    "Evict least frequently used item",
			maxSize: 2,
			actions: []func(c *LFUCache){
				func(c *LFUCache) { c.Put("key1", 1) },
				func(c *LFUCache) { c.Put("key2", 2) },
				func(c *LFUCache) { c.Get("key1") },
				func(c *LFUCache) { c.Put("key3", 3) },
			},
			expected: map[string]uint32{"key1": 1, "key3": 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewLFUCache(tt.maxSize)
			
			for _, action := range tt.actions {
				action(cache)
			}

			for key, expectedShardIdx := range tt.expected {
				if shardIdx, ok := cache.Get(key); !ok || shardIdx != expectedShardIdx {
					t.Errorf("expected key %s to have shardIdx %d, got %d", key, expectedShardIdx, shardIdx)
				}
			}
		})
	}
}
