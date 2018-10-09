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
package rpc

import (
	"context"
	"errors"
	"github.com/dappley/go-dappley/common"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"strings"
	"fmt"
)

type RpcService struct {
	node *network.Node
}

// Create Wallet Response
func (rpcSerivce *RpcService) RpcCreateWallet(ctx context.Context, in *rpcpb.CreateWalletRequest) (*rpcpb.CreateWalletResponse, error) {
	msg := ""
	addr := ""
	if in.Name == "getWallet" {
		wallet, err := logic.GetWallet()
		if err != nil {
			msg = err.Error()
		}
		if wallet != nil {
			locked, err := logic.IsWalletLocked()
			if err != nil {
				msg = err.Error()
			} else if locked {
				msg = "WalletExistsLocked"
			} else {
				msg = "WalletExistsNotLocked"
			}
		} else {
			msg = "NewWallet"
		}
	} else if in.Name == "createWallet" {
		locked, err := logic.IsWalletLocked()
		if err != nil {
			return &rpcpb.CreateWalletResponse{
				Message: err.Error(),
			}, nil
		} else if locked {
			passPhrase := in.Passphrase
			if len(passPhrase) == 0 {
				logger.Error("CreateWallet: Password is empty!")
				msg = "Create Wallet Error: Password Empty!"
				return &rpcpb.CreateWalletResponse{
					Message: msg,
					Address: ""}, nil
			}
			wallet, err := logic.CreateWalletWithpassphrase(passPhrase)
			if err != nil {
				msg = "Create Wallet Error: Password not correct!"
				addr = ""
			} else if wallet != nil {
				err = logic.SetUnLockWallet()
				if err != nil {
					msg = err.Error()
				}
				addr = wallet.GetAddress().Address
				msg = "Create Wallet: "

			} else {
				msg = "Create Wallet Error: Wallet Empty!"
				addr = ""
			}
		} else { //unlock
			wallet, err := logic.AddWallet()
			if err != nil {
				msg = err.Error()
			} else if wallet != nil {
				addr = wallet.GetAddress().Address
				msg = "Create Wallet: "

			} else {
				msg = "Create Wallet Error: Wallet Empty!"
				addr = ""
			}
		}

	} else {
		msg = "Error: not recognize the command!"
	}
	return &rpcpb.CreateWalletResponse{
		Message: msg,
		Address: addr}, nil
}

func (rpcSerivce *RpcService) RpcGetBalance(ctx context.Context, in *rpcpb.GetBalanceRequest) (*rpcpb.GetBalanceResponse, error) {
	msg := ""
	if in.Name == "getWallet" {
		wallet, err := logic.GetWallet()
		if err != nil {
			msg = err.Error()
		} else if wallet != nil {
			locked, err := logic.IsWalletLocked()
			if err != nil {
				msg = err.Error()
			} else if locked {
				msg = "WalletExistsLocked"
			} else {
				msg = "WalletExistsNotLocked"
			}
		} else {
			msg = "NoWallet"
		}
		return &rpcpb.GetBalanceResponse{Message: msg}, nil
	} else if in.Name == "getBalance" {
		pass := in.Passphrase
		address := in.Address
		msg = "Get Balance"
		fl := storage.NewFileLoader(client.GetWalletFilePath())
		wm := client.NewWalletManager(fl)
		err := wm.LoadFromFile()
		if err != nil {
			return &rpcpb.GetBalanceResponse{Message: "GetBalance : Error loading local wallets"}, err
		}

		wallet := client.NewWallet()
		if wm.Locked {
			wallet, err = wm.GetWalletByAddressWithPassphrase(core.NewAddress(address), pass)
			if err != nil {
				return &rpcpb.GetBalanceResponse{Message: err.Error()}, err
			} else {
				wm.SetUnlockTimer(logic.GetUnlockDuration())
				fmt.Println("Set unlock **** get balance", wm.Locked)
			}
		} else {
			wallet = wm.GetWalletByAddress(core.NewAddress(address))
			if wallet == nil {
				return &rpcpb.GetBalanceResponse{Message: errors.New("Address not found in the wallet!").Error()}, nil
			}
		}

		getbalanceResp := rpcpb.GetBalanceResponse{}
		amount, err := logic.GetBalance(wallet.GetAddress(), rpcSerivce.node.GetBlockchain().GetDb())
		if err != nil {
			getbalanceResp.Message = "Failed to get balance from blockchain"
			return &getbalanceResp, nil
		}
		getbalanceResp.Amount = amount.Int64()
		getbalanceResp.Message = msg
		return &getbalanceResp, nil
	} else {
		return &rpcpb.GetBalanceResponse{Message: "GetBalance Error: not recognize the command!"}, nil
	}
}

