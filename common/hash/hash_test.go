package hash

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHash_Equals(t *testing.T) {
	hash := Hash([]byte("test"))
	hash2 := Hash([]byte("test"))
	assert.True(t, hash.Equals(hash2))

	hash3 := Hash([]byte("test1"))
	assert.False(t, hash.Equals(hash3))
}
