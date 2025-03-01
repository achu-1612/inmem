package inmem

//go:generate mockgen -package inmem -destination spec.mock.go -source spec.go -self_package "github.com/achu-1612/inmem"

// Cache is the interface that defines the methods for a cache store.
type Cache interface {
	// Size returns the number of items in the cache.
	Size() int

	// // Clear clears all items from the cache.
	Clear()

	// Set sets a key in the cache with a value and a time-to-live (TTL) in seconds.
	Set(string, interface{}, int64)

	// Get returns a value from the cache given a key.
	Get(string) (interface{}, bool)

	// Delete deletes a key from the cache.
	Delete(string)
}
