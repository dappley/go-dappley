package client

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"errors"
	"io/ioutil"
	"log"
	"os"

	"github.com/dappley/go-dappley/core"
)

const WalletFile = "../bin/wallets.dat"

type Wallets struct {
	Wallets map[*core.KeyPair]core.Address
}

func NewWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[*core.KeyPair]core.Address)

	err := wallets.LoadFromFile()

	return &wallets, err
}

func (ws *Wallets) CreateWallet() core.Address {
	wallet := core.NewKeyPair()
	address := wallet.GenerateAddress()

	ws.Wallets[wallet] = address

	return address
}

func (ws *Wallets) DeleteWallet(key *core.KeyPair) error {
	for value := range ws.Wallets {
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

func (ws *Wallets) GetAddresses() []core.Address {
	var addresses []core.Address

	for _, address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws Wallets) GetWallet(address core.Address) core.KeyPair {
	for key, value := range ws.Wallets {
		if value == address {
			return *key
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
