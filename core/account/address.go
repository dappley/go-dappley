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

package account

import (
	"errors"

	"github.com/btcsuite/btcutil/base58"
	accountpb "github.com/dappley/go-dappley/core/account/pb"
	"github.com/golang/protobuf/proto"
)

var (
	ErrInvalidAddress = errors.New("invalid address")
)

type Address struct {
	address string
}

func NewAddress(addressString string) Address {
	return Address{addressString}
}
func (a Address) decode() []byte {
	pubKeyHash := base58.Decode(a.String())
	if len(pubKeyHash) != GetAddressPayloadLength() {
		return nil
	}
	return pubKeyHash
}

func (a Address) getAddressCheckSum() []byte {
	addresshash := a.decode()
	if addresshash == nil {
		return nil
	}
	actualChecksum := addresshash[len(addresshash)-addressChecksumLen:]
	return actualChecksum
}

//String returns the address in string type
func (a Address) String() string {
	return a.address
}

//ToProto converts Address object to protobuf message
func (a *Address) ToProto() proto.Message {
	return &accountpb.Address{
		Address: a.address,
	}
}

//FromProto converts protobuf message to Address object
func (a *Address) FromProto(pb proto.Message) {
	a.address = pb.(*accountpb.Address).Address
}
