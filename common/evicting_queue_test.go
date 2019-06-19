package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvictingQueue_Push(t *testing.T) {
	eq := NewEvictingQueue(3)
	var pushAndPeekTests = []struct {
		in  interface{}
		out interface{}
	}{
		{1, 1},
		{2, 1},
		{3, 1},
	}
	for _, v := range pushAndPeekTests {
		eq.Push(v.in)
		require.Equal(t, v.out, eq.Peek())
	}
	require.False(t, eq.IsEmpty())
	require.Equal(t, []Element{1, 2, 3}, eq.toArray())
	// capacity constraint
	require.Equal(t, []Element{2, 3, 4}, eq.Push(4).toArray())
}

func TestEvictingQueue_Peek(t *testing.T) {
	eq := NewEvictingQueue(3)
	require.Nil(t, eq.Peek())
	require.Equal(t, 1, eq.Push(1).Peek())
}

func TestEvictingQueue_IsEmpty(t *testing.T) {
	eq := NewEvictingQueue(3)
	require.True(t, eq.IsEmpty())
	require.False(t, eq.Push(1).IsEmpty())
	eq.Pop()
	require.True(t, eq.IsEmpty())
}

func TestEvictingQueue_Len(t *testing.T) {
	eq := NewEvictingQueue(3)
	require.Equal(t, 0, eq.Len())
	require.Equal(t, 1, eq.Push(1).Len())
	eq.Pop()
	require.Equal(t, 0, eq.Len())
}

func TestEvictingQueue_ForEach(t *testing.T) {
	eq := NewEvictingQueue(5)
	for i := 0; i < 5; i++ {
		eq.Push(i)
	}
	i := 0
	eq.ForEach(func(element Element) {
		assert.Equal(t, i, element)
		i++
	})
}

func TestEvictingQueue_Pop(t *testing.T) {
	eq := NewEvictingQueue(3).Push(1).Push(2).Push(3)
	for _, v := range []interface{}{1, 2, 3} {
		require.Equal(t, v, eq.Peek())
		require.Equal(t, v, eq.Pop())
	}
	require.Nil(t, eq.Pop())
	require.True(t, eq.IsEmpty())
}

func TestEvictingQueue_String(t *testing.T) {
	require.Equal(t, "[1 2 3]",
		NewEvictingQueue(3).
			Push(1).
			Push(2).
			Push(3).
			String())
}

func TestEvictingQueue_MarshalJSON(t *testing.T) {
	bytes, err := json.Marshal(
		NewEvictingQueue(3).
			Push(1).
			Push(2).
			Push(3))
	require.Nil(t, err)
	require.Equal(t, "[1,2,3]", string(bytes))
}
