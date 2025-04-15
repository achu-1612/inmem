package eviction

import "testing"

func TestLRUCache_Get(t *testing.T) {
	deleteFinalizer := func(key string, value any) {}
	evictFinalizer := func(key string, value any) {}

	t.Run("Get existing key", func(t *testing.T) {
		cache := newLRU(2, deleteFinalizer, evictFinalizer)
		cache.Put("key1", "value1")
		cache.Put("key2", "value2")

		value, found := cache.Get("key1")
		if !found {
			t.Errorf("expected key1 to be found")
		}
		if value != "value1" {
			t.Errorf("expected value1, got %v", value)
		}
	})

	t.Run("Get non-existing key", func(t *testing.T) {
		cache := newLRU(2, deleteFinalizer, evictFinalizer)
		cache.Put("key1", "value1")

		_, found := cache.Get("key2")
		if found {
			t.Errorf("expected key2 to not be found")
		}
	})

	t.Run("Get updates LRU order", func(t *testing.T) {
		cache := newLRU(2, deleteFinalizer, evictFinalizer)
		cache.Put("key1", "value1")
		cache.Put("key2", "value2")

		// Access key1 to make it most recently used
		cache.Get("key1")

		// Add another key to trigger eviction
		cache.Put("key3", "value3")

		_, found := cache.Get("key2")
		if found {
			t.Errorf("expected key2 to be evicted")
		}

		_, found = cache.Get("key1")
		if !found {
			t.Errorf("expected key1 to still be in the cache")
		}
	})
}
func TestLRUCache_Put(t *testing.T) {
	deleteFinalizer := func(key string, value any) {}
	evictFinalizer := func(key string, value any) {}

	t.Run("Put new key", func(t *testing.T) {
		cache := newLRU(2, deleteFinalizer, evictFinalizer)
		cache.Put("key1", "value1")

		value, found := cache.Get("key1")
		if !found {
			t.Errorf("expected key1 to be found")
		}
		if value != "value1" {
			t.Errorf("expected value1, got %v", value)
		}
	})

	t.Run("Put existing key updates value", func(t *testing.T) {
		cache := newLRU(2, deleteFinalizer, evictFinalizer)
		cache.Put("key1", "value1")
		cache.Put("key1", "value2")

		value, found := cache.Get("key1")
		if !found {
			t.Errorf("expected key1 to be found")
		}
		if value != "value2" {
			t.Errorf("expected value2, got %v", value)
		}
	})

	t.Run("Put evicts least recently used key", func(t *testing.T) {
		cache := newLRU(2, deleteFinalizer, evictFinalizer)
		cache.Put("key1", "value1")
		cache.Put("key2", "value2")

		// Add another key to trigger eviction
		cache.Put("key3", "value3")

		_, found := cache.Get("key1")
		if found {
			t.Errorf("expected key1 to be evicted")
		}

		value, found := cache.Get("key2")
		if !found {
			t.Errorf("expected key2 to still be in the cache")
		}
		if value != "value2" {
			t.Errorf("expected value2, got %v", value)
		}

		value, found = cache.Get("key3")
		if !found {
			t.Errorf("expected key3 to still be in the cache")
		}
		if value != "value3" {
			t.Errorf("expected value3, got %v", value)
		}
	})
}

func TestLRUCache_Delete(t *testing.T) {
	deleteFinalizer := func(key string, value any) {}
	evictFinalizer := func(key string, value any) {}

	t.Run("Delete existing key", func(t *testing.T) {
		cache := newLRU(2, deleteFinalizer, evictFinalizer)
		cache.Put("key1", "value1")
		cache.Put("key2", "value2")

		cache.Delete("key1")

		_, found := cache.Get("key1")
		if found {
			t.Errorf("expected key1 to be deleted")
		}

		value, found := cache.Get("key2")
		if !found {
			t.Errorf("expected key2 to still be in the cache")
		}
		if value != "value2" {
			t.Errorf("expected value2, got %v", value)
		}
	})

	t.Run("Delete non-existing key", func(t *testing.T) {
		cache := newLRU(2, deleteFinalizer, evictFinalizer)
		cache.Put("key1", "value1")

		cache.Delete("key2") // Attempt to delete a non-existing key

		value, found := cache.Get("key1")
		if !found {
			t.Errorf("expected key1 to still be in the cache")
		}
		if value != "value1" {
			t.Errorf("expected value1, got %v", value)
		}
	})

	t.Run("Delete updates LRU order", func(t *testing.T) {
		cache := newLRU(2, deleteFinalizer, evictFinalizer)
		cache.Put("key1", "value1")
		cache.Put("key2", "value2")

		// Delete key1
		cache.Delete("key1")

		// Add another key to ensure no eviction of key2
		cache.Put("key3", "value3")

		value, found := cache.Get("key2")
		if !found {
			t.Errorf("expected key2 to still be in the cache")
		}
		if value != "value2" {
			t.Errorf("expected value2, got %v", value)
		}

		value, found = cache.Get("key3")
		if !found {
			t.Errorf("expected key3 to still be in the cache")
		}
		if value != "value3" {
			t.Errorf("expected value3, got %v", value)
		}
	})
}

func TestLRUCache_Clear(t *testing.T) {
	deleteFinalizer := func(key string, value any) {}
	evictFinalizer := func(key string, value any) {}

	t.Run("Clear cache", func(t *testing.T) {
		cache := newLRU(2, deleteFinalizer, evictFinalizer)
		cache.Put("key1", "value1")
		cache.Put("key2", "value2")

		// Clear the cache
		cache.Clear()

		// Ensure the cache is empty
		_, found := cache.Get("key1")
		if found {
			t.Errorf("expected key1 to be cleared from the cache")
		}

		_, found = cache.Get("key2")
		if found {
			t.Errorf("expected key2 to be cleared from the cache")
		}

		// Ensure the internal list is empty
		if cache.list.Len() != 0 {
			t.Errorf("expected internal list to be empty, got length %d", cache.list.Len())
		}

		// Ensure the internal map is empty
		if len(cache.cache) != 0 {
			t.Errorf("expected internal map to be empty, got length %d", len(cache.cache))
		}
	})
}
