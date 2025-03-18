package inmem

import (
	"context"
	"hash/fnv"
)

const (
	defaultNumShards = 5
	minNumShards     = 3
)

// shardedCache represents a sharded cache instance.
type shardedCache struct {
	shards    []Cache
	numShards int
	hashFn    func(string) uint32
	// getShardFun func(string) uint32
}

// NewShardedCache returns a new sharded cache instance.
func NewShardedCache(ctx context.Context, opt Options) Cache {
	numShards := opt.ShardCount
	if numShards < minNumShards {
		numShards = defaultNumShards // Default number of shards
	}

	shards := make([]Cache, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = NewCache(ctx, opt, i)
	}

	sc := &shardedCache{
		shards:    shards,
		numShards: numShards,
		hashFn:    opt.HashFunction,
	}

	if sc.hashFn == nil {
		sc.hashFn = defaultHashFn
	}

	return sc
}

// defaultHashFn is the default hash function used to determine the shard for a given key.
func defaultHashFn(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))

	return h.Sum32()
}

// func defaultGetShardFun(hashFn func(string) uint32, shardCount int, key string) uint32 {
// 	return hashFn(key) % uint32(shardCount)
// }

// getShard returns the shard for a given key.
func (sc *shardedCache) getShard(key string) Cache {
	shardIndex := sc.hashFn(key) % uint32(sc.numShards)

	return sc.shards[shardIndex]
}

// Size returns the total number of items in the cache.
func (sc *shardedCache) Size() int {
	totalSize := 0

	for _, shard := range sc.shards {
		totalSize += shard.Size()
	}

	return totalSize
}

// Get returns a value from the cache given a key.
func (sc *shardedCache) Get(key string) (any, bool) {
	shard := sc.getShard(key)

	return shard.Get(key)
}

// Set sets a key in the cache with a value and a time-to-live (TTL) in seconds.
func (sc *shardedCache) Set(key string, value any, ttl int64) {
	shard := sc.getShard(key)
	shard.Set(key, value, ttl)
}

// Delete deletes a key from the cache.
func (sc *shardedCache) Delete(key string) {
	shard := sc.getShard(key)
	shard.Delete(key)
}

// Clear clears all items from the cache.
func (sc *shardedCache) Clear() {
	for _, shard := range sc.shards {
		shard.Clear()
	}
}

// Dump saves the cache to the given file.
func (sc *shardedCache) Dump() error {
	for _, shard := range sc.shards {
		if err := shard.Dump(); err != nil {
			return err
		}
	}

	return nil
}

// TransactionType returns the type of transaction used by the cache.
func (sc *shardedCache) TransactionType() TransactionType {
	return sc.shards[0].TransactionType()
}

// Begin starts a new transaction.
func (sc *shardedCache) Begin() error {
	for _, shard := range sc.shards {
		if err := shard.Begin(); err != nil {
			return err
		}
	}

	return nil
}

// Commit commits the current transaction.
func (sc *shardedCache) Commit() error {
	for _, shard := range sc.shards {
		if err := shard.Commit(); err != nil {
			return err
		}
	}

	return nil
}

// Rollback rolls back the current transaction.
func (sc *shardedCache) Rollback() error {
	for _, shard := range sc.shards {
		if err := shard.Rollback(); err != nil {
			return err
		}
	}

	return nil
}
