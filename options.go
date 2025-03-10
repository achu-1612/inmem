package inmem

import "time"

// Options represents the options for the cache store initialization.
type Options struct {
	Finalizer       func(string, any)
	TransactionType TransactionType
	Sync            bool
	SyncFilePath    string
	SyncInterval    time.Duration
	NumShards       int
	HashFunction    func(string) uint32
}
