package inmem

import (
	"context"
	"encoding/gob"
	"os"
)

// New creates a new in-memory cache
func New(ctx context.Context, opt Options) (Cache, error) {
	gob.Register(Item{})

	if opt.SyncFolderPath != "" {
		// create folder if not exists
		if err := os.MkdirAll(opt.SyncFolderPath, os.ModePerm); err != nil {
			return nil, err
		}
	}

	if opt.Sharding {
		return NewShardedCache(ctx, opt), nil
	}

	return NewCache(ctx, opt, 0), nil
}
