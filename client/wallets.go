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

package client

import (
	"errors"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"encoding/gob"
	"bytes"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1/bitelliptic"
	logger "github.com/sirupsen/logrus"
)


const WalletFile = "../bin/wallets.dat"

type WalletManager struct {
	Wallets []Wallet
}

func LoadWalletFromFile(filePath string) (*WalletManager, error) {

	fileContent, err := storage.GetFileConnection(filePath)

	wm := &WalletManager{}
	if err != nil {
		wm.SaveWalletToFile(filePath)
		fileContent, err = storage.GetFileConnection(filePath)
	}
	var wallets WalletManager

	gob.Register(bitelliptic.S256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		logger.Error("WalletManager: Load wallets failed!")
		logger.Error(err)
	}

	wm.Wallets = wallets.Wallets

	return wm, nil
}

// SaveToFile saves wallets to a file
func (wm WalletManager) SaveWalletToFile(filePath string) {
	var content bytes.Buffer

	gob.Register(bitelliptic.S256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(wm)
	if err != nil {
		logger.Error("WalletManager: save wallets to file failed!")
		logger.Error(err)
	}
	storage.SaveToFile(filePath, content)

}

func (wm *WalletManager) AddWallet(wallet Wallet){
	wm.Wallets = append(wm.Wallets, wallet)
}

func (wm *WalletManager) DeleteWallet(key *core.KeyPair) error {
	for i, value := range wm.Wallets {
		if value.Key == key {
			wm.Wallets = append(wm.Wallets[:i], wm.Wallets[i+1:]...)
			return nil
		}
	}

	return errors.New("wallet is not exist")

}

func (wm *WalletManager) DeleteWallets() error {
	if len(wm.Wallets) == 0 {
		return errors.New("no wallet yet")
	}
	wm.Wallets = wm.Wallets[:0]
	return nil
}

func (wm *WalletManager) GetAddresses() []core.Address {
	var addresses []core.Address

	for _, address := range wm.Wallets {
		addresses = append(addresses, address.GetAddresses()...)
	}

	return addresses
}

func (wm WalletManager) GetKeyPairByAddress(address core.Address) core.KeyPair {
	for _, value := range wm.Wallets {

		if value.ContainAddress(address) {
			return *value.Key
		}
	}
	return core.KeyPair{}

}

func (wm WalletManager) GetWalletByAddress(address core.Address) Wallet {
	for _, wallet := range wm.Wallets {
		if wallet.ContainAddress(address) {
			return wallet
		}
	}
	return Wallet{}
}


