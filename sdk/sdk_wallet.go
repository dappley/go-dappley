package sdk

import (
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	logger "github.com/sirupsen/logrus"
)

type DappSdkWallet struct {
	conn  *DappSdkConn
	addrs []core.Address
	wm    *client.WalletManager
}

//NewDappleySdkWallet creates a new NewDappleySdkWallet instance that connects to a Dappley node with grpc port
func NewDappleySdkWallet(conn *DappSdkConn, numOfWallets int, password string) *DappSdkWallet {

	dappleySdkWallet := &DappSdkWallet{
		conn: conn,
	}

	var err error

	dappleySdkWallet.wm, err = logic.GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		logger.WithError(err).Error("Cannot get wallet manager.")
		return nil
	}

	dappleySdkWallet.addrs = dappleySdkWallet.wm.GetAddresses()
	numOfExisitingWallets := len(dappleySdkWallet.addrs)

	for i := numOfExisitingWallets; i < numOfWallets; i++ {
		_, err := logic.CreateWalletWithpassphrase(password)
		if err != nil {
			logger.WithError(err).Error("Cannot create new wallet.")
			return nil
		}
		logger.WithFields(logger.Fields{
			"address": dappleySdkWallet.addrs[i],
		}).Info("Wallet is created")
	}

	dappleySdkWallet.addrs = dappleySdkWallet.wm.GetAddresses()

	return dappleySdkWallet
}

func (sdkw *DappSdkWallet) GetAddrs() []core.Address {
	return sdkw.addrs
}
