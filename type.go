package inmem

type Item struct {
	Object     any
	Expiration int64
}

type Options struct {
	Finalizer       func(string, any)
	TransactionType TransactionType
}

type txStore struct {
	changes map[string]Item
	deletes map[string]struct{}
}
