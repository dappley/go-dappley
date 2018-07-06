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
	Wallets map[string]*core.Address
}

func NewWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*core.Address)

	err := wallets.LoadFromFile()

	return &wallets, err
}

func (ws *Wallets) CreateWallet() string {
	wallet := core.NewAddress()
	address := fmt.Sprintf("%s", wallet.GetAddress())

	ws.Wallets[address] = wallet

	return address
}

func (ws *Wallets) DeleteWallet(address string) error {
	addresses := ws.GetAddresses()
	for _, value := range addresses {
		if value == address {
			delete(ws.Wallets, address)
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

func (ws *Wallets) GetAddresses() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws Wallets) GetWallet(address string) core.Address {
	return *ws.Wallets[address]
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
