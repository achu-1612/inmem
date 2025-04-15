package eviction

type Policy string

const (
	PolicyLRU  Policy = "lru"  // Least Recently Used
	PolicyFIFO Policy = "fifo" // First In First Out
	PolicyLFU  Policy = "lfu"  // Least Frequently Used
	PolicyARC  Policy = "arc"  // Adaptive Replacement Cache
)

const (
	defaultCapacity = 100 // Default capacity for the eviction cache
)

// Eviction is the interface that defines the methods for an eviction policy.
type Eviction interface {
	// Get retrieves a value from the eviction cache given a key.
	Get(key string) (any, bool)
	// Put adds a key-value pair to the eviction cache.
	Put(key string, value any)
	// Delete removes a key from the eviction cache.
	Delete(key string)
	// Clear clears the eviction cache
	Clear()
}

// Options holds the configuration for the eviction cache.
type Options struct {
	Capacity        int
	Policy          Policy
	DeleteFinalizer func(key string, value any)
	EvictFinalizer  func(key string, value any)
}

// New creates a new Eviction instance based on the provided options.
func New(opt Options) Eviction {
	if opt.DeleteFinalizer == nil {
		opt.DeleteFinalizer = func(key string, value any) {}
	}

	if opt.EvictFinalizer == nil {
		opt.EvictFinalizer = func(key string, value any) {}
	}

	if opt.Capacity <= 0 {
		opt.Capacity = defaultCapacity
	}

	switch opt.Policy {
	case PolicyLFU:
		return newLFU(opt.Capacity, opt.DeleteFinalizer, opt.EvictFinalizer)

	case PolicyLRU:
		return newLRU(opt.Capacity, opt.DeleteFinalizer, opt.EvictFinalizer)

	case PolicyFIFO:
		panic("FIFO eviction policy is not implemented yet")

	case PolicyARC:
		panic("ARC eviction policy is not implemented yet")

	default:
		return &nilEviction{}
	}
}
