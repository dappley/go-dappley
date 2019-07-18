package network

type Storage interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, val []byte) error
}
