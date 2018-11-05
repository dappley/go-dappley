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

package util

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	logger "github.com/sirupsen/logrus"
)
const keyFunction = "scFunction"
const keyArs = "scArgs"

// IntToHex converts an int64 to a byte array
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		logger.Panic(err)
	}

	return buff.Bytes()
}

// UintToHex converts an uint64 to a byte array
// TODO  Same as IntToHex, need refactor
func UintToHex(num uint64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		logger.Panic(err)
	}

	return buff.Bytes()
}

func EncodeScInput(function, args string) string{
	input := map[string]string{keyFunction:function, keyArs:args}
	encodedStr, _ := json.Marshal(input)
	return string(encodedStr)
}

func DecodeScInput(s string) (function, args string){
	var input map[string]string
	json.Unmarshal([]byte(s),&input)
	return input[keyFunction],input[keyArs]
}