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

package common

import (
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

const (
	maxUint64 = ^uint64(0)
)

func TestAmount(t *testing.T) {
	bigInt0 := big.NewInt(0)

	bigInt1 := big.NewInt(1)
	bigIntNeg1 := big.NewInt(-1)

	bigMaxUint64 := &big.Int{}
	bigMaxUint64.SetUint64(maxUint64)

	bigUint128 := &big.Int{}
	bigUint128.Mul(bigMaxUint64, big.NewInt(67280421310721))

	tests := []struct {
		name        string
		input       *big.Int // input
		expected    []byte // expected Big-Endian result
		expectedErr error
	}{
		{"0", bigInt0, []byte{}, nil},
		{"1", bigInt1, []byte{1}, nil},
		{"-1", bigIntNeg1, []byte{}, ErrAmountUnderflow},
		{"max uint64", bigMaxUint64, []byte{255, 255, 255, 255, 255, 255, 255, 255}, nil},
		{"uint64 and above", bigUint128, []byte{61, 48, 241, 156, 209, 0, 255, 255, 194, 207, 14, 99, 46, 255}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u1, err := NewAmountFromBigInt(tt.input)
			if err != nil {
				assert.Equal(t, tt.expectedErr, err)
				return
			}
			b := u1.Bytes()

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
				return
			}

			assert.Nil(t, u1.Validate())
			assert.Equal(t, tt.expected, b)

			u2 := NewAmountFromBytes(b)
			assert.Equal(t, u1.Bytes(), u2.Bytes())
		})
	}
}

func TestAmountOperation(t *testing.T) {
	a := NewAmount(10)
	b := NewAmount(9)
	tmp := NewAmount(uint64(1 << 63))
	assert.Equal(t, tmp.Bytes(), []byte{0x80, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0})

	sumExpect := NewAmount(19)
	sumResult := a.Add(b)
	assert.Equal(t, sumExpect.Bytes(), sumResult.Bytes())

	diffExpect := NewAmount(1)
	diffResult, err := a.Sub(b)
	assert.Nil(t, err)
	assert.Equal(t, diffExpect.Bytes(), diffResult.Bytes())

	result, err := b.Sub(a)
	assert.Equal(t, ErrAmountUnderflow, err)
	assert.Nil(t, result)

	productExpect := NewAmount(90)
	productResult := a.mul(b)
	assert.Equal(t, productExpect.Bytes(), productResult.Bytes())

	productExpect = NewAmount(40)
	productResult = a.Times(4)
	assert.Equal(t, productExpect.Bytes(), productResult.Bytes())

	assert.Equal(t, a.Cmp(b), 1)
	assert.Equal(t, b.Cmp(a), -1)
	assert.Equal(t, a.Cmp(a), 0)
}
