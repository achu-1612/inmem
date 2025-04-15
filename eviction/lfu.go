package eviction

import (
	"container/list"
)

type LFUResource interface {
	// Key returns key for the eviction entry.
	Key() string
	// IncrementFrequency increments the key access frequency by 1.
	IncrementFrequency()
	// Frequency returns the frequency for the entry.
	Frequency() int
	// Value return the value for the entry.
	Value() any
	// Set sets the value for the entry.
	Set(any)
}

type lfuItem struct {
	key       string
	value     any
	frequency int
}

func (l *lfuItem) Key() string {
	return l.key
}

func (l *lfuItem) IncrementFrequency() {
	l.frequency++

}

func (l *lfuItem) Frequency() int {
	return l.frequency
}

func (l *lfuItem) Value() any {
	return l.value

}

func (l *lfuItem) Set(value any) {
	l.value = value
}

func newLFUResource(key string, value any) LFUResource {
	return &lfuItem{
		key:   key,
		value: value,
	}
}

// make sure LFUResource implements the LFUResource interface
var _ Eviction = (*lfuCache)(nil)

// lfuCache is a cache that evicts the least frequently used item
// It implements the Eviction interface.
type lfuCache struct {
	maxSize   int
	size      int
	cache     map[string]*list.Element
	frequency map[int]*list.List
	minFreq   int
	finalizer func(string, any)
}

func newLFU(maxSize int, finalizer func(string, any)) *lfuCache {
	return &lfuCache{
		maxSize:   maxSize,
		cache:     make(map[string]*list.Element),
		frequency: make(map[int]*list.List),
		finalizer: finalizer,
	}
}

// Get retrieves a value from the eviction cache given a key.
func (c *lfuCache) Get(key string) (any, bool) {
	if elem, ok := c.cache[key]; ok {
		c.incrementFrequency(elem)

		return elem.Value.(LFUResource).Value(), true
	}

	return nil, false
}

// Put adds a key-value pair to the eviction cache.
func (c *lfuCache) Put(key string, data any) {
	if c.maxSize == 0 {
		return
	}

	if elem, ok := c.cache[key]; ok {
		elem.Value.(LFUResource).Set(data)
		c.incrementFrequency(elem)

		return
	}

	if c.size == c.maxSize {
		c.evict()
	}

	res := newLFUResource(key, data)
	res.IncrementFrequency()

	if c.frequency[1] == nil {
		c.frequency[1] = list.New()
	}

	elem := c.frequency[1].PushFront(res)

	c.cache[key] = elem
	c.size++
	c.minFreq = 1
}

// incrementFrequency increments the frequency of the key
func (c *lfuCache) incrementFrequency(elem *list.Element) {
	entry := elem.Value.(LFUResource)
	freq := entry.Frequency()

	c.frequency[freq].Remove(elem)

	if c.frequency[freq].Len() == 0 {
		delete(c.frequency, freq)

		if c.minFreq == freq {
			c.minFreq++
		}
	}

	entry.IncrementFrequency()

	freq = entry.Frequency()

	if c.frequency[freq] == nil {
		c.frequency[freq] = list.New()
	}

	c.frequency[freq].PushFront(entry)
}

// evict evicts the least frequently used item
func (c *lfuCache) evict() {
	if c.minFreq == 0 {
		return
	}

	list := c.frequency[c.minFreq]

	elem := list.Back()

	if elem != nil {
		list.Remove(elem)

		entry := elem.Value.(LFUResource)

		delete(c.cache, entry.Key())
		c.finalizer(entry.Key(), entry.Value())

		c.size--

		if list.Len() == 0 {
			delete(c.frequency, c.minFreq)
		}
	}
}

// Delete removes a key from the eviction cache.
func (c *lfuCache) Delete(key string) {
	elem, ok := c.cache[key]
	if !ok {
		return
	}

	entry := elem.Value.(LFUResource)
	freq := entry.Frequency()

	c.frequency[freq].Remove(elem)

	if c.frequency[freq].Len() == 0 {
		delete(c.frequency, freq)

		if c.minFreq == freq {
			c.minFreq++
		}
	}

	delete(c.cache, entry.Key())

	c.size--
}

// Clear clears the eviction cache
func (c *lfuCache) Clear() {
	c.cache = make(map[string]*list.Element)
	c.frequency = make(map[int]*list.List)
	c.size = 0
	c.minFreq = 0
}
