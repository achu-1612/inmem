package eviction

import (
	"testing"
	"time"
)

func TestTTL_Pop(t *testing.T) {
	tests := []struct {
		name     string
		actions  func(ttl TTL)
		expected string
	}{
		{
			name: "Pop single item",
			actions: func(ttl TTL) {
				ttl.Put("key1", time.Now().Unix())
			},
			expected: "key1",
		},
		{
			name: "Pop multiple items in order of expiration",
			actions: func(ttl TTL) {
				ttl.Put("key1", time.Now().Add(10*time.Second).Unix())
				ttl.Put("key2", time.Now().Add(5*time.Second).Unix())
				ttl.Put("key3", time.Now().Add(15*time.Second).Unix())
			},
			expected: "key2",
		},
		{
			name: "Pop after deleting an item",
			actions: func(ttl TTL) {
				ttl.Put("key1", time.Now().Add(10*time.Second).Unix())
				ttl.Put("key2", time.Now().Add(5*time.Second).Unix())
				ttl.Delete("key2")
			},
			expected: "key1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttl := NewTTL()
			tt.actions(ttl)

			result := ttl.Pop()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTTL_AllMethods(t *testing.T) {
	t.Run("Test Len", func(t *testing.T) {
		ttl := NewTTL()
		if ttl.Len() != 0 {
			t.Errorf("expected length 0, got %d", ttl.Len())
		}
		ttl.Put("key1", time.Now().Unix())
		if ttl.Len() != 1 {
			t.Errorf("expected length 1, got %d", ttl.Len())
		}
	})

	t.Run("Test Top", func(t *testing.T) {
		ttl := NewTTL()
		if ttl.Top() != nil {
			t.Errorf("expected nil, got %v", ttl.Top())
		}
		ttl.Put("key1", time.Now().Add(10*time.Second).Unix())
		ttl.Put("key2", time.Now().Add(5*time.Second).Unix())
		top := ttl.Top()
		if top.Key() != "key2" {
			t.Errorf("expected key2, got %s", top.Key())
		}
	})

	t.Run("Test Delete", func(t *testing.T) {
		ttl := NewTTL()
		ttl.Put("key1", time.Now().Unix())
		ttl.Delete("key1")
		if ttl.Len() != 0 {
			t.Errorf("expected length 0 after delete, got %d", ttl.Len())
		}
	})

	t.Run("Test Clear", func(t *testing.T) {
		ttl := NewTTL()
		ttl.Put("key1", time.Now().Unix())
		ttl.Put("key2", time.Now().Unix())
		ttl.Clear()
		if ttl.Len() != 0 {
			t.Errorf("expected length 0 after clear, got %d", ttl.Len())
		}
	})
}
