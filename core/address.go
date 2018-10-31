// Copyright (C) 2018 go-dappley authors
//
// This file is part of the go-dappley library.
//
// the go-dappley library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either pubKeyHash 3 of the License, or
// (at your option) any later pubKeyHash.
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
	"bytes"
	"errors"

	"github.com/btcsuite/btcutil/base58"
)

var (
	ErrInvalidAddress = errors.New("Invalid Address")
)

type Address struct {
	Address string
}

func NewAddress(addressString string) Address {
	address := Address{}
	address.Address = addressString
	return address
}

//String returns the address in string type
func (a Address) String() string {
	return a.Address
}

//isContract checks if an address is a Contract address
func (a Address) IsContract() (bool, error) {
	pubKeyHash, ok := a.GetPubKeyHash()
	if !ok {
		return false, ErrInvalidAddress
	}
	pkh := PubKeyHash{pubKeyHash}
	return pkh.IsContract()
}

//ValidateAddress checks if an address is valid
func (a Address) ValidateAddress() bool {
	_, ok := a.GetPubKeyHash()
	return ok
}

//GetPubKeyHash decodes the address to the original public key hash. If unsuccessful, return false
func (a Address) GetPubKeyHash() ([]byte, bool) {
	pubKeyHash := base58.Decode(a.String())

	if len(pubKeyHash) < addressChecksumLen {
		return nil, false
	}
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	pubKeyHash = pubKeyHash[0 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := Checksum(pubKeyHash)

	if bytes.Compare(actualChecksum, targetChecksum) == 0 {
		return pubKeyHash, true
	} else {
		return nil, false
	}
}
