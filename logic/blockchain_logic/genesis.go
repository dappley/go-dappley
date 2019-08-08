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
package blockchain_logic

import (
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/block"
	"github.com/dappley/go-dappley/core/transaction"
	"github.com/dappley/go-dappley/core/transaction_base"
	"github.com/dappley/go-dappley/logic/block_logic"
)

const genesisCoinbaseData = "Hello world"

func NewGenesisBlock(address account.Address, subsidy *common.Amount) *block.Block {

	txin := transaction_base.TXInput{nil, -1, nil, []byte(genesisCoinbaseData)}
	txout := transaction_base.NewTXOutput(subsidy, address)
	txs := []*transaction.Transaction{}
	tx := transaction.Transaction{nil, []transaction_base.TXInput{txin}, []transaction_base.TXOutput{*txout}, common.NewAmount(0), common.NewAmount(0), common.NewAmount(0)}
	tx.ID = tx.Hash()
	txs = append(txs, &tx)

	blk := block.NewBlockWithRawInfo(
		nil,
		nil,
		0,
		1532392928, //July 23,2018 17:42 PST
		0,
		txs)

	blk.SetHash(block_logic.CalculateHash(blk))
	return blk
}
