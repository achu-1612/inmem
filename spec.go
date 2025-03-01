package inmem

type Cache interface {
	Len() int
	Clear()
	Set(string, interface{}, int64)
	Get(string) (interface{}, bool)
	Delete(string)
}
