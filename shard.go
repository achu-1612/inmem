package inmem

import (
	"context"
	"fmt"
	"hash/fnv"
)

const (
	defaultNumShards = 5
	minNumShards     = 3
)

// shardedCache represents a sharded cache instance.
type shardedCache struct {
	shards        []Cache
	numShards     uint32
	indexResolver ShardIndexResolver
}

// NewShardedCache returns a new sharded cache instance.
func NewShardedCache(ctx context.Context, opt Options) (Cache, error) {
	numShards := opt.ShardCount
	if numShards < minNumShards {
		numShards = defaultNumShards // Default number of shards
	}

	shards := make([]Cache, numShards)
	for i := 0; i < int(numShards); i++ {
		c, err := NewCache(ctx, opt, i)
		if err != nil {
			return nil, fmt.Errorf("new cache: %v", err)
		}

		shards[i] = c
	}

	sc := &shardedCache{
		shards:    shards,
		numShards: numShards,
	}

	if opt.ShardIndexCache {
		lfu, err := NewEviction(
			EvictionOptions{
				Policy:    EvictionPolicyLFU,
				MaxSize:   opt.ShardIndexCacheSize,
				Allocator: func(s string) LFUResource { return &shardIndexEntry{key: s} },
			},
		)

		if err != nil {
			return nil, fmt.Errorf("new lfu cache: %v", err)
		}

		sc.indexResolver = &shardResolverWithCache{
			numShards: numShards,
			lfu:       lfu,
		}
	} else {
		sc.indexResolver = &shardResolverWithoutCache{
			numShards: numShards,
		}
	}

	return sc, nil
}

// getShard returns the shard for a given key.
func (sc *shardedCache) getShard(key string) Cache {
	return sc.shards[sc.indexResolver.GetShardIndex(key)]
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

var _ ShardIndexResolver = &shardResolverWithoutCache{}
var _ ShardIndexResolver = &shardResolverWithCache{}

type shardResolverWithoutCache struct {
	numShards uint32
}

func (sr *shardResolverWithoutCache) GetShardIndex(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))

	return h.Sum32() % sr.numShards
}

type shardResolverWithCache struct {
	numShards uint32
	lfu       Eviction
}

func (sr *shardResolverWithCache) GetShardIndex(key string) uint32 {
	if shardIdx, ok := sr.lfu.Get(key); ok {
		return shardIdx.(uint32)
	}

	h := fnv.New32a()
	h.Write([]byte(key))

	idx := h.Sum32() % sr.numShards

	sr.lfu.Put(key, idx)

	return idx
}

// shardIndexEntry implements the LFUResource interface,
// so that it can be used with the LFUCache.
type shardIndexEntry struct {
	key       string
	shardIdx  uint32
	frequency int
}

func (e *shardIndexEntry) Value() any {
	return e.shardIdx
}

func (e *shardIndexEntry) Set(value any) {
	e.shardIdx = value.(uint32)
}

func (e *shardIndexEntry) Frequency() int {
	return e.frequency
}

func (e *shardIndexEntry) IncrementFrequency() {
	e.frequency++
}

func (e *shardIndexEntry) Key() string {
	return e.key
}
