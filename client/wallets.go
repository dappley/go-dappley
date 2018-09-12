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

type Wallets struct {
	Wallets []Wallet
}

func LoadWalletFromFile(filePath string) (*Wallets, error) {

	fileContent, err := storage.GetFileConnection(filePath)

	ws := &Wallets{}
	if err != nil {
		ws.SaveWalletToFile(filePath)
		fileContent, err = storage.GetFileConnection(filePath)
	}
	var wallets Wallets

	gob.Register(bitelliptic.S256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		logger.Error("Wallets: Load wallets failed!")
		logger.Error(err)
	}

	ws.Wallets = wallets.Wallets

	return ws, nil
}

// SaveToFile saves wallets to a file
func (ws Wallets) SaveWalletToFile(filePath string) {
	var content bytes.Buffer

	gob.Register(bitelliptic.S256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		logger.Error("Wallets: save wallets to file failed!")
		logger.Error(err)
	}
	storage.SaveToFile(filePath, content)

}

func (ws *Wallets) CreateWallet() Wallet {
	wallet := NewWallet()

	ws.Wallets = append(ws.Wallets, wallet)

	return wallet
}

func (ws *Wallets) DeleteWallet(key *core.KeyPair) error {
	for i, value := range ws.Wallets {
		if value.Key == key {
			ws.Wallets = append(ws.Wallets[:i], ws.Wallets[i+1:]...)
			return nil
		}
	}

	return errors.New("wallet is not exist")

}

func (ws *Wallets) DeleteWallets() error {
	if len(ws.Wallets) == 0 {
		return errors.New("no wallet yet")
	}
	ws.Wallets = ws.Wallets[:0]
	return nil
}

func (ws *Wallets) GetAddresses() []core.Address {
	var addresses []core.Address

	for _, address := range ws.Wallets {
		addresses = append(addresses, address.GetAddresses()...)
	}

	return addresses
}

func (ws Wallets) GetKeyPairByAddress(address core.Address) core.KeyPair {
	for _, value := range ws.Wallets {

		if value.ContainAddress(address) {
			return *value.Key
		}
	}
	return core.KeyPair{}

}

func (ws Wallets) GetWalletByAddress(address core.Address) Wallet {
	for _, wallet := range ws.Wallets {
		if wallet.ContainAddress(address) {
			return wallet
		}
	}
	return Wallet{}
}


