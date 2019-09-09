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
	"errors"
	"os"
	"sync"

	logger "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	ErrLevelDbNotAbleToOpenFile = errors.New("leveldb failed to open file")
)

type LevelDB struct {
	db    *leveldb.DB
	batch *leveldb.Batch
	batchPool *sync.Pool
}

//Create a new database instance
func OpenDatabase(dbFilePath string) *LevelDB {

	fp := dbFilePath

	db1, err := leveldb.OpenFile(fp, nil)
	if err != nil {
		logger.Panic(ErrLevelDbNotAbleToOpenFile)
	}

	return &LevelDB{
		db:    db1,
		batch: nil,
		batchPool:&sync.Pool{New: func() interface{} {
			return &leveldb.Batch{}
		}},
	}
}

func (ldb *LevelDB) Close() error {
	logger.Info("LevelDB: is closing the database connection.")
	return ldb.db.Close()
}

func (ldb *LevelDB) Get(key []byte) ([]byte, error) {
	val, err := ldb.db.Get(key, nil)
	if err != nil && err == leveldb.ErrNotFound {
		return nil, ErrKeyInvalid
	}
	return val, err
}

func (ldb *LevelDB) Put(key []byte, val []byte) error {
	logger.WithFields(logger.Fields{
		"key length"  : len(key),
		"data length" : len(val),
	}).Debug("Leveldb put")

	if ldb.batch != nil {
		ldb.batch.Put(key, val)
		return nil
	}
	err := ldb.db.Put(key, val, nil)
	if err != nil {
		logger.Error(err)
	}
	return err
}

func (ldb *LevelDB) Del(key []byte) error {
	return ldb.db.Delete(key, nil)
}

func (ldb *LevelDB) EnableBatch() {
	//ldb.batch = new(leveldb.Batch)
	ldb.batch = ldb.batchPool.Get().(*leveldb.Batch)
}

func (ldb *LevelDB) Flush() error {
	if ldb.batch != nil {
		logger.Debugf("LevelDB: is flushing %d operations to storage.", ldb.batch.Len())
		return ldb.db.Write(ldb.batch, nil)
	}
	return nil
}

func (ldb *LevelDB) DisableBatch() {
	ldb.batchPool.Put(ldb.batch)
	ldb.batch = nil
}

func DbExists(dbFilePath string) bool {
	if _, err := os.Stat(dbFilePath); os.IsNotExist(err) {
		return false
	}
	return true
}
