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
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"strings"
)

const ProtoVersion = "1.0.0"

type RpcService struct {
	node *network.Node
}

func (rpcService *RpcService) RpcGetVersion(ctx context.Context, in *rpcpb.GetVersionRequest) (*rpcpb.GetVersionResponse, error) {
	clientProtoVersions := strings.Split(in.ProtoVersion, ".")

	if len(clientProtoVersions) != 3 {
		return &rpcpb.GetVersionResponse{ErrorCode: ProtoVersionNotSupport, ProtoVersion: ProtoVersion, ServerVersion: ""}, nil
	}

	serverProtoVersions := strings.Split(ProtoVersion, ".")

	// Major version must equal
	if serverProtoVersions[0] != clientProtoVersions[0] {
		return &rpcpb.GetVersionResponse{ErrorCode: ProtoVersionNotSupport, ProtoVersion: ProtoVersion, ServerVersion: ""}, nil
	}

	return &rpcpb.GetVersionResponse{ErrorCode: OK, ProtoVersion: ProtoVersion, ServerVersion: ""}, nil
}

// SayHello implements helloworld.GreeterServer
func (rpcSerivce *RpcService) RpcCreateWallet(ctx context.Context, in *rpcpb.CreateWalletRequest) (*rpcpb.CreateWalletResponse, error) {
	msg := ""
	addr := ""
	if in.Name == "getWallet" {
		wallet, err := logic.GetWallet()
		if err != nil {
			msg = err.Error()
		}
		if wallet != nil {
			msg = "WalletExists"
		} else {
			msg = "NewWallet"
		}
	} else if in.Name == "createWallet" {
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
			addr = wallet.GetAddress().Address
			msg = "Create Wallet: "

		} else {
			msg = "Create Wallet Error: Wallet Empty!"
			addr = ""
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
		}
		if wallet != nil {
			msg = "WalletExists"
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

		wallet, err := wm.GetWalletByAddressWithPassphrase(core.NewAddress(address), pass)
		if err != nil {
			return &rpcpb.GetBalanceResponse{Message: err.Error()}, err
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
			msg = "WalletExists"
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

		addressList, err := wm.GetAddressesWithPassphrase(pass)
		if err != nil {
			if strings.Contains(err.Error(), "Password not correct") {
				return &rpcpb.GetWalletAddressResponse{Message: "ListWalletAddresses: Password not correct"}, nil
			} else {
				return &rpcpb.GetWalletAddressResponse{Message: err.Error()}, err
			}
		}
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

func (rpcService *RpcService) RpcGetUTXO(ctx context.Context, in *rpcpb.GetUTXORequest) (*rpcpb.GetUTXOResponse, error) {
	utxoIndex := core.LoadUTXOIndex(rpcService.node.GetBlockchain().GetDb())
	publicKeyHash, err := core.NewAddress(in.Address).GetPubKeyHash()
	if err == false {
		return &rpcpb.GetUTXOResponse{ErrorCode: InvalidAddress}, nil
	}

	utxos := utxoIndex.GetUTXOsByPubKeyHash(publicKeyHash)
	response := rpcpb.GetUTXOResponse{ErrorCode: OK}
	for _, utxo := range utxos {
		response.Utxos = append(
			response.Utxos,
			&rpcpb.UTXO{
				Amount:        utxo.Value.BigInt().Int64(),
				PublicKeyHash: utxo.PubKeyHash,
				Txid:          utxo.Txid,
				TxIndex:       uint32(utxo.TxIndex),
			},
		)
	}

	return &response, nil
}

func (rpcService *RpcService) RpcGetBlocks(ctx context.Context, in *rpcpb.GetBlocksRequest) (*rpcpb.GetBlocksResponse, error) {
	return &rpcpb.GetBlocksResponse{ErrorCode: OK}, nil
}

func (rpcService *RpcService) RpcSendTransaction(ctx context.Context, in *rpcpb.SendTransactionRequest) (*rpcpb.SendTransactionResponse, error) {
	tx := core.Transaction{nil, nil, nil, 0}
	tx.FromProto(in)

	if tx.IsCoinbase() {
		return &rpcpb.SendTransactionResponse{ErrorCode: InvalidTransaction}, nil
	}

	//TODO Check double spend in transaction pool
	utxoIndex := core.LoadUTXOIndex(rpcService.node.GetBlockchain().GetDb())
	if tx.Verify(utxoIndex, 0) == false {
		return &rpcpb.SendTransactionResponse{ErrorCode: InvalidTransaction}, nil
	}

	rpcService.node.GetBlockchain().GetTxPool().Push(tx)
	rpcService.node.TxBroadcast(&tx)

	return &rpcpb.SendTransactionResponse{ErrorCode: OK}, nil
}
