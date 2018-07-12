package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAddress(t *testing.T) {
	key := NewKeyPair()
	address1 := key.GenerateAddress()
	address2 := key.GenerateAddress()

	assert.NotNil(t, address1)
	assert.NotNil(t, address2)

	assert.Equal(t, address1, address2)

}
