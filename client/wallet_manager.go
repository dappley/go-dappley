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
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/storage"
	"encoding/gob"
	"bytes"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1/bitelliptic"
	logger "github.com/sirupsen/logrus"
	"os"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/client/pb"
)

const walletConfigFilePath = "../client/wallet.conf"

type WalletManager struct {
	Wallets  	[]*Wallet
	fileLoader 	storage.FileStorage
}

func GetWalletFilePath() string{
	conf := &walletpb.WalletConfig{}
	config.LoadConfig(walletConfigFilePath, conf)
	if conf == nil {
		return ""
	}
	return conf.GetFilePath()
}

func NewWalletManager(fileLoader storage.FileStorage) *WalletManager{
	return &WalletManager{
		fileLoader: fileLoader,
	}
}

func (wm *WalletManager) LoadFromFile() error{

	fileContent, err := wm.fileLoader.ReadFromFile()

	if err != nil {
		wm.SaveWalletToFile()
		fileContent, err = wm.fileLoader.ReadFromFile()
	}
	var wallets []*Wallet

	gob.Register(bitelliptic.S256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		logger.Error("WalletManager: Load Wallets failed!")
		logger.Error(err)
		return err
	}

	wm.Wallets = wallets

	return nil
}

// SaveToFile saves Wallets to a file
func (wm *WalletManager) SaveWalletToFile() {
	var content bytes.Buffer

	gob.Register(bitelliptic.S256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(wm.Wallets)
	if err != nil {
		logger.Error("WalletManager: save Wallets to file failed!")
		logger.Error(err)
	}
	wm.fileLoader.SaveToFile(content)
}

func RemoveWalletFile(){
	conf := &walletpb.WalletConfig{}
	config.LoadConfig(walletConfigFilePath, conf)
	if conf == nil {
		return
	}
	os.Remove(conf.GetFilePath())
}

func (wm *WalletManager) AddWallet(wallet *Wallet){
	wm.Wallets = append(wm.Wallets, wallet)
}

func (wm *WalletManager) GetAddresses() []core.Address {
	var addresses []core.Address

	for _, wallet := range wm.Wallets {
		addresses = append(addresses, wallet.GetAddresses()...)
	}

	return addresses
}

func (wm *WalletManager) GetKeyPairByAddress(address core.Address) *core.KeyPair {

	wallet := wm.GetWalletByAddress(address)
	if wallet == nil {
		return nil
	}
	return wallet.Key

}

func (wm *WalletManager) GetWalletByAddress(address core.Address) *Wallet {
	for _, wallet := range wm.Wallets {
		if wallet.ContainAddress(address) {
			return wallet
		}
	}
	return nil
}


