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
package account

import (
	"bytes"
	"crypto/sha256"

	"github.com/btcsuite/btcutil/base58"
	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/golang/protobuf/proto"
)

type TransactionAccount struct {
	address    Address
	pubKeyHash PubKeyHash
}

func NewContractTransactionAccount() *TransactionAccount {
	account := &TransactionAccount{}
	account.pubKeyHash = newContractPubKeyHash()
	account.address = account.pubKeyHash.GenerateAddress()
	return account
}

func NewTransactionAccountByPubKey(pubkey []byte) *TransactionAccount {
	account := &TransactionAccount{}
	account.pubKeyHash = newUserPubKeyHash(pubkey)
	account.address = account.pubKeyHash.GenerateAddress()
	return account
}

func NewContractAccountByPubKeyHash(pubKeyHash PubKeyHash) *TransactionAccount {
	account := &TransactionAccount{}
	account.pubKeyHash = pubKeyHash
	account.address = account.pubKeyHash.GenerateAddress()
	return account
}

func NewTransactionAccountByAddress(address Address) *TransactionAccount {
	account := &TransactionAccount{}
	account.address = address
	account.pubKeyHash, _ = generatePubKeyHashByAddress(address)
	return account
}

//GeneratePubKeyHashByAddress decodes the address to the original public key hash. If unsuccessful, return false
func generatePubKeyHashByAddress(a Address) (PubKeyHash, bool) {
	pubKeyHash := base58.Decode(a.String())

	if len(pubKeyHash) != GetAddressPayloadLength() {
		return nil, false
	}
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	pubKeyHash = pubKeyHash[0 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := Checksum(pubKeyHash)

	if bytes.Compare(actualChecksum, targetChecksum) == 0 {
		return pubKeyHash, true
	}
	return nil, false

}

func turnAddressToPubKeyHash(a Address) PubKeyHash {
	pubKeyHash := base58.Decode(a.String())
	pubKeyHash = pubKeyHash[0 : len(pubKeyHash)-addressChecksumLen]
	return pubKeyHash
}

func (ca *TransactionAccount) IsValid() bool {
	if !ca.GetPubKeyHash().IsValid() {
		return false
	}
	actualChecksum := ca.address.getAddressCheckSum()
	if actualChecksum == nil {
		return false
	}
	targetChecksum := Checksum(ca.pubKeyHash)
	if bytes.Compare(actualChecksum, targetChecksum) == 0 {
		return true
	}
	return false
}

//Checksum finds the checksum of a public key hash
func Checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}

// GetAddressPayloadLength get the payload length
func GetAddressPayloadLength() int {
	// 1byte(version byte) + 20byte(public key hash bytes) + addressChecksumLen
	return 21 + addressChecksumLen
}

func (ca TransactionAccount) GetAddress() Address {
	return ca.address
}

func (ca TransactionAccount) GetPubKeyHash() PubKeyHash {
	return ca.pubKeyHash
}

func (ca *TransactionAccount) ToProto() proto.Message {
	addr := &accountpb.Address{
		Address: ca.address.address,
	}
	return &accountpb.TransactionAccount{
		Address:    addr,
		PubKeyHash: ca.pubKeyHash,
	}
}

func (ca *TransactionAccount) FromProto(pb proto.Message) {
	address := Address{}
	address.FromProto(pb.(*accountpb.TransactionAccount).Address)
	ca.address = address
	ca.pubKeyHash = pb.(*accountpb.TransactionAccount).PubKeyHash
}
