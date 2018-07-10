package client

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/dappley/go-dappley/core"
)

const WalletFile = "../bin/client.dat"

type Wallets struct {
	Wallets map[*core.KeyPair]Wallet
}

func NewWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[*core.KeyPair]Wallet)

	err := wallets.LoadFromFile()

	return &wallets, err
}

func CreateAddressByKeyPar(key *core.KeyPair) string {
	return fmt.Sprintf("%s", key.GetAddress())
}

func (ws *Wallets) AddWallet(wallet Wallet) {
	ws.Wallets[wallet.key] = wallet
}

func (ws *Wallets) DeleteWallet(key *core.KeyPair) error {
	keys := ws.GetKeys()
	for _, value := range keys {
		if value == key {
			delete(ws.Wallets, key)
			return nil
		}
	}
	return errors.New("wallet is not exist")

}

func (ws *Wallets) DeleteWallets() error {
	if len(ws.Wallets) == 0 {
		return errors.New("no wallet yet")
	}
	for k := range ws.Wallets {
		delete(ws.Wallets, k)
	}
	return nil
}

func (ws *Wallets) GetKeys() []*core.KeyPair {
	var keys []*core.KeyPair

	for key, _ := range ws.Wallets {
		keys = append(keys, key)
	}
	return keys
}

func (ws *Wallets) GetAddresses() []string {
	var addresses []string

	for _, wallet := range ws.Wallets {
		addresses = append(addresses, wallet.GetAddress()...)
	}

	return addresses
}

func (ws Wallets) GetWallet(address string) core.KeyPair {
	for key, wallet := range ws.Wallets {
		addresses := wallet.GetAddress()
		for _, value := range addresses {
			if value == address {
				return *key
			}
		}
	}

	return core.KeyPair{}
}

func (ws *Wallets) LoadFromFile() error {
	if _, err := os.Stat(WalletFile); os.IsNotExist(err) {
		ws.SaveToFile()
	} else if err != nil {
		return err
	}

	fileContent, err := ioutil.ReadFile(WalletFile)
	if err != nil {
		log.Panic(err)
	}

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}

	ws.Wallets = wallets.Wallets

	return nil
}

// SaveToFile saves wallets to a file
func (ws Wallets) SaveToFile() {
	var content bytes.Buffer

	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(WalletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}
