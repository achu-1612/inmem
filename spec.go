package inmem

//go:generate mockgen -package inmem -destination spec.mock.go -source spec.go -self_package "github.com/achu-1612/inmem"

type TransactionType string

const (
	TransactionTypeAtomic     TransactionType = "atomic"
	TransactionTypeOptimistic TransactionType = "optimistic"
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
