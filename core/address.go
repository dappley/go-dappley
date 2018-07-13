package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"log"

	"github.com/dappley/go-dappley/util"
	"github.com/dappley/go-dappley/crypto/hash"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
)

const version = byte(0x00)
const addressChecksumLen = 4

type Address struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func NewAddress() *Address {
	private, public := newKeyPair()
	return &Address{private, public}
}

func (w Address) GetAddress() []byte {
	pubKeyHash := HashPubKey(w.PublicKey)

	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := util.Base58Encode(fullPayload)

	return address
}

func HashPubKey(pubKey []byte) []byte {

	sha := hash.Sha3256(pubKey)
	content := hash.Ripemd160(sha)

	//publicSHA256 := sha256.Sum256(pubKey)
	//
	//RIPEMD160Hasher := ripemd160.New()
	//_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	//if err != nil {
	//	log.Panic(err)
	//}
	//publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return content
}

func ValidateAddress(address string) bool {
	pubKeyHash := util.Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {

	private, err := secp256k1.NewECDSAPrivateKey()
	//curve := elliptic.P256()
	//private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pubKey
}
