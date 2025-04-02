package inmem

import "errors"

var (
	ErrorInTransaction             = errors.New("already in transaction")
	ErrNotInTransaction            = errors.New("not in transaction")
	ErrConcurrentModification      = errors.New("concurrent modification")
	ErrInvalidMaxSizeForEviction   = errors.New("invalid lfu max size")
	ErrInvalidAllocatorForEviction = errors.New("invalid lfu allocator")
)
