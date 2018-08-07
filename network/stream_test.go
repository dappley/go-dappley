package network

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestStream_decode(t *testing.T){
	//mock an integer array
	data := []byte{1,2,3,4}
	//encode data
	encodeData := encodeMessage(data)
	//decode data
	decodeData, err:= decodeMessage(encodeData)
	//there should be no error
	assert.Nil(t,err)
	//the data should be the same as before
	assert.ElementsMatch(t, data, decodeData)

}

func TestStream_decodeFragmentedMsg(t *testing.T){
	//mock an integer array that does not follow the format of encoded data
	data := []byte{1,2,3,4}
	//decode
	decodeData, err:= decodeMessage(data)
	//there should be an error returned
	assert.Equal(t,ErrInvalidMessageFormat,err)
	assert.Nil(t, decodeData)
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