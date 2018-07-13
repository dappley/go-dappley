package client

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io/ioutil"
	"log"
	"os"

	"github.com/dappley/go-dappley/core"
	"crypto/elliptic"
)

const WalletFile = "../bin/wallets.dat"

type Wallets struct {
	Wallets []Wallet
}

func NewWallets() (*Wallets, error) {
	wallets := Wallets{}

	err := wallets.LoadFromFile()

	return &wallets, err
}

func (ws *Wallets) CreateWallet() core.Address {
	wallet := NewWallet()
	address := wallet.GetAddress()

	ws.Wallets = append(ws.Wallets, wallet)

	return address
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
