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

package transaction

import (
	"bytes"
	"github.com/dappley/go-dappley/core/account"
)

// Old transaction adapter
type TxAdapter struct {
	*Transaction
}

// Returns TxAdapter
func NewTxAdapter(tx *Transaction) TxAdapter {
	adapter := TxAdapter{tx}
	adapter.fillType()
	return adapter
}

// Returns tx with determinated type
func (adapter *TxAdapter) fillType() {
	tx := adapter.Transaction
	if adapter.Type > TxTypeDefault {
		return
	}
	txType := TxTypeDefault
	if adapter.IsContract() {
		txType = TxTypeContract
	} else if adapter.isCoinbase() {
		txType = TxTypeCoinbase
	} else if adapter.isGasRewardTx() {
		txType = TxTypeGasReward
	} else if adapter.isGasChangeTx() {
		txType = TxTypeGasChange
	} else if adapter.isRewardTx() {
		txType = TxTypeReward
	} else if adapter.isContractGenTx() {
		txType = TxTypeContractGen
	} else {
		txType = TxTypeNormal
	}
	tx.Type = txType
}

func (adapter *TxAdapter) isCoinbase() bool {
	if !adapter.isVinCoinbase() {
		return false
	}

	if len(adapter.Vout) != 1 {
		return false
	}

	if len(adapter.Vin[0].PubKey) == 0 {
		return false
	}

	if bytes.Equal(adapter.Vin[0].PubKey, RewardTxData) {
		return false
	}

	if bytes.Equal(adapter.Vin[0].PubKey, GasRewardData) {
		return false
	}

	if bytes.Equal(adapter.Vin[0].PubKey, GasChangeData) {
		return false
	}

	return true
}

// isRewardTx returns if the transaction is system level smart contract reward
func (adapter *TxAdapter) isRewardTx() bool {

	if !adapter.isVinCoinbase() {
		return false
	}

	if !bytes.Equal(adapter.Vin[0].PubKey, RewardTxData) {
		return false
	}

	return true
}

// isGasRewardTx returns if the transaction is reward to miner after smart contract execution
func (adapter *TxAdapter) isGasRewardTx() bool {

	if !adapter.isVinCoinbase() {
		return false
	}

	if len(adapter.Vout) != 1 {
		return false
	}

	if !bytes.Equal(adapter.Vin[0].PubKey, GasRewardData) {
		return false
	}
	return true
}

// isGasChangeTx returns if the transaction is gas change to from address after smart contract execution
func (adapter *TxAdapter) isGasChangeTx() bool {

	if !adapter.isVinCoinbase() {
		return false
	}

	if len(adapter.Vout) != 1 {
		return false
	}

	if !bytes.Equal(adapter.Vin[0].PubKey, GasChangeData) {
		return false
	}

	return true
}

// IsContract returns true if tx deploys/executes a smart contract; false otherwise
func (adapter *TxAdapter) IsContract() bool {
	if len(adapter.Vout) == 0 {
		return false
	}
	isContract, _ := adapter.Vout[ContractTxouputIndex].PubKeyHash.IsContract()
	return isContract
}

func (adapter *TxAdapter) isVinCoinbase() bool {
	return len(adapter.Vin) == 1 && len(adapter.Vin[0].Txid) == 0 && adapter.Vin[0].Vout == -1
}

// IsFromContract returns true if tx is generated from a contract execution; false otherwise
func (adapter *TxAdapter) isContractGenTx() bool {
	if len(adapter.Vin) == 0 {
		return false
	}

	for _, vin := range adapter.Vin {
		pubKeyHash := account.PubKeyHash(vin.PubKey)
		if isContract, _ := pubKeyHash.IsContract(); !isContract {
			return false
		}
	}
	return true
}