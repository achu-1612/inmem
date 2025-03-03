package inmem

//go:generate mockgen -package inmem -destination spec.mock.go -source spec.go -self_package "github.com/achu-1612/inmem"

// Cache is the interface that defines the methods for a cache store.
type Cache interface {
	// Size returns the number of items in the cache.
	Size() int

	// Get returns a value from the cache given a key.
	Get(key string) (interface{}, bool)

	// List returns a list of all items in the cache.
	// List() ([]interface{}, error)

	// Set sets a key in the cache with a value and a time-to-live (TTL).
	Set(key string, value interface{}, ttl int64)

	// Delete deletes a key from the cache.
	Delete(key string)

	// Clear clears all items from the cache.
	Clear()
}

type TransactionType string

const (
	TransactionTypeAtomic     TransactionType = "atomic"
	TransactionTypeOptimistic TransactionType = "optimistic"
)

// Transaction is the interface that defines the methods for a transaction.
type Transaction interface {
	// Type returns the type of the transaction.
	Type() string

	// Begin starts a transaction.
	Begin() error

	// Commit commits a transaction.
	Commit() error

	// Rollback rolls back a transaction.
	Rollback() error
}
