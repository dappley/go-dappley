// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-dappley library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-dappley library.  If not, see <http://www.gnu.org/licenses/>.
//

package core

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"

	"github.com/btcsuite/btcutil/base58"
	"github.com/dappley/go-dappley/crypto/hash"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	logger "github.com/sirupsen/logrus"
)

var (
	ErrInvalidPubKeyHashVersion = errors.New("Invalid Public Key Hash Version ")
)

const versionUser = byte(0x5A)
const versionContract = byte(0x58)
const addressChecksumLen = 4

type KeyPair struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func NewKeyPair() *KeyPair {
	private, public := newKeyPair()
	return &KeyPair{private, public}
}

func (w KeyPair) GenerateAddress(isContract bool) Address {
	return GenerateAddressByPublicKey(w.PublicKey, isContract)
}

func GenerateContractAddress() Address{
	keyPair := NewKeyPair()
	return keyPair.GenerateAddress(true)
}

func GenerateAddressByPublicKey(publicKey []byte, isContract bool) Address {

	var pubKeyHash []byte
	if isContract {
		pubKeyHash, _ = GenerateContractPubKeyHash(publicKey)
	}else{
		pubKeyHash, _ = HashPubKey(publicKey)
	}

	checksum := Checksum(pubKeyHash)
	fullPayload := append(pubKeyHash, checksum...)
	return NewAddress(base58.Encode(fullPayload))
}

//IsHashPubKeyContract
func IsHashPubKeyContract(pubKeyHash []byte) (bool, error){
	if pubKeyHash[0] == versionUser {
		return false, nil
	}

	if pubKeyHash[0] == versionContract {
		return true, nil
	}

	return false, ErrInvalidPubKeyHashVersion
}

func HashPubKey(pubKey []byte) ([]byte, error) {
	pubKeyHash, err := GeneratePublicKeyHash(pubKey)
	if err!=nil {
		return pubKeyHash, err
	}
	pubKeyHash = append([]byte{versionUser}, pubKeyHash...)
	return pubKeyHash,nil
}

func GenerateContractPubKeyHash(pubKey []byte) ([]byte, error) {
	pubKeyHash, err := GeneratePublicKeyHash(pubKey)
	if err != nil {
		return pubKeyHash, err
	}
	pubKeyHash = append([]byte{versionContract}, pubKeyHash...)
	return pubKeyHash,nil
}

func GeneratePublicKeyHash(pubKey []byte) ([]byte, error){
	if pubKey == nil || len(pubKey) < 32 {
		err := errors.New("pubkey not correct")
		return nil, err
	}
	sha := hash.Sha3256(pubKey)
	content := hash.Ripemd160(sha)
	return content, nil
}

func Checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	private, err := secp256k1.NewECDSAPrivateKey()
	if err != nil {
		logger.Panic(err)
	}

	pubKey, _ := secp256k1.FromECDSAPublicKey(&private.PublicKey)
	//remove the uncompressed point at pubKey[0]
	return *private, pubKey[1:]
}
func GetKeyPairByString(privateKey string) *KeyPair {
	private, err := secp256k1.HexToECDSAPrivateKey(privateKey)
	if err != nil {
		logger.Panic(err)
	}

	pubKey, _ := secp256k1.FromECDSAPublicKey(&private.PublicKey)
	return &KeyPair{*private, pubKey[1:]}
}
