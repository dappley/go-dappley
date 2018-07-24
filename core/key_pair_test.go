package core

import (
	"testing"

	"github.com/dappley/go-dappley/crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAddress(t *testing.T) {
	key1 := NewKeyPair()
	key2 := NewKeyPair()
	address1 := key1.GenerateAddress()
	address2 := key1.GenerateAddress()

	address3 := key2.GenerateAddress()

	assert.NotNil(t, address1)
	assert.NotNil(t, address2)
	assert.NotNil(t, address3)

	assert.Equal(t, address1, address2)
	assert.NotEqual(t, address1, address3)

}

func TestNewKeyPair(t *testing.T) {
	key1 := NewKeyPair()

	pubKey := append(key1.PrivateKey.PublicKey.X.Bytes(), key1.PrivateKey.PublicKey.Y.Bytes()...)
	assert.NotNil(t, key1)
	assert.NotNil(t, key1.PrivateKey)
	assert.NotNil(t, key1.PublicKey)

	assert.Equal(t, pubKey, key1.PublicKey)

}

func TestHashPubKey(t *testing.T) {
	key1 := NewKeyPair()
	sha := hash.Sha3256(key1.PublicKey)
	expect := hash.Ripemd160(sha)
	content := HashPubKey(key1.PublicKey)
	assert.Equal(t, expect, content)
}
