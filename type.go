package inmem

import "time"

type Item struct {
	Object     any
	Expiration int64
}

func (i *Item) Expired(now time.Time) bool {
	if i.Expiration == 0 {
		return false
	}

	return now.UnixNano() > i.Expiration
}

type Options struct {
	Finalizer       func(string, any)
	TransactionType TransactionType
	Sync            bool
	SyncFilePath    string
	SyncInterval    time.Duration
}

type txStore struct {
	changes map[string]Item
	deletes map[string]struct{}
}
