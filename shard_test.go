package inmem

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	ctx := context.Background()
	opt := Options{
		ShardCount: 10,
		DebugLogs:  false,
		SupressLog: true,
		// ShardIndexCache:     true,
		// ShardIndexCacheSize: 10000,
		// Add other options as needed
	}

	cache := NewShardedCache(ctx, opt)

	n := 200000

	now := time.Now()

	for i := 0; i < n; i++ {
		cache.Set("key"+fmt.Sprint(i), "value", 0)
	}

	for i := 0; i < n; i++ {
		cache.Get("key" + fmt.Sprint(i))
	}

	t.Log(time.Since(now))

}

func BenchmarkShardedCache(b *testing.B) {
	ctx := context.Background()
	opt := Options{
		ShardCount:          10,
		DebugLogs:           false,
		SupressLog:          true,
		ShardIndexCache:     true,
		ShardIndexCacheSize: 100,
	}

	cache := NewShardedCache(ctx, opt)

	b.Run("Write", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Set("key"+fmt.Sprint(i), "value"+fmt.Sprint(i), int64(time.Hour.Seconds()))
		}
	})

	b.Run("Read", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Get("key" + fmt.Sprint(i))
		}
	})

	b.Run("Delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Delete("key" + fmt.Sprint(i))
		}
	})

	b.ReportAllocs()
}

func BenchmarkMemoryUsage(b *testing.B) {
	ctx := context.Background()
	opt := Options{
		ShardCount:          10,
		ShardIndexCache:     true,
		ShardIndexCacheSize: 1000,
		// Add other options as needed
	}

	cache := NewShardedCache(ctx, opt)

	b.Run("MemoryUsage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Set("key"+fmt.Sprint(i), "value"+fmt.Sprint(i), int64(time.Hour.Seconds()))
		}
		b.ReportAllocs()
	})
}

// write a benchmark to store keys and access some keys multiple times, many times

func BenchmarkStore(b *testing.B) {
	ctx := context.Background()
	opt := Options{
		ShardCount: 10,
		DebugLogs:  false,
		SupressLog: true,
		// ShardIndexCache:     true,
		ShardIndexCacheSize: 10000,
		// Add other options as needed
	}

	cache := NewShardedCache(ctx, opt)

	n := 200000

	for i := 0; i < n; i++ {
		cache.Set("key"+fmt.Sprint(i), "value", 0)
	}

	b.Run("Read", func(b *testing.B) {
		b.Log(b.N)
		for i := 0; i < b.N; i++ {
			cache.Get("key" + fmt.Sprint(i))
		}
	})

	b.ReportAllocs()
}

func BenchmarkGet(b *testing.B) {
	ctx := context.Background()
	opt := Options{
		ShardCount:          10,
		DebugLogs:           false,
		SupressLog:          true,
		ShardIndexCache:     false,
		ShardIndexCacheSize: 10000,
		// Add other options as needed
	}

	cache := NewShardedCache(ctx, opt)

	n := 2000

	for i := 0; i < n; i++ {
		cache.Set("key"+fmt.Sprint(i), "value", 0)
	}

	b.Run("Read", func(b *testing.B) {
		b.Log(b.N)

		for i := 0; i < b.N; i++ {
			cache.Get("key0")
		}
	})

	b.ReportAllocs()
}
