package storage

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

//test get and put methods
func TestRamStorage_GetandPut(t *testing.T) {
	r := NewRamStorage()

	//setup various key scenario
	var keyValuePairs = []struct{
		key 	[]byte
		value 	[]byte
	} {
		{[]byte("a"),[]byte("b")},	//lowercase letter
		{[]byte("A"),[]byte("B")},	//CAPITAL letter
		{[]byte("1"),[]byte("2")},	//number
		{[]byte(""),[]byte("3")},	//empty string
	}

	//for each key value pair. Store it, read it and compare if the value stays unchanged
	for i,kv := range keyValuePairs {
		t.Run(string(i),func(t *testing.T){
			r.Put(kv.key,kv.value)
			v,err := r.Get(kv.key)
			assert.Nil(t, err)
			assert.Equal(t, v,kv.value)
		})
	}

	//Read Invalid Key
	_, err := r.Get([]byte("d"))

	//there should be an error returned
	assert.NotNil(t,err)
}

//test close method
func TestRamStorage_Close(t *testing.T) {
	r := NewRamStorage()

	//write a key-value pair
	r.Put([]byte("1"),[]byte("2"))

	//close the storage
	r.Close()

	//the data should not be accessible
	v, err := r.Get([]byte("1"))

	//there should be an error returned
	assert.NotNil(t,err)
	//there should be not value returned
	assert.Nil(t, v)
}

//check if two storage instances affect each other
func TestRamStorage_IndependantStorage(t *testing.T) {
	r1 := NewRamStorage()
	r2 := NewRamStorage()

	//write a key-value pair in r1 storage should not affect r2
	r1.Put([]byte("1"),[]byte("a"))
	r2.Put([]byte("1"),[]byte("b"))

	//get value in storage r1 and compare with original value
	v1, err:= r1.Get([]byte("1"))
	assert.Nil(t, err)
	assert.Equal(t,[]byte("a"),v1 )

	//get value in storage r2
	v2, err:= r2.Get([]byte("1"))
	assert.Nil(t, err)
	assert.Equal(t,[]byte("b"),v2 )
}
