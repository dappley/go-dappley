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
	"bytes"
	"encoding/gob"
	"errors"
	"github.com/dappley/go-dappley/client/pb"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1/bitelliptic"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"os"
	"strings"
	"sync"
	"time"
	"path/filepath"
)

const walletConfigFilePath = "../client/wallet.conf"

type WalletManager struct {
	Wallets    []*Wallet
	fileLoader storage.FileStorage
	PassPhrase []byte
	mutex      sync.Mutex
	timer      time.Timer
	Locked     bool
}

type WalletData struct {
	Wallets    []*Wallet
	PassPhrase []byte
	Locked     bool
}

func GetWalletFilePath() string {
	conf := &walletpb.WalletConfig{}
	if Exists(walletConfigFilePath) {
		config.LoadConfig(walletConfigFilePath, conf)
	} else if Exists(strings.Replace(walletConfigFilePath, "..", "../..", 1)) {
		config.LoadConfig(strings.Replace(walletConfigFilePath, "..", "../..", 1), conf)
	}

	if conf == nil {
		return ""
	}
	walletPath := strings.Replace(conf.GetFilePath(), "/wallets.dat", "", 1)
	walletfile := ""
	err := errors.New("")
	if Exists(walletPath) {
		walletfile, err = filepath.Abs(conf.GetFilePath())
	} else if Exists(strings.Replace(walletPath, "..", "../..", 1)) {
		walletfile, err = filepath.Abs(strings.Replace(conf.GetFilePath(),"..", "../..", 1))
	}
	if err != nil && err.Error() == ""{
		return walletfile
	}
	return walletfile
}

func NewWalletManager(fileLoader storage.FileStorage) *WalletManager {
	return &WalletManager{
		fileLoader: fileLoader,
	}
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}


func (wm *WalletManager) NewTimer(timeout time.Duration) {
	wm.timer = *time.NewTimer(timeout)
}

func (wm *WalletManager) LoadFromFile() error {

	wm.mutex.Lock()
	fileContent, err := wm.fileLoader.ReadFromFile()

	if err != nil {
		wm.mutex.Unlock()
		wm.SaveWalletToFile()
		fileContent, err = wm.fileLoader.ReadFromFile()
		wm.mutex.Lock()
	}

	var walletdata *WalletData
	gob.Register(bitelliptic.S256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&walletdata)
	if err != nil {
		wm.mutex.Unlock()
		logger.Error("WalletManager: Load Wallets failed!")
		logger.Error(err)
		return err
	}

	wm.Wallets = walletdata.Wallets
	wm.PassPhrase = walletdata.PassPhrase
	wm.Locked = walletdata.Locked
	wm.mutex.Unlock()
	return nil
}

func (wm *WalletManager) IsFileEmpty() (bool, error) {
	fileContent, err := wm.fileLoader.ReadFromFile()
	if err != nil {
		return true, err
	} else {
		return len(fileContent) == 0, nil
	}
}

// SaveToFile saves Wallets to a file
func (wm *WalletManager) SaveWalletToFile() {
	var content bytes.Buffer
	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	gob.Register(bitelliptic.S256())
	encoder := gob.NewEncoder(&content)
	walletdata := WalletData{}
	walletdata.Wallets = wm.Wallets
	walletdata.PassPhrase = wm.PassPhrase
	walletdata.Locked = wm.Locked
	err := encoder.Encode(walletdata)
	if err != nil {
		logger.Error("WalletManager: save Wallets to file failed!")
		logger.Error(err)
	}
	wm.fileLoader.SaveToFile(content)
}

func RemoveWalletFile() {
	conf := &walletpb.WalletConfig{}
	config.LoadConfig(walletConfigFilePath, conf)
	if conf == nil {
		return
	}
	os.Remove(strings.Replace(conf.GetFilePath(), "wallets", "wallets_test", -1))
}

func (wm *WalletManager) AddWallet(wallet *Wallet) {
	wm.mutex.Lock()
	wm.Wallets = append(wm.Wallets, wallet)
	wm.mutex.Unlock()
}

func (wm *WalletManager) GetAddresses() []core.Address {
	var addresses []core.Address

	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	for _, wallet := range wm.Wallets {
		addresses = append(addresses, wallet.GetAddresses()...)
	}

	return addresses
}

func (wm *WalletManager) GetAddressesWithPassphrase(password string) ([]string, error) {
	var addresses []string

	wm.mutex.Lock()
	err := bcrypt.CompareHashAndPassword(wm.PassPhrase, []byte(password))
	if err != nil {
		wm.mutex.Unlock()
		return nil, errors.New("Password not correct!")
	}
	for _, wallet := range wm.Wallets {
		address := wallet.GetAddresses()[0].Address
		addresses = append(addresses, address)
	}
	wm.mutex.Unlock()
	return addresses, nil
}

func (wm *WalletManager) GetKeyPairByAddress(address core.Address) *core.KeyPair {

	wallet := wm.GetWalletByAddress(address)
	if wallet == nil {
		return nil
	}
	return wallet.Key

}

func (wm *WalletManager) GetWalletByAddress(address core.Address) *Wallet {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	for _, wallet := range wm.Wallets {
		if wallet.ContainAddress(address) {
			return wallet
		}
	}
	return nil
}

func (wm *WalletManager) GetWalletByAddressWithPassphrase(address core.Address, password string) (*Wallet, error) {
	err := bcrypt.CompareHashAndPassword(wm.PassPhrase, []byte(password))
	if err == nil {
		wallet := wm.GetWalletByAddress(address)
		if wallet == nil {
			return nil, errors.New("Address not found in the wallets!")
		} else {
			return wallet, nil
		}
	} else {
		return nil, errors.New("Password does not match!")
	}
}

func (wm *WalletManager) SetUnlockTimer(timeout time.Duration) {
	wm.Locked = false
	wm.SaveWalletToFile()
	wm.NewTimer(timeout)
	wm.timer.Reset(timeout)
	go wm.UnlockExpire()
}

func (wm *WalletManager) UnlockExpire() {
	defer wm.timer.Stop()
	select {
	case <-wm.timer.C:
		wm.LoadFromFile()
		wm.Locked = true
		wm.SaveWalletToFile()
	}
}
