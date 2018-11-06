package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDecodeScInput(t *testing.T) {
	expectedFunction := "getBalance"
	expectedArgs := []string{"01","02"}
	input := `{"args":["01","02"],"function":"getBalance"}`
	function,args := DecodeScInput(input)
	assert.Equal(t, expectedFunction, function)
	assert.Equal(t, expectedArgs, args)
}

func TestPrepareArgs(t *testing.T) {
	args := []string{"01","02"}
	expectedRes := "\"01\",\"02\""
	assert.Equal(t, expectedRes, PrepareArgs(args))
}