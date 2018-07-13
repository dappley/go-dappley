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
