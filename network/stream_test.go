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

func TestStream_decode(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		retData []byte
		retErr  error
	}{
		{
			name:    "CorrectData",
			input:   []byte{0x7e, 0x7e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x1, 0x1, 0x2, 0x3, 0x4, 0x5},
			retData: []byte{1, 2, 3, 4, 5},
			retErr:  nil,
		},
		{
			name:    "IncorrectStartingByte",
			input:   []byte{0x7E, 0x55, 0x44, 0x7F, 0x7F, 0x00, 0x44, 0x7F, 0x7F, 0x00, 0x44, 0x7F, 0x7F, 0x00},
			retData: nil,
			retErr:  ErrInvalidMessageFormat,
		},
		{
			name:    "Not enough bytes for header",
			input:   []byte{0x7E, 0x7E, 0x55, 0x44, 0x7F, 0x7F, 0x01},
			retData: nil,
			retErr:  ErrLengthTooShort,
		},
		{
			name:    "Fragmented data",
			input:   []byte{0x7e, 0x7e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x1, 0x1, 0x2, 0x3, 0x4},
			retData: nil,
			retErr:  ErrFragmentedData,
		},
		{
			name:    "Incorrect checksum",
			input:   []byte{0x7e, 0x7e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x3, 0x1, 0x2, 0x3, 0x4, 0x5},
			retData: nil,
			retErr:  ErrCheckSumIncorrect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret, err := decodeMessage(tt.input)
			assert.Equal(t, tt.retData, ret)
			assert.Equal(t, tt.retErr, err)
		})
	}
}

func TestStream_containStartingBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "containAtBeginning",
			input:    []byte{0x7E, 0x7E, 0x7F},
			expected: true,
		},
		{
			name:     "containAtTheEnd",
			input:    []byte{0x7F, 0x7E, 0x7E},
			expected: false,
		},
		{
			name:     "containInTheMiddle",
			input:    []byte{0x7F, 0x7E, 0x7E, 0x7F},
			expected: false,
		},
		{
			name:     "NotContaining",
			input:    []byte{0x7F, 0x7F},
			expected: false,
		},
		{
			name:     "EmptyInput",
			input:    []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, containStartingBytes(tt.input))
		})
	}
}

func TestStream_constructHeader(t *testing.T) {
	bytes := []byte{}
	for i := 0; i < 300; i++ {
		bytes = append(bytes, byte(i))
	}
	assert.Equal(t, []byte{0x7E, 0x7E, 0, 0, 0, 0, 0, 0, 1, 44, 0x29}, constructHeader(bytes))
}

func TestStream_checkSum(t *testing.T) {
	bytes := []byte{}
	for i := 0; i < 300; i++ {
		bytes = append(bytes, byte(i))
	}
	assert.Equal(t, byte(50), checkSum(bytes))
}

func TestStream_Send(t *testing.T) {
	s := &Stream{
		"",
		nil,
		nil,
		0,
		[]byte{},
		make(chan []byte, 100),
		make(chan []byte, highPriorityChLength),
		make(chan []byte, normalPriorityChLength),
		make(chan bool, WriteChTotalLength),
		make(chan bool, 1), //two channels to stop
		make(chan bool, 1),
	}
	data1 := []byte("data1")
	data2 := []byte("data2")
	s.Send(data1, NormalPriorityCommand)
	s.Send(data2, HighPriorityCommand)
	assert.Equal(t, 2, len(s.msgNotifyCh))
	assert.Equal(t, 1, len(s.highPriorityWriteCh))
	assert.Equal(t, 1, len(s.normalPriorityWriteCh))

	select {
	case receivedData := <-s.highPriorityWriteCh:
		assert.Equal(t, data2, receivedData)
	default:
		assert.Error(t, nil, "No data in high priority channel")
	}

	select {
	case receivedData := <-s.normalPriorityWriteCh:
		assert.Equal(t, data1, receivedData)
	default:
		assert.Error(t, nil, "No data in normal priority channel")
	}

}
