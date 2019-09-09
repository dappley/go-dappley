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

package transactionbase

import (
	"testing"

	transactionbasepb "github.com/dappley/go-dappley/core/transactionbase/pb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestTXInput_Proto(t *testing.T) {
	vin := TXInput{
		[]byte("txid"),
		1,
		[]byte("signature"),
		[]byte("PubKey"),
	}

	pb := vin.ToProto()
	var i interface{} = pb
	_, correct := i.(proto.Message)
	assert.Equal(t, true, correct)
	mpb, err := proto.Marshal(pb)
	assert.Nil(t, err)

	newpb := &transactionbasepb.TXInput{}
	err = proto.Unmarshal(mpb, newpb)
	assert.Nil(t, err)

	vin2 := TXInput{}
	vin2.FromProto(newpb)

	assert.Equal(t, vin, vin2)
}
