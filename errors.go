package inmem

import "errors"

var (
	ErrorInTransaction  = errors.New("already in transaction")
	ErrNotInTransaction = errors.New("not in transaction")
)
