package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncodeScInput(t *testing.T) {
	expectedStr := `{"scArgs":"01","scFunction":"getBalance"}`
	assert.Equal(t, expectedStr, EncodeScInput("getBalance","01"))
}

func TestDecodeScInput(t *testing.T) {
	expectedFunction := "getBalance"
	expectedArgs := "01"
	input := `{"scArgs":"01","scFunction":"getBalance"}`
	function,args := DecodeScInput(input)
	assert.Equal(t, expectedFunction, function)
	assert.Equal(t, expectedArgs, args)
}