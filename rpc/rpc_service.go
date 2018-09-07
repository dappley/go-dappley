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

	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/logic"
)

type RpcService struct{
	node *network.Node
}

// SayHello implements helloworld.GreeterServer
func (rpcSerivce *RpcService) RpcCreateWallet(ctx context.Context, in *rpcpb.CreateWalletRequest) (*rpcpb.CreateWalletResponse, error) {
	return &rpcpb.CreateWalletResponse{Message: "Hello " + in.Name}, nil
}

func (rpcSerivce *RpcService) RpcGetBalance(ctx context.Context, in *rpcpb.GetBalanceRequest) (*rpcpb.GetBalanceResponse, error) {
	return &rpcpb.GetBalanceResponse{Message: "Hello " + in.Name}, nil
}

func (rpcSerivce *RpcService) RpcSend(ctx context.Context, in *rpcpb.SendRequest) (*rpcpb.SendResponse, error) {
	sendFromAddress := core.NewAddress(in.From)
	sendToAddress := core.NewAddress(in.To)
	sendAmount := common.NewAmountFromBytes(in.Amount)

	if sendAmount.Cmp(common.NewAmount(0)) <= 0 {
		return &rpcpb.SendResponse{Message: "Invalid send amount"}, core.ErrInvalidAmount
	}

	localWallets, err := client.NewWallets()
	if err != nil {
		return &rpcpb.SendResponse{Message: "Error loading local wallets"}, err
	}

	senderWallet := localWallets.GetWalletByAddress(sendFromAddress)
	if len(senderWallet.Addresses) == 0 {
		return &rpcpb.SendResponse{Message: "Sender wallet not found"}, errors.New("sender address not found in local wallet")
	}

	err = logic.Send(senderWallet, sendToAddress, sendAmount, 0, rpcSerivce.node.GetBlockchain())
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

func (rpcSerivce *RpcService) RpcGetBlockchainInfo(ctx context.Context, in *rpcpb.GetBlockchainInfoRequest) (*rpcpb.GetBlockchainInfoResponse, error){
	return &rpcpb.GetBlockchainInfoResponse{
		TailBlockHash: rpcSerivce.node.GetBlockchain().GetTailBlockHash(),
		BlockHeight:   rpcSerivce.node.GetBlockchain().GetMaxHeight(),
	}, nil
}
