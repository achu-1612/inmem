package inmem

import "time"

// Options represents the options for the cache store initialization.
type Options struct {
	Finalizer       func(string, any)
	TransactionType TransactionType
	Sync            bool
	SyncFolderPath  string
	SyncInterval    time.Duration

	Sharding     bool
	ShardCount   int
	HashFunction func(string) uint32
}
