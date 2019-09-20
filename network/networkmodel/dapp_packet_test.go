package networkmodel

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDappPacket_ExtractDappPacketFromRawBytes(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		retData *DappPacket
		retErr  error
	}{
		{
			name:  "CorrectData",
			input: []byte{0x7e, 0x7e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0xF, 0x10, 0x1, 0x2, 0x3, 0x4, 0x5},
			retData: &DappPacket{
				[]byte{0x7e, 0x7e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0xF, 0x10},
				[]byte{1, 2, 3, 4, 5},
			},
			retErr: nil,
		},
		{
			name:  "Input data longer than required",
			input: []byte{0x7e, 0x7e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0xF, 0x10, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6},
			retData: &DappPacket{
				[]byte{0x7e, 0x7e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0xF, 0x10},
				[]byte{1, 2, 3, 4, 5},
			},
			retErr: nil,
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
			name:    "Not enough bytes for data",
			input:   []byte{0x7e, 0x7e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x00, 0xF, 0x10, 0x1, 0x2, 0x3, 0x4},
			retData: nil,
			retErr:  ErrLengthTooShort,
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
			packet, err := DeserializeIntoDappPacket(tt.input)
			assert.Equal(t, tt.retErr, err)
			assert.Equal(t, tt.retData, packet)
		})
	}
}

func TestDappPacket_containStartingBytes(t *testing.T) {
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
			packet := &DappPacket{header: tt.input}
			assert.Equal(t, tt.expected, packet.containStartingBytes())
		})
	}
}

func TestDappPacket_constructHeader(t *testing.T) {
	bytes := []byte{}
	for i := 0; i < 300; i++ {
		bytes = append(bytes, byte(i))
	}
	assert.Equal(t, []byte{0x7E, 0x7E, 0, 0, 0, 0, 0, 0, 1, 44, 0x0, 0x32, 0x5b}, constructHeader(bytes, false))
}

func TestDappPacket_checkSum(t *testing.T) {
	bytes := []byte{}
	for i := 0; i < 300; i++ {
		bytes = append(bytes, byte(i))
	}
	assert.Equal(t, byte(50), checkSum(bytes))
}
