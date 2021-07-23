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
	"math/big"

	errorValues "github.com/dappley/go-dappley/errors"
)

// Amount implements an unsigned integer type with arbitrary/no upper bound. It is based on big.Int.
type Amount struct {
	big.Int
}

// Validate returns error if a is not a valid amount, otherwise returns nil.
func (a Amount) Validate() error {
	if a.Sign() < 0 {
		return errorValues.ErrAmountUnderflow
	}
	return nil
}

// NewAmount returns a new Amount struct with given an unsigned integer.
func NewAmount(i uint64) *Amount {
	return &Amount{*new(big.Int).SetUint64(i)}
}

// NewAmountFromBigInt returns a new Amount struct with given big.Int representation.
func NewAmountFromBigInt(i *big.Int) (*Amount, error) {
	a := &Amount{*i}
	if err := a.Validate(); nil != err {
		return nil, err
	}
	return a, nil
}

// NewAmountFromString returns a new Amount struct given string representation of an integer in base 10.
func NewAmountFromString(s string) (*Amount, error) {
	i := new(big.Int)
	var success bool
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		_, success = i.SetString(s[2:], 16)
	} else {
		_, success = i.SetString(s, 10)
	}
	if !success {
		return nil, errorValues.ErrAmountInvalidString
	}
	if err := (&Amount{*i}).Validate(); nil != err {
		return nil, err
	}
	return &Amount{*i}, nil
}

func NewAmountFromBytes(b []byte) *Amount {
	return &Amount{*new(big.Int).SetBytes(b)}
}

// Uint64 returns the big.Int representation of the amount.
func (a Amount) BigInt() *big.Int {
	return &a.Int
}

// Add returns sum of a + b
func (a Amount) Add(b *Amount) *Amount {
	return &Amount{*new(big.Int).Add(a.BigInt(), b.BigInt())}
}

// Sub returns difference of a - b. Returns nil with ErrAmountUnderflow if the result is negative (a < b)
func (a Amount) Sub(b *Amount) (*Amount, error) {
	diff := &Amount{*new(big.Int).Sub(a.BigInt(), b.BigInt())}
	if err := diff.Validate(); nil != err {
		return nil, err
	}
	return diff, nil
}

// Mul returns product of a * b
func (a Amount) Mul(b *Amount) *Amount {
	return &Amount{*new(big.Int).Mul(a.BigInt(), b.BigInt())}
}

// Times returns product of a * b where b is uint64
func (a Amount) Times(b uint64) *Amount {
	return a.Mul(NewAmount(b))
}

// Times returns the quotient of a/b where b is uint64
func (a Amount) Div(b uint64) *Amount {
	return a.div(NewAmount(b))
}

// Times returns the quotient of a/b
func (a Amount) div(b *Amount) *Amount {
	return &Amount{*new(big.Int).Div(a.BigInt(), b.BigInt())}
}

// Cmp compares a and b and returns:
//   -1 if a <  b
//    0 if a == b
//   +1 if a >  b
func (a Amount) Cmp(b *Amount) int {
	return a.BigInt().Cmp(b.BigInt())
}

func (a Amount) IsZero() bool {
	return a.Cmp(NewAmount(0)) == 0
}
