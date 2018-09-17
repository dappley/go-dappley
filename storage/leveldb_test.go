package storage

import (
	"testing"

	"os"

	"github.com/stretchr/testify/assert"
)

const BlockchainDbFile = "../bin/blockchain.db"

func TestLevelDB_DbExists(t *testing.T) {
	setup()
	assert.False(t, DbExists(BlockchainDbFile))
	ldb := OpenDatabase(BlockchainDbFile)
	err := ldb.Put([]byte("a"), []byte("1"))
	assert.Nil(t, err)
	assert.True(t, DbExists(BlockchainDbFile))
	teardown()
}

//put key value pairs into database and read later
func TestLevelDB_PutAndGet(t *testing.T) {

	setup()

	//use default path
	ldb := OpenDatabase(BlockchainDbFile)

	err := ldb.Put([]byte("a"), []byte("1"))
	assert.Nil(t, err)

	err = ldb.Put([]byte("b"), []byte("2"))
	assert.Nil(t, err)

	err = ldb.Put([]byte("c"), []byte("3"))
	assert.Nil(t, err)

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
	err = ldb.Put([]byte("c"), []byte("5"))
	assert.Nil(t, err)

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
	ldb := OpenDatabase("../bin/test.db")

	//put new values in
	err := ldb.Put([]byte("a"), []byte("1"))
	assert.Nil(t, err)

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
