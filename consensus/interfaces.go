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

import "github.com/dappley/go-dappley/core"

type NewBlock struct {
	*core.Block
	IsValid bool
}

// Requirement inspects the given block and returns true if it fulfills the requirement
type Requirement func(block *core.Block) bool

var noRequirement = func(block *core.Block) bool { return true }

type BlockProducer interface {
	// Setup tells the producer to give rewards to beneficiaryAddr and return the new block through newBlockCh
	Setup(bc *core.Blockchain, beneficiaryAddr string, newBlockCh chan *NewBlock)

	SetPrivateKey(key string)

	// Beneficiary returns the address which receives rewards
	Beneficiary() string

	// SetRequirement defines the requirement that a new block must fulfill
	SetRequirement(requirement Requirement)

	Start()

	Stop()
}
