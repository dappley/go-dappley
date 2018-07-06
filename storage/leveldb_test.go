package storage

import (
	"testing"

	"os"

	"github.com/stretchr/testify/assert"
)

const BlockchainDbFile = "../bin/blockchain.DB"

//put key value pairs into database and read later
func TestLevelDB_PutAndGet(t *testing.T) {

	setup()

	//use default path
	ldb := OpenDatabase(BlockchainDbFile)

	ldb.Put([]byte("a"), []byte("1"))

	ldb.Put([]byte("b"), []byte("2"))

	ldb.Put([]byte("c"), []byte("3"))

	val, err := ldb.Get([]byte("a"))
	assert.Nil(t, err)
	assert.Equal(t, val, []byte("1"))

	val, err = ldb.Get([]byte("b"))
	assert.Nil(t, err)
	assert.Equal(t, val, []byte("2"))

	val, err = ldb.Get([]byte("c"))
	assert.Nil(t, err)
	assert.Equal(t, val, []byte("3"))

	//modify value and compare
	ldb.Put([]byte("c"), []byte("5"))

	val, err = ldb.Get([]byte("c"))
	assert.Nil(t, err)
	assert.Equal(t, val, []byte("5"))

	err = ldb.Close()
	assert.Nil(t, err)

	teardown()
}

//Test if database access after closing will result in error
func TestLevelDB_Close(t *testing.T) {

	setup()
	//create new database
	ldb := OpenDatabase("../bin/test.DB")

	//put new values in
	ldb.Put([]byte("a"), []byte("1"))

	//Currently we should be able to read it
	val, err := ldb.Get([]byte("a"))
	assert.Nil(t, err)
	assert.Equal(t, val, []byte("1"))

	//Close the database
	err = ldb.Close()
	assert.Nil(t, err)

	teardown()
}

func setup() {
	cleanUpDatabase()
}

func teardown() {
	cleanUpDatabase()
}

func cleanUpDatabase() {
	os.RemoveAll(BlockchainDbFile)
}
