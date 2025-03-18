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
		// Add other options as needed
	}

	cache := NewShardedCache(ctx, opt)

	n := 2000000

	now := time.Now()

	for i := 0; i < n; i++ {
		cache.Set("key"+fmt.Sprint(i), "value", 0)
	}

	t.Log(time.Since(now))

}

func BenchmarkShardedCache(b *testing.B) {
	ctx := context.Background()
	opt := Options{
		ShardCount: 10,
		DebugLogs:  false,
		SupressLog: true,
		// Add other options as needed
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
		ShardCount: 10,
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
