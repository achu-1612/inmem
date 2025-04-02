package inmem

//go:generate mockgen -package inmem -destination spec.mock.go -source spec.go -self_package "github.com/achu-1612/inmem"

type TransactionType string

const (
	TransactionTypeAtomic     TransactionType = "atomic"
	TransactionTypeOptimistic TransactionType = "optimistic"
)

type EvictionPolicy string

const (
	EvictionPolicyNil EvictionPolicy = "nil"
	EvictionPolicyLFU EvictionPolicy = "lfu"
)

// Cache is the interface that defines the methods for a cache store.
type Cache interface {
	// Size returns the number of items in the cache.
	Size() int

	// Get returns a value from the cache given a key.
	Get(key string) (any, bool)

	// Set sets a key in the cache with a value and a time-to-live (TTL).
	Set(key string, value any, ttl int64)

	// Delete deletes a key from the cache.
	Delete(key string)

	// Clear clears all items from the cache.
	Clear()

	// Dump saves the cache to the configured directory.
	Dump() error

	// TransactionType returns the type of the transaction.
	TransactionType() TransactionType

	// Begin starts a transaction.
	Begin() error

	// Commit commits a transaction.
	Commit() error

	// Rollback rolls back a transaction.
	Rollback() error
}

// Eviction is the interface that defines the methods for an eviction policy.
type Eviction interface {
	Delete(key string)
	Put(key string, value any)
	Get(key string) (any, bool)
	// Clear clears the eviction acache
	Clear()
}

type ShardIndexResolver interface {
	// GetShardIndex returns the shard index for a given key.
	GetShardIndex(key string) uint32
}

// LFUResource is an interface that represents a resource in the LFUCache
// It allows the cache pluggable to any type of resource/value
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
