package core

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"

	"github.com/dappley/go-dappley/crypto/hash"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/util"
)

const version = byte(0x00)
const addressChecksumLen = 4

type KeyPair struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func NewKeyPair() *KeyPair {
	private, public := newKeyPair()
	return &KeyPair{private, public}
}

func (w KeyPair) GenerateAddress() Address {
	pubKeyHash, _ := HashPubKey(w.PublicKey)

	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := util.Base58Encode(fullPayload)

	return NewAddress(fmt.Sprintf("%s", address))
}

func HashPubKey(pubKey []byte) ([]byte, error) {
	if pubKey == nil || len(pubKey) < 32 {
		err := errors.New("pubkey not correct")
		return nil, err
	}
	sha := hash.Sha3256(pubKey)
	content := hash.Ripemd160(sha)
	return content, nil

}

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	private, err := secp256k1.NewECDSAPrivateKey()
	if err != nil {
		log.Panic(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pubKey
}
