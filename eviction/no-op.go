package eviction

// make sure nilEviction implements the Eviction interface
var _ Eviction = (*nilEviction)(nil)

// nilEviction is a no-op/dummy eviction implementation
type nilEviction struct{}

// Get retrieves a value from the eviction cache given a key.
func (n *nilEviction) Get(key string) (any, bool) {
	return nil, true
}

// Put adds a key-value pair to the eviction cache.
func (n *nilEviction) Put(key string, value any) {}

// Delete removes a key from the eviction cache.
func (n *nilEviction) Delete(key string) {}

// Clear clears the eviction cache
func (n *nilEviction) Clear() {}
