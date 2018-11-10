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
const quotationMark = "\""

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

type argStruct struct{
	Function string `json:"function"`
	Args 	 []string `json:"args"`
}

func DecodeScInput(s string) (function string, args []string){
	var input argStruct
	err := json.Unmarshal([]byte(s),&input)
	if err!= nil{
		logger.WithFields(logger.Fields{
			"input"				: s,
			"decoded function"	: input.Function,
			"decoded args" 		: input.Args,
		}).Warn("Unable to decode smart contract input!")
	}
	return input.Function, input.Args
}

func PrepareArgs(args []string) string{
	totalArgs := ""
	for i,arg := range args{
		if i==0{
			totalArgs += quoteArg(arg)
		}else{
			totalArgs += "," + quoteArg(arg)
		}
	}
	return totalArgs
}

func quoteArg(arg string) string{
	return quotationMark + arg + quotationMark
}