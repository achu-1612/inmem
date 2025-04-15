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

func TestLFUCache_Get(t *testing.T) {
	tests := []struct {
		name     string
		maxSize  int
		actions  []func(c Eviction)
		getKey   string
		expected any
		found    bool
	}{
		{
			name:    "Get existing item",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key2", 2) },
			},
			getKey:   "key1",
			expected: 1,
			found:    true,
		},
		{
			name:    "Get non-existing item",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
			},
			getKey:   "key2",
			expected: nil,
			found:    false,
		},
		{
			name:    "Get item after eviction",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key2", 2) },
				func(c Eviction) { c.Put("key3", 3) },
			},
			getKey:   "key1",
			expected: nil,
			found:    false,
		},
		{
			name:    "Get item updates frequency",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key2", 2) },
				func(c Eviction) { c.Get("key1") },
				func(c Eviction) { c.Put("key3", 3) },
			},
			getKey:   "key1",
			expected: 1,
			found:    true,
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

			value, ok := cache.Get(tt.getKey)
			if ok != tt.found {
				t.Errorf("expected found to be %v, got %v", tt.found, ok)
			}
			if value != tt.expected {
				t.Errorf("expected value to be %v, got %v", tt.expected, value)
			}
		})
	}
}

func TestLFUCache_Delete(t *testing.T) {
	tests := []struct {
		name      string
		maxSize   int
		actions   []func(c Eviction)
		deleteKey string
		expected  map[string]int
	}{
		{
			name:    "Delete existing item",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key2", 2) },
			},
			deleteKey: "key1",
			expected:  map[string]int{"key2": 2},
		},
		{
			name:    "Delete non-existing item",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
			},
			deleteKey: "key2",
			expected:  map[string]int{"key1": 1},
		},
		{
			name:    "Delete item after eviction",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key2", 2) },
				func(c Eviction) { c.Put("key3", 3) },
			},
			deleteKey: "key1",
			expected:  map[string]int{"key2": 2, "key3": 3},
		},
		{
			name:    "Delete all items",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key2", 2) },
				func(c Eviction) { c.Delete("key1") },
				func(c Eviction) { c.Delete("key2") },
			},
			deleteKey: "key2",
			expected:  map[string]int{},
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

			cache.Delete(tt.deleteKey)

			for key, expectedValue := range tt.expected {
				if value, ok := cache.Get(key); !ok {
					t.Errorf("expected key %s to be in cache", key)
				} else if value.(int) != expectedValue {
					t.Errorf("expected key %s to have value %d, got %d", key, expectedValue, value)
				}
			}

			for key := range cache.cache {
				if _, exists := tt.expected[key]; !exists {
					t.Errorf("key %s should not be in cache", key)
				}
			}
		})
	}
}

func TestLFUCache_Clear(t *testing.T) {
	tests := []struct {
		name     string
		maxSize  int
		actions  []func(c Eviction)
		expected map[string]int
	}{
		{
			name:    "Clear empty cache",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Clear() },
			},
			expected: map[string]int{},
		},
		{
			name:    "Clear cache with items",
			maxSize: 3,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key2", 2) },
				func(c Eviction) { c.Put("key3", 3) },
				func(c Eviction) { c.Clear() },
			},
			expected: map[string]int{},
		},
		{
			name:    "Clear cache after eviction",
			maxSize: 2,
			actions: []func(c Eviction){
				func(c Eviction) { c.Put("key1", 1) },
				func(c Eviction) { c.Put("key2", 2) },
				func(c Eviction) { c.Put("key3", 3) },
				func(c Eviction) { c.Clear() },
			},
			expected: map[string]int{},
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

			if len(cache.cache) != 0 {
				t.Errorf("expected cache to be empty, but it has %d items", len(cache.cache))
			}

			if len(cache.frequency) != 0 {
				t.Errorf("expected frequency map to be empty, but it has %d items", len(cache.frequency))
			}

			if cache.size != 0 {
				t.Errorf("expected cache size to be 0, but got %d", cache.size)
			}

			if cache.minFreq != 0 {
				t.Errorf("expected minFreq to be 0, but got %d", cache.minFreq)
			}
		})
	}
}
