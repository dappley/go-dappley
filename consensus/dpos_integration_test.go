// +build integration

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
	"time"

	"github.com/dappley/go-dappley/core/block_producer_info"

	"github.com/dappley/go-dappley/core/account"
	"github.com/stretchr/testify/assert"
)

func TestDpos_Start(t *testing.T) {
	cbAddr := account.NewAddress("dPGZmHd73UpZhrM6uvgnzu49ttbLp4AzU8")
	keystr := "5a66b0fdb69c99935783059bb200e86e97b506ae443a62febd7d0750cd7fac55"

	producer := block_producer_info.NewBlockProducerInfo(cbAddr.String())
	dpos := NewDPOS(producer)
	dpos.SetKey(keystr)

	miners := []string{cbAddr.String()}
	dynasty := NewDynasty(miners, 2, 2)
	dpos.SetDynasty(dynasty)

	dpos.Start()
	//wait for all producer gets a chance to produce
	time.Sleep(time.Second * 2 * 2)
	dpos.Stop()

	assert.Equal(t, len(dpos.notifierCh), 1)
}
