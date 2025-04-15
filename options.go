package inmem

import (
	"time"

	"github.com/achu-1612/inmem/eviction"
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
	// When sharding is enabled, the defined folder will container multiple .gob files (one per peach shard)
	Sync bool
	// SyncFolderPath is the path to the folder where the cache will be synchronized.
	SyncFolderPath string
	// SyncInterval is the interval at which the cache will be synchronized.
	SyncInterval time.Duration

	// Sharding enables the cache to be sharded.
	Sharding bool
	// ShardCount is the number of shards to be created.
	ShardCount uint32

	// SupressLog suppresses the logs.
	SupressLog bool
	// DebugLogs enables the debug logs.
	DebugLogs bool

	// EvictionPolicy is the eviction policy to be used.
	EvictionPolicy eviction.Policy

	// MaxSize is the maximum size of the cache, after this size is reached, the cache will start evicting items.
	MaxSize int
}
