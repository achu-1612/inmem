package inmem

import (
	"time"
)

// Options represents the options for the cache store initialization.
type Options struct {
	// Finalizer is the finalizer function that is called when an item is evicted from the cache.
	// Note: When in transaction, the finalizer is called after the transaction is committed.
	Finalizer func(string, any)

	// TransactionType is the type of transaction to be used.
	// Options are:
	// 1. Optimistic
	// 2. Pessimistic
	TransactionType TransactionType

	// Sync enables the cache to be synchronized with a folder.
	// When sharding is enabled, the defined folder will container mutliple .gob files (one per peach shard)
	Sync bool
	// SyncFolderPath is the path to the folder where the cache will be synchronized.
	SyncFolderPath string
	// SyncInterval is the interval at which the cache will be synchronized.
	SyncInterval time.Duration

	// Sharding enables the cache to be sharded.
	Sharding bool
	// ShardCount is the number of shards to be created.
	ShardCount int
	// HashFunction is the hash function to be used to determine the shard for a given key.
	HashFunction func(string) uint32
	// ShardIndexCache enables the cache to store the shard index for a given key.
	// Note: It will save calling the hash function for the same key multiple times.
	ShardIndexCache bool

	// SupressLog suppresses the logs.
	SupressLog bool
	// DebugLogs enables the debug logs.
	DebugLogs bool
}
