package storage

import (
	"os"
	"testing"

	errorValues "github.com/dappley/go-dappley/errors"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const testDbFile = "../bin/testleveldb.db"

func TestMain(m *testing.M) {
	logger.SetLevel(logger.WarnLevel)
	setup()
	retCode := m.Run()
	teardown()
	os.Exit(retCode)
}

func TestLevelDB_DbExists(t *testing.T) {
	assert.False(t, DbExists(testDbFile))
	ldb := OpenDatabase(testDbFile)
	defer ldb.Close()
	err := ldb.Put([]byte("a"), []byte("1"))
	assert.Nil(t, err)
	assert.True(t, DbExists(testDbFile))
}

//put key value pairs into database and read later
func TestLevelDB_PutAndGet(t *testing.T) {
	//use default path
	ldb := OpenDatabase(testDbFile)
	defer ldb.Close()

	err := ldb.Put([]byte("a"), []byte("1"))
	assert.Nil(t, err)

	err = ldb.Put([]byte("b"), []byte("2"))
	assert.Nil(t, err)

	err = ldb.Put([]byte("c"), []byte("3"))
	assert.Nil(t, err)

	val, err := ldb.Get([]byte("a"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("1"), val)

	val, err = ldb.Get([]byte("b"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("2"), val)

	val, err = ldb.Get([]byte("c"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("3"), val)

	//modify value and compare
	err = ldb.Put([]byte("c"), []byte("5"))
	assert.Nil(t, err)

	val, err = ldb.Get([]byte("c"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("5"), val)

	err = ldb.Close()
	assert.Nil(t, err)
}

//Test if database access after closing will result in error
func TestLevelDB_Close(t *testing.T) {
	//create new database
	ldb := OpenDatabase(testDbFile)

	//put new values in
	err := ldb.Put([]byte("a"), []byte("1"))
	assert.Nil(t, err)

	//Currently we should be able to read it
	val, err := ldb.Get([]byte("a"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("1"), val)

	//Close the database
	err = ldb.Close()
	assert.Nil(t, err)
}

func TestLevelDB_BatchWrite(t *testing.T) {
	ldb := OpenDatabase(testDbFile)
	defer ldb.Close()

	ldb.EnableBatch()
	ldb.Put([]byte("1"), []byte("a"))
	ldb.Put([]byte("1"), []byte("c")) // batch on same key
	ldb.Put([]byte("2"), []byte("1234"))

	// Not written to storage before flushing
	_, err := ldb.Get([]byte("1"))
	assert.Equal(t, errorValues.InvalidKey, err)
	_, err = ldb.Get([]byte("2"))
	assert.Equal(t, errorValues.InvalidKey, err)

	err = ldb.Flush()
	assert.Nil(t, err)

	// Should be able to read the value after flush
	v1, err := ldb.Get([]byte("1"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("c"), v1)
	v2, err := ldb.Get([]byte("2"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("1234"), v2)

	ldb.DisableBatch()
}

func setup() {
	cleanUpDatabase()
}

func teardown() {
	cleanUpDatabase()
}

func cleanUpDatabase() {
	os.RemoveAll(testDbFile)
}
