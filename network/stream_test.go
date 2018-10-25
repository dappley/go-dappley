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

package network

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestStream_decode(t *testing.T){
	tests := []struct{
		name 		string
		input 		[]byte
		retData 	[]byte
		retErr 		error
	}{
		{
			name: 		"CorrectData",
			input: 		[]byte{0x7E,0x7E,0x55,0x44,0x7F,0x7F,0x00},
			retData:   	[]byte{0x55,0x44},
			retErr: 	nil,
		},
		{
			name: 		"IncorrectStartingByte",
			input: 		[]byte{0x7E,0x55,0x44,0x7F,0x7F,0x00},
			retData:   	nil,
			retErr: 	ErrInvalidMessageFormat,
		},
		{
			name: 		"IncorrectEndingingByte",
			input: 		[]byte{0x7E,0x7E,0x55,0x44,0x7F,0x7F,0x01},
			retData:   	nil,
			retErr: 	ErrInvalidMessageFormat,
		},
		{
			name: 		"IncorrectData",
			input: 		[]byte{0x55,0x44},
			retData:   	nil,
			retErr: 	ErrInvalidMessageFormat,
		},
	}

	for _,tt := range tests{
		t.Run(tt.name,func(t *testing.T){
			ret,err := decodeMessage(tt.input)
			assert.Equal(t,tt.retData,ret)
			assert.Equal(t,tt.retErr,err)
		})
	}
}

func TestStream_containStartingBytes(t *testing.T){
	tests := []struct{
		name 		string
		input 		[]byte
		expected 	bool
	}{
		{
			name:		"containAtBeginning",
			input:		[]byte{0x7E,0x7E,0x7F},
			expected:	true,
		},
		{
			name:		"containAtTheEnd",
			input:		[]byte{0x7F,0x7E,0x7E},
			expected:	false,
		},
		{
			name:		"containInTheMiddle",
			input:		[]byte{0x7F,0x7E,0x7E,0x7F},
			expected:	false,
		},
		{
			name:		"NotContaining",
			input:		[]byte{0x7F,0x7F},
			expected:	false,
		},
		{
			name:		"EmptyInput",
			input:		[]byte{},
			expected:	false,
		},
	}

	for _,tt := range tests{
		t.Run(tt.name,func(t *testing.T){
			assert.Equal(t,tt.expected,containStartingBytes(tt.input))
		})
	}
}

func TestStream_containEndingBytes(t *testing.T){
	tests := []struct{
		name 		string
		input 		[]byte
		expected 	bool
	}{
		{
			name:		"containAtBeginning",
			input:		[]byte{0x7F,0x7F,0x00,0x00},
			expected:	false,
		},
		{
			name:		"containAtTheEnd",
			input:		[]byte{0x33,0x33,0x7F,0x7F,0x00,},
			expected:	true,
		},
		{
			name:		"containInTheMiddle",
			input:		[]byte{0x33,0x33,0x7F,0x7F,0x00,0x33,0x33},
			expected:	false,
		},
		{
			name:		"NotContaining",
			input:		[]byte{0xDF,0x23},
			expected:	false,
		},
		{
			name:		"EmptyInput",
			input:		[]byte{},
			expected:	false,
		},
	}

	for _,tt := range tests{
		t.Run(tt.name,func(t *testing.T){
			assert.Equal(t,tt.expected,containEndingBytes(tt.input))
		})
	}
}