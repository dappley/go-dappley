package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type GenericSlice []interface{}

func TestReverseSlice(t *testing.T) {
	// len(slice) == 0
	require.Equal(t, GenericSlice{}, ReverseSlice(GenericSlice{}))
	// len(slice) == 1
	require.Equal(t, GenericSlice{1}, ReverseSlice(GenericSlice{1}))
	// len(slice) is even
	require.Equal(t, GenericSlice{2, 1}, ReverseSlice(GenericSlice{1, 2}))
	// len(slice) is odd
	require.Equal(t, GenericSlice{3, 2, 1}, ReverseSlice(GenericSlice{1, 2, 3}))
	// type assertion
	require.Equal(t, []int{3, 2, 1}, ReverseSlice([]int{1, 2, 3}).([]int))
}
