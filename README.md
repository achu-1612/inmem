# inmem

`inmem` is an in-memory cache library for Go, designed to provide a simple and efficient caching mechanism with support for sharding, transactions, and persistence.

## Features

- **In-Memory Caching**: Store key-value pairs in memory for fast access.
- **Sharding**: Distribute cache items across multiple shards for improved performance.
- **Transactions**: Support for atomic and optimistic transactions.
- **Persistence**: Periodically save cache data to disk and load it on startup.
- **Customizable**: Configure cache options such as sync interval, sharding, and hash functions.

## Installation

To install the package, run:

```sh
go get github.com/achu-1612/inmem
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/achu-1612/inmem"
)

func main() {
    cache, err := inmem.New(context.Background(), inmem.Options{})
    if err != nil {
        panic(err)
    }

    cache.Set("key", "value", 0)
    value, found := cache.Get("key")
    if found {
        fmt.Println("Found value:", value)
    } else {
        fmt.Println("Value not found")
    }
}
```

### Sharded Cache

```go
package main

import (
    "context"
    "github.com/achu-1612/inmem"
)

func main() {
    cache, err := inmem.New(context.Background(), inmem.Options{
        Sharding:   true,
        ShardCount: 4,
    })
    if err != nil {
        panic(err)
    }

    cache.Set("key", "value", 0)
    value, found := cache.Get("key")
    if found {
        fmt.Println("Found value:", value)
    } else {
        fmt.Println("Value not found")
    }
}
```

### Transactions

```go
package main

import (
    "context"
    "github.com/achu-1612/inmem"
)

func main() {
    cache, err := inmem.New(context.Background(), inmem.Options{
        TransactionType: inmem.TransactionTypeAtomic,
    })
    if err != nil {
        panic(err)
    }

    cache.Begin()
    cache.Set("key", "value", 0)
    cache.Commit()

    value, found := cache.Get("key")
    if found {
        fmt.Println("Found value:", value)
    } else {
        fmt.Println("Value not found")
    }
}
```

### Persistence

```go
package main

import (
    "context"
    "github.com/achu-1612/inmem"
    "time"
)

func main() {
    cache, err := inmem.New(context.Background(), inmem.Options{
        Sync:           true,
        SyncInterval:   time.Minute,
        SyncFolderPath: "cache_data",
    })
    if err != nil {
        panic(err)
    }

    cache.Set("key", "value", 0)
    value, found := cache.Get("key")
    if found {
        fmt.Println("Found value:", value)
    } else {
        fmt.Println("Value not found")
    }
}
```

## API

### `func New(ctx context.Context, opt Options) (Cache, error)`

Creates a new cache instance with the given options.

### `func (c *cache) Set(key string, value any, ttl int64)`

Sets a key in the cache with a value and a time-to-live (TTL) in seconds.

### `func (c *cache) Get(key string) (any, bool)`

Gets a value from the cache given a key.

### `func (c *cache) Delete(key string)`

Deletes a key from the cache.

### `func (c *cache) Clear()`

Clears all items from the cache.

### `func (c *cache) Begin() error`

Begins a new transaction.

### `func (c *cache) Commit() error`

Commits the current transaction.

### `func (c *cache) Rollback() error`

Rolls back the current transaction.

### `func (c *cache) Dump() error`

Saves the cache data to disk.

## License

This project is licensed under the MIT License.