func (rpcSerivce *RpcService) RpcSend(ctx context.Context, in *rpcpb.SendRequest) (*rpcpb.SendResponse, error) {
	sendFromAddress := core.NewAddress(in.From)
	sendToAddress := core.NewAddress(in.To)
	sendAmount := common.NewAmountFromBytes(in.Amount)

	if sendAmount.Validate() != nil || sendAmount.IsZero() {
		return &rpcpb.SendResponse{Message: "Invalid send amount"}, core.ErrInvalidAmount
	}

	if len(in.Walletpath) == 0 {
		return &rpcpb.SendResponse{Message: "Wallet path empty error"}, core.ErrInvalidAmount
	}

	fl := storage.NewFileLoader(in.Walletpath)
	wm := client.NewWalletManager(fl)
	err := wm.LoadFromFile()

	if err != nil {
		return &rpcpb.SendResponse{Message: "Error loading local wallets"}, err
	}

	senderWallet := wm.GetWalletByAddress(sendFromAddress)
	if len(senderWallet.Addresses) == 0 {
		return &rpcpb.SendResponse{Message: "Sender wallet not found"}, errors.New("sender address not found in local wallet")
	}

	err = logic.Send(senderWallet, sendToAddress, sendAmount, 0, rpcSerivce.node.GetBlockchain(), rpcSerivce.node)
	if err != nil {
		return &rpcpb.SendResponse{Message: "Error sending"}, err
	}

	return &rpcpb.SendResponse{Message: "Sent"}, nil
}

func (rpcSerivce *RpcService) RpcGetPeerInfo(ctx context.Context, in *rpcpb.GetPeerInfoRequest) (*rpcpb.GetPeerInfoResponse, error) {
	return &rpcpb.GetPeerInfoResponse{
		PeerList: rpcSerivce.node.GetPeerList().ToProto().(*networkpb.Peerlist),
	}, nil
}

func (rpcSerivce *RpcService) RpcAddProducer(ctx context.Context, in *rpcpb.AddProducerRequest) (*rpcpb.AddProducerResponse, error) {
	if len(in.Address) == 0 {
		return &rpcpb.AddProducerResponse{
			Message: "Error: Address is empty!",
		}, nil
	}
	if in.Name == "addProducer" {
		err := rpcSerivce.node.GetBlockchain().GetConsensus().AddProducer(in.Address)
		if err == nil {
			return &rpcpb.AddProducerResponse{
				Message: "Add producer sucessfully!",
			}, nil
		} else {
			return &rpcpb.AddProducerResponse{
				Message: "Error: Add producer failed! " + err.Error(),
			}, nil
		}
	} else {
		return &rpcpb.AddProducerResponse{
			Message: "Error: Command not recognized!",
		}, nil
	}

	return &rpcpb.AddProducerResponse{}, nil
}

