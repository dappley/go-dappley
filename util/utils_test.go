package util

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"
)

func TestDecodeScInput(t *testing.T) {
	expectedFunction := "getBalance"
	expectedArgs := []string{"01", "02"}
	input := `{"args":["01","02"],"function":"getBalance"}`
	function, args := DecodeScInput(input)
	assert.Equal(t, expectedFunction, function)
	assert.Equal(t, expectedArgs, args)
}

func TestPrepareArgs(t *testing.T) {
	args := []string{"01", "02"}
	expectedRes := "\"01\",\"02\""
	assert.Equal(t, expectedRes, PrepareArgs(args))
}

func TestReverseSlice(t *testing.T) {
	// len(slice) == 0
	require.Equal(t, []GenericType{}, ReverseSlice([]GenericType{}))
	// len(slice) == 1
	require.Equal(t, []GenericType{1}, ReverseSlice([]GenericType{1}))
	// len(slice) is even
	require.Equal(t, []GenericType{2, 1}, ReverseSlice([]GenericType{1, 2}))
	// len(slice) is odd
	require.Equal(t, []GenericType{3, 2, 1}, ReverseSlice([]GenericType{1, 2, 3}))
}
