package inmem

import (
	"container/list"
)

/*
	Refer the paper -  An O(1) algorith for implementing the  LFU cache eviction scheme
	Dt- 16 Aug 2010
*/

// entry store the key, shard index and frequency of the key
type entry struct {
	key       string
	shardIdx  uint32
	frequency int
}

// LFUCache is a cache that evicts the least frequently used item
type LFUCache struct {
	maxSize   int
	size      int
	cache     map[string]*list.Element
	frequency map[int]*list.List
	minFreq   int
}

// NewLFUCache returns a new LFUCache instance
func NewLFUCache(maxSize int) *LFUCache {
	return &LFUCache{
		maxSize:   maxSize,
		cache:     make(map[string]*list.Element),
		frequency: make(map[int]*list.List),
	}
}

// Get returns the shard index for the given key
func (c *LFUCache) Get(key string) (uint32, bool) {
	if elem, ok := c.cache[key]; ok {
		c.incrementFrequency(elem)

		return elem.Value.(*entry).shardIdx, true
	}

	return 0, false
}

// Put inserts the key and shard index into the cache
func (c *LFUCache) Put(key string, shardIdx uint32) {
	if c.maxSize == 0 {
		return
	}

	if elem, ok := c.cache[key]; ok {
		elem.Value.(*entry).shardIdx = shardIdx
		c.incrementFrequency(elem)

		return
	}

	if c.size == c.maxSize {
		c.evict()
	}

	newEntry := &entry{key: key, shardIdx: shardIdx, frequency: 1}

	if c.frequency[1] == nil {
		c.frequency[1] = list.New()
	}

	elem := c.frequency[1].PushFront(newEntry)

	c.cache[key] = elem
	c.size++
	c.minFreq = 1
}

// incrementFrequency increments the frequency of the key
func (c *LFUCache) incrementFrequency(elem *list.Element) {
	entry := elem.Value.(*entry)
	freq := entry.frequency

	c.frequency[freq].Remove(elem)

	if c.frequency[freq].Len() == 0 {
		delete(c.frequency, freq)

		if c.minFreq == freq {
			c.minFreq++
		}
	}

	entry.frequency++

	if c.frequency[entry.frequency] == nil {
		c.frequency[entry.frequency] = list.New()
	}

	c.frequency[entry.frequency].PushFront(elem)
}

// evict evicts the least frequently used item
func (c *LFUCache) evict() {
	if c.minFreq == 0 {
		return
	}

	list := c.frequency[c.minFreq]

	elem := list.Back()

	if elem != nil {
		list.Remove(elem)

		entry := elem.Value.(*entry)

		delete(c.cache, entry.key)

		c.size--

		if list.Len() == 0 {
			delete(c.frequency, c.minFreq)
		}
	}
}
