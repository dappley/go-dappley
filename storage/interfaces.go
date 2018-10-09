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
	"bytes"
	"errors"
)

var (
	ErrKeyInvalid = errors.New("Key is invalid")
)

type Storage interface {
	Close() error

	Get(key []byte) ([]byte, error)

	Put(key []byte, val []byte) error

	// EnableBatch enable batch write.
	EnableBatch()

	// DisableBatch disable batch write.
	DisableBatch()

	// Flush write and flush pending batch write.
	Flush() error
}

type FileStorage interface {
	ReadFromFile() ([]byte, error)
	SaveToFile(buffer bytes.Buffer)
}