// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package storage

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRamStorage(t *testing.T) {
	r := NewRamStorage()
	assert.Equal(t, r.data, new(sync.Map))
}

//test get and put methods
func TestRamStorage_GetandPut(t *testing.T) {
	r := NewRamStorage()

	//setup various key scenario
	var keyValuePairs = []struct {
		key   []byte
		value []byte
	}{
		{[]byte("a"), []byte("b")}, //lowercase letter
		{[]byte("A"), []byte("B")}, //CAPITAL letter
		{[]byte("1"), []byte("2")}, //number
		{[]byte(""), []byte("3")},  //empty string
	}

	//for each key value pair. Store it, read it and compare if the value stays unchanged
	for i, kv := range keyValuePairs {
		t.Run(string(i), func(t *testing.T) {
			r.Put(kv.key, kv.value)
			v, err := r.Get(kv.key)
			assert.Nil(t, err)
			assert.Equal(t, v, kv.value)
		})
	}

	//Read Invalid Key
	_, err := r.Get([]byte("d"))

	//there should be an error returned
	assert.Equal(t, err, ErrKeyInvalid)
}

//test close method
func TestRamStorage_Close(t *testing.T) {
	r := NewRamStorage()

	//write a key-value pair
	r.Put([]byte("1"), []byte("2"))

	//close the storage
	r.Close()

	//the data should not be accessible
	v, err := r.Get([]byte("1"))

	//there should be an error returned
	assert.Equal(t, err, ErrKeyInvalid)
	//there should be not value returned
	assert.Nil(t, v)
}

//check if two storage instances affect each other
func TestRamStorage_IndependantStorage(t *testing.T) {
	r1 := NewRamStorage()
	r2 := NewRamStorage()

	//write a key-value pair in r1 storage should not affect r2
	r1.Put([]byte("1"), []byte("a"))
	r2.Put([]byte("1"), []byte("b"))

	//get value in storage r1 and compare with original value
	v1, err := r1.Get([]byte("1"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("a"), v1)

	//get value in storage r2
	v2, err := r2.Get([]byte("1"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("b"), v2)
}

func TestRamStorage_BatchWrite(t *testing.T) {
	rs := NewRamStorage()
	rs.EnableBatch()
	rs.Put([]byte("1"), []byte("a"))
	rs.Put([]byte("1"), []byte("c")) // batch on same key
	rs.Put([]byte("2"), []byte("1234"))

	// Not written to storage before flushing
	_, err := rs.Get([]byte("1"))
	assert.Equal(t, ErrKeyInvalid, err)
	_, err = rs.Get([]byte("2"))
	assert.Equal(t, ErrKeyInvalid, err)

	err = rs.Flush()
	assert.Nil(t, err)

	// Should be able to read the value after flush
	v1, err := rs.Get([]byte("1"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("c"), v1)
	v2, err := rs.Get([]byte("2"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("1234"), v2)

	rs.DisableBatch()
}
