package sdk

import (
	"sync"

	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic/lutxo"

	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/logic/laccount"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
)

type DappSdkAccount struct {
	addrs     []account.Address
	balances  map[account.Address]uint64
	wm        *laccount.AccountManager
	sdk       *DappSdk
	utxoIndex *lutxo.UTXOIndex
	mutex     *sync.RWMutex
}

//NewDappSdkAccount creates a new NewDappSdkAccount instance that connects to a Dappley node with grpc port
func NewDappSdkAccount(numOfAccounts uint32, password string, sdk *DappSdk) *DappSdkAccount {

	dappSdkAccount := &DappSdkAccount{
		sdk:   sdk,
		mutex: &sync.RWMutex{},
	}

	var err error

	dappSdkAccount.wm, err = logic.GetAccountManager(laccount.GetAccountFilePath())
	if err != nil {
		logger.WithError(err).Error("DappSdkAccount: Cannot get account manager.")
		return nil
	}

	dappSdkAccount.addrs = dappSdkAccount.wm.GetAddresses()
	numOfExisitingAccounts := len(dappSdkAccount.addrs)

	for i := numOfExisitingAccounts; i < int(numOfAccounts); i++ {
		w, err := logic.CreateAccountWithpassphrase(password)
		if err != nil {
			logger.WithError(err).Error("DappSdkAccount: Cannot create new account.")
			return nil
		}
		logger.WithFields(logger.Fields{
			"address": w.GetKeyPair().GenerateAddress(),
		}).Info("DappSdkAccount: Account is created")
	}

	dappSdkAccount.addrs = dappSdkAccount.wm.GetAddresses()
	dappSdkAccount.Initialize()

	return dappSdkAccount
}

func (sdkw *DappSdkAccount) GetAddrs() []account.Address { return sdkw.addrs }

func (sdkw *DappSdkAccount) GetBalance(address account.Address) uint64 {
	sdkw.mutex.RLock()
	defer sdkw.mutex.RUnlock()

	return sdkw.balances[address]
}

func (sdkw *DappSdkAccount) GetAccountManager() *laccount.AccountManager { return sdkw.wm }

func (sdkw *DappSdkAccount) GetUtxoIndex() *lutxo.UTXOIndex { return sdkw.utxoIndex }

func (sdkw *DappSdkAccount) Initialize() {
	sdkw.mutex.Lock()
	defer sdkw.mutex.Unlock()

	sdkw.utxoIndex = lutxo.NewUTXOIndex(utxo.NewUTXOCache(storage.NewRamStorage()))
	sdkw.balances = make(map[account.Address]uint64)
}

func (sdkw *DappSdkAccount) IsZeroBalance() bool {
	sdkw.mutex.RLock()
	defer sdkw.mutex.RUnlock()
	for _, addr := range sdkw.GetAddrs() {
		if sdkw.balances[addr] > 0 {
			return false
		}
	}
	return true
}

//UpdateBalances updates all the balances of the addresses in the account
func (sdkw *DappSdkAccount) DisplayBalances() {
	sdkw.mutex.RLock()
	defer sdkw.mutex.RUnlock()

	for _, addr := range sdkw.GetAddrs() {
		logger.WithFields(logger.Fields{
			"address": addr.String(),
			"balance": sdkw.balances[addr],
		}).Info("DappSdkAccount: Updating account balance...")
	}
}

//Update updates the balance and utxos of all addresses in the account from the server
func (sdkw *DappSdkAccount) Update() error {

	logger.Info("DappSdkAccount: Updating from server")

	sdkw.Initialize()

	for _, addr := range sdkw.addrs {

		kp := sdkw.wm.GetKeyPairByAddress(addr)
		_, err := account.NewUserPubKeyHash(kp.GetPublicKey())
		if err != nil {
			return err
		}

		utxos, err := sdkw.sdk.GetUtxoByAddr(addr)
		if err != nil {
			return err
		}

		for _, utxoPb := range utxos {
			utxo := utxo.UTXO{}
			utxo.FromProto(utxoPb)
			sdkw.utxoIndex.AddUTXO(utxo.TXOutput, utxo.Txid, utxo.TxIndex)
			sdkw.UpdateBalance(addr, sdkw.GetBalance(addr)+utxo.TXOutput.Value.Uint64())
		}
	}

	return nil
}

//AddToBalance adds the difference to the current balance
func (sdkw *DappSdkAccount) UpdateBalance(addr account.Address, amount uint64) {
	sdkw.mutex.Lock()
	defer sdkw.mutex.Unlock()
	sdkw.balances[addr] = amount
}
