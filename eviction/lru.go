package eviction

import (
	"container/list"
)

// make sure NilEviction implements the Eviction interface
var _ Eviction = (*lruCache)(nil)

// lruCache represents a Least Recently Used cache.
type lruCache struct {
	capacity  int
	cache     map[string]*list.Element
	list      *list.List
	finalizer func(string, any)
}

// NewLRUCache creates a new LRUCache with the specified capacity.
func newLRU(capacity int, finalizer func(string, any)) *lruCache {
	return &lruCache{
		capacity:  capacity,
		cache:     make(map[string]*list.Element),
		list:      list.New(),
		finalizer: finalizer,
	}
}

// lruItem represents a key-value pair stored in the cache.
type lruItem struct {
	key   string
	value any
}

// Get retrieves a value from the eviction cache given a key.
func (lru *lruCache) Get(key string) (any, bool) {
	if elem, found := lru.cache[key]; found {
		lru.list.MoveToFront(elem)

		return elem.Value.(*lruItem).value, true
	}

	return nil, false
}

// Put adds a key-value pair to the eviction cache.
func (lru *lruCache) Put(key string, value any) {
	if elem, found := lru.cache[key]; found {
		elem.Value.(*lruItem).value = value
		lru.list.MoveToFront(elem)

		return
	}

	if lru.list.Len() >= lru.capacity {
		lru.evict()
	}

	entry := &lruItem{key: key, value: value}
	elem := lru.list.PushFront(entry)
	lru.cache[key] = elem
}

// Delete removes a key from the eviction cache.
func (lru *lruCache) Delete(key string) {
	if elem, found := lru.cache[key]; found {
		lru.list.Remove(elem)

		delete(lru.cache, key)
	}
}

// evict removes the least recently used item from the cache.
func (lru *lruCache) evict() {
	back := lru.list.Back()
	if back != nil {
		entry := back.Value.(*lruItem)
		delete(lru.cache, entry.key)
		lru.list.Remove(back)
	}
}

// Clear clears the eviction cache
func (lru *lruCache) Clear() {
	lru.cache = make(map[string]*list.Element)
	lru.list = list.New()
}
