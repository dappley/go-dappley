package core

import (
	"bytes"
	"fmt"

	"github.com/dappley/go-dappley/util"
)

type Address struct {
	Address string
}


func NewAddress(addressString string) Address {
	address := Address{}
	address.Address = addressString
	return address
}

func (a Address) ValidateAddress() bool {
	fmt.Println(a.Address)
	pubKeyHash := util.Base58Decode([]byte(a.Address))
	fmt.Println(pubKeyHash)
	if len(pubKeyHash) < addressChecksumLen {
		return false
	}
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}
