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

package consensus

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestProofOfWork_Validate(t *testing.T) {
	var cbAddr = string("1JEye2HYHHbjrGv6RPHs9aU3Tt5ktWRVon")
	pow := NewProofOfWork()
	blk := pow.ProduceBlock(cbAddr, "", []byte{})
	//hash :=blk.GetHash()
	assert.True(t,pow.Validate(blk))
	blk.SetNonce(blk.GetNonce()+1)
	assert.False(t, pow.Validate(blk))
}

