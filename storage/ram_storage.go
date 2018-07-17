package storage

import (
	"sync"
	"errors"
)

var (
	ErrKeyInvalid = errors.New("Key is invalid")
)


type RamStorage struct{
	data *sync.Map
}

func NewRamStorage() *RamStorage{
	return &RamStorage{
		data:new(sync.Map),
	}
}

func (rs *RamStorage) Get(key []byte) ([]byte, error){
	value, ok := rs.data.Load(string(key))
	if ok {
		return value.([]byte), nil
	}
	return nil, ErrKeyInvalid
}

func (rs *RamStorage) Put(key []byte, val []byte){
	rs.data.Store(string(key),val)
}

func (rs *RamStorage) Close() error{
	rs.data.Range(func(key,value interface{}) bool{
		rs.data.Delete(key)
		return true
	})
	return nil
}


