package inmem

import (
	"container/list"
)

// LFUResourceAllocator is a function that returns a new LFUResource
type LFUResourceAllocator func(string) LFUResource

// make sure LFUResource implements the LFUResource interface
var _ Eviction = (*LFUCache)(nil)

// LFUCache is a cache that evicts the least frequently used item
// It implements the Eviction interface
type LFUCache struct {
	maxSize           int
	size              int
	cache             map[string]*list.Element
	frequency         map[int]*list.List
	minFreq           int
	resourceAllocator func(string) LFUResource
	finalizer         func(string, any)
}

func (c *LFUCache) Get(key string) (any, bool) {
	if elem, ok := c.cache[key]; ok {
		c.incrementFrequency(elem)

		return elem.Value.(LFUResource).Value(), true
	}

	return nil, false
}

func (c *LFUCache) Put(key string, data any) {
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

	res := c.resourceAllocator(key)
	res.Set(data)
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
func (c *LFUCache) incrementFrequency(elem *list.Element) {
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
func (c *LFUCache) evict() {
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

func (c *LFUCache) Delete(key string) {
	if elem, ok := c.cache[key]; ok {
		c.deleteElement(elem)
	}
}

// deleteElement deletes the element from the cache
func (c *LFUCache) deleteElement(elem *list.Element) {
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

// Clear clears/resets the cache.
func (c *LFUCache) Clear() {
	c.cache = make(map[string]*list.Element)
	c.frequency = make(map[int]*list.List)
	c.size = 0
	c.minFreq = 0
}

// make sure NilEviction implements the Eviction interface
var _ Eviction = (*NilEviction)(nil)

// NilEviction is a no-op/dummy eviction implementation
type NilEviction struct{}

func (n *NilEviction) Delete(key string) {}

func (n *NilEviction) Put(key string, value any) {}

func (n *NilEviction) Get(key string) (any, bool) {
	return nil, true
}

func (n *NilEviction) Clear() {}

// NewEviction returns a new eviction implementation based on the options provided.
func NewEviction(
	opts EvictionOptions,
) (Eviction, error) {
	if opts.Policy != EvictionPolicyLFU {
		return &NilEviction{}, nil
	}

	if opts.MaxSize <= 0 {
		return nil, ErrInvalidMaxSizeForEviction
	}

	if opts.Allocator == nil {
		// use a default allocator if none is provided
		opts.Allocator = func(key string) LFUResource {
			return &defaultEvictionEntry{key: key}
		}
	}

	if opts.Finalizer == nil {
		// use a no-op finalizer if none is provided
		opts.Finalizer = func(string, any) {}
	}

	return &LFUCache{
		maxSize:           opts.MaxSize,
		cache:             make(map[string]*list.Element),
		frequency:         make(map[int]*list.List),
		resourceAllocator: opts.Allocator,
		finalizer:         opts.Finalizer,
	}, nil
}

// make sure defaultEvictionEntry implements the LFUResource interface
var _ LFUResource = (*defaultEvictionEntry)(nil)

// defaultEvictionEntry implements the LFUResource interface,
// so that it can be used with the LFUCache.
type defaultEvictionEntry struct {
	key       string
	value     any
	frequency int
}

func (e *defaultEvictionEntry) Value() any {
	return e.value
}

func (e *defaultEvictionEntry) Set(value any) {
	e.value = value
}

func (e *defaultEvictionEntry) Frequency() int {
	return e.frequency
}

func (e *defaultEvictionEntry) IncrementFrequency() {
	e.frequency++
}

func (e *defaultEvictionEntry) Key() string {
	return e.key
}
