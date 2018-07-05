package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"os"
)

//put key value pairs into database and read later
func TestLevelDB_PutAndGet(t *testing.T){

	setup()

	//use default path
	ldb, err := OpenDatabase("")
	assert.Nil(t, err)

	err = ldb.Put([]byte("a"),[]byte("1"))
	assert.Nil(t, err)
	err = ldb.Put([]byte("b"),[]byte("2"))
	assert.Nil(t, err)
	err = ldb.Put([]byte("c"),[]byte("3"))
	assert.Nil(t, err)

	val, err := ldb.Get([]byte("a"))
	assert.Nil(t, err)
	assert.Equal(t,val,[]byte("1"))

	val, err = ldb.Get([]byte("b"))
	assert.Nil(t, err)
	assert.Equal(t,val,[]byte("2"))

	val, err = ldb.Get([]byte("c"))
	assert.Nil(t, err)
	assert.Equal(t,val,[]byte("3"))

	//modify value and compare
	err = ldb.Put([]byte("c"),[]byte("5"))
	assert.Nil(t, err)
	val, err = ldb.Get([]byte("c"))
	assert.Nil(t, err)
	assert.Equal(t,val,[]byte("5"))


	err = ldb.Close()
	assert.Nil(t,err)

	teardown()
}

//Test if database access after closing will result in error
func TestLevelDB_Close(t *testing.T) {

	setup()
	//create new database
	ldb, err := OpenDatabase("../bin/test.DB")
	assert.Nil(t, err)

	//put new values in
	err = ldb.Put([]byte("a"),[]byte("1"))
	assert.Nil(t, err)

	//Currently we should be able to read it
	val, err := ldb.Get([]byte("a"))
	assert.Nil(t, err)
	assert.Equal(t,val,[]byte("1"))

	//Close the database
	err = ldb.Close()
	assert.Nil(t,err)

	//reading should return error
	val, err = ldb.Get([]byte("a"))
	assert.NotNil(t, err)
	assert.Nil(t,val)

	//Writing should return error
	err = ldb.Put([]byte("b"),[]byte("2"))
	assert.NotNil(t, err)

	teardown()
}

func setup() {
	cleanUpDatabase()
}

func teardown() {
	cleanUpDatabase()
}

func cleanUpDatabase() {
	os.RemoveAll(DefaultDbFile)
}