func (rpcSerivce *RpcService) RpcGetBlockchainInfo(ctx context.Context, in *rpcpb.GetBlockchainInfoRequest) (*rpcpb.GetBlockchainInfoResponse, error) {
	return &rpcpb.GetBlockchainInfoResponse{
		TailBlockHash: rpcSerivce.node.GetBlockchain().GetTailBlockHash(),
		BlockHeight:   rpcSerivce.node.GetBlockchain().GetMaxHeight(),
	}, nil
}

func (rpcSerivce *RpcService) RpcGetWalletAddress(ctx context.Context, in *rpcpb.GetWalletAddressRequest) (*rpcpb.GetWalletAddressResponse, error) {

	if in.Name == "getWallet" {
		msg := ""
		wallet, err := logic.GetWallet()
		if err != nil {
			msg = err.Error()
		}
		if wallet != nil {
			locked, err := logic.IsWalletLocked()
			if err != nil {
				msg = err.Error()
			} else if locked {
				msg = "WalletExistsLocked"
			} else {
				msg = "WalletExistsNotLocked"
			}
		} else {
			msg = "NoWallet"
		}
		getWalletAddress := rpcpb.GetWalletAddressResponse{}
		getWalletAddress.Message = msg
		return &getWalletAddress, nil
	} else if in.Name == "listAddresses" {
		pass := in.Passphrase
		fl := storage.NewFileLoader(client.GetWalletFilePath())
		wm := client.NewWalletManager(fl)
		err := wm.LoadFromFile()
		if err != nil {
			return &rpcpb.GetWalletAddressResponse{Message: "ListWalletAddresses: Error loading local wallet"}, err
		}
		addressList := []string{}
		if wm.Locked {
			addressList, err = wm.GetAddressesWithPassphrase(pass)
		} else {
			addresses := wm.GetAddresses()
			for _, addr := range addresses {
				addressList = append(addressList, addr.Address)
			}
			err = nil
		}

		if err != nil {
			if strings.Contains(err.Error(), "Password not correct") {
				return &rpcpb.GetWalletAddressResponse{Message: "ListWalletAddresses: Password not correct"}, nil
			} else {
				return &rpcpb.GetWalletAddressResponse{Message: err.Error()}, err
			}
		}
		if wm.Locked {
			wm.SetUnlockTimer(logic.GetUnlockDuration())
		}

		//set the timer
		getWalletAddress := rpcpb.GetWalletAddressResponse{}
		getWalletAddress.Address = addressList
		return &getWalletAddress, nil
	} else {
		getWalletAddress := rpcpb.GetWalletAddressResponse{}
		getWalletAddress.Message = "Error: not recognize the command!"
		return &getWalletAddress, nil
	}
}

func (rpcSerivce *RpcService) RpcAddBalance(ctx context.Context, in *rpcpb.AddBalanceRequest) (*rpcpb.AddBalanceResponse, error) {
	sendToAddress := core.NewAddress(in.Address)
	sendAmount := common.NewAmountFromBytes(in.Amount)
	if sendAmount.Validate() != nil || sendAmount.IsZero() {
		return &rpcpb.AddBalanceResponse{Message: "Invalid send amount (must be >0)"}, nil
	}

	fl := storage.NewFileLoader(client.GetWalletFilePath())
	wm := client.NewWalletManager(fl)
	err := wm.LoadFromFile()
	if err != nil {
		return &rpcpb.AddBalanceResponse{Message: "Error loading local wallets"}, err
	}

	receiverWallet := wm.GetWalletByAddress(sendToAddress)
	if receiverWallet == nil {
		return &rpcpb.AddBalanceResponse{Message: "Address not found in the wallet!"}, nil
	} else {
		err = logic.AddBalance(sendToAddress, sendAmount, rpcSerivce.node.GetBlockchain())
		if err != nil {
			return &rpcpb.AddBalanceResponse{Message: "Add balance failed, " + err.Error()}, nil
		} else {
			addBalanceResponse := rpcpb.AddBalanceResponse{}
			addBalanceResponse.Message = "Add balance succeed!"
			return &addBalanceResponse, nil
		}
	}
}
