package eviction

import (
	"testing"
)

func TestLRUCache_Put(t *testing.T) {
	t.Run("Add items within capacity", func(t *testing.T) {
		cache := newLRU(3, nil)
		cache.Put("key1", "value1")
		cache.Put("key2", "value2")
		cache.Put("key3", "value3")

		if val, found := cache.Get("key1"); !found || val != "value1" {
			t.Errorf("Expected key1 to have value 'value1', got %v", val)
		}
		if val, found := cache.Get("key2"); !found || val != "value2" {
			t.Errorf("Expected key2 to have value 'value2', got %v", val)
		}
		if val, found := cache.Get("key3"); !found || val != "value3" {
			t.Errorf("Expected key3 to have value 'value3', got %v", val)
		}
	})

	t.Run("Evict least recently used item", func(t *testing.T) {
		cache := newLRU(2, nil)
		cache.Put("key1", "value1")
		cache.Put("key2", "value2")
		cache.Put("key3", "value3") // This should evict "key1"

		if _, found := cache.Get("key1"); found {
			t.Error("Expected key1 to be evicted, but it was found")
		}
		if val, found := cache.Get("key2"); !found || val != "value2" {
			t.Errorf("Expected key2 to have value 'value2', got %v", val)
		}
		if val, found := cache.Get("key3"); !found || val != "value3" {
			t.Errorf("Expected key3 to have value 'value3', got %v", val)
		}
	})

	t.Run("Update existing key", func(t *testing.T) {
		cache := newLRU(2, nil)
		cache.Put("key1", "value1")
		cache.Put("key1", "valueUpdated")

		if val, found := cache.Get("key1"); !found || val != "valueUpdated" {
			t.Errorf("Expected key1 to have updated value 'valueUpdated', got %v", val)
		}
	})

	t.Run("Evict after accessing an item", func(t *testing.T) {
		cache := newLRU(2, nil)
		cache.Put("key1", "value1")
		cache.Put("key2", "value2")
		cache.Get("key1")           // Access key1 to make it recently used
		cache.Put("key3", "value3") // This should evict "key2"

		if _, found := cache.Get("key2"); found {
			t.Error("Expected key2 to be evicted, but it was found")
		}
		if val, found := cache.Get("key1"); !found || val != "value1" {
			t.Errorf("Expected key1 to have value 'value1', got %v", val)
		}
		if val, found := cache.Get("key3"); !found || val != "value3" {
			t.Errorf("Expected key3 to have value 'value3', got %v", val)
		}
	})

	t.Run("Add items to full capacity and clear", func(t *testing.T) {
		cache := newLRU(2, nil)
		cache.Put("key1", "value1")
		cache.Put("key2", "value2")
		cache.Clear()

		if _, found := cache.Get("key1"); found {
			t.Error("Expected key1 to be cleared, but it was found")
		}
		if _, found := cache.Get("key2"); found {
			t.Error("Expected key2 to be cleared, but it was found")
		}
	})

	t.Run("Delete an existing key", func(t *testing.T) {
		cache := newLRU(2, nil)
		cache.Put("key1", "value1")
		cache.Delete("key1")

		if _, found := cache.Get("key1"); found {
			t.Error("Expected key1 to be deleted, but it was found")
		}
	})

	t.Run("Delete a non-existing key", func(t *testing.T) {
		cache := newLRU(2, nil)
		cache.Put("key1", "value1")
		cache.Delete("key2") // Deleting a non-existing key should not cause issues

		if val, found := cache.Get("key1"); !found || val != "value1" {
			t.Errorf("Expected key1 to have value 'value1', got %v", val)
		}
	})
}
