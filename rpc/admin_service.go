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
	"encoding/hex"
	"sync"

	"github.com/dappley/go-dappley/consensus"
	errval "github.com/dappley/go-dappley/errors"

	"time"

	"github.com/dappley/go-dappley/logic/lblockchain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
	networkpb "github.com/dappley/go-dappley/network/pb"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/wallet"
)

type AdminRpcService struct {
	bm      *lblockchain.BlockchainManager
	node    *network.Node
	dynasty *consensus.Dynasty
	mutex   sync.Mutex
}

func (adminRpcService *AdminRpcService) RpcAddPeer(ctx context.Context, in *rpcpb.AddPeerRequest) (*rpcpb.AddPeerResponse, error) {
	err := adminRpcService.node.GetNetwork().ConnectToSeedByString(in.GetFullAddress())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &rpcpb.AddPeerResponse{}, nil
}

func (adminRpcService *AdminRpcService) RpcChangeProducer(ctx context.Context, in *rpcpb.ChangeProducerRequest) (*rpcpb.ChangeProducerResponse, error) {

	addresses := in.GetAddresses()
	height := in.GetHeight()
	kind := in.GetKind()
	adminRpcService.mutex.Lock()
	_, err := logic.SendProducerModifyTX(addresses, height, adminRpcService.bm.Getblockchain(), kind)
	adminRpcService.mutex.Unlock()
	if err != nil {
		switch err {
		case errval.InvalidSenderAddress, errval.InvalidRcverAddress, errval.InvalidAmount:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errval.InsufficientFund:
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		default:
			return nil, status.Error(codes.Unknown, err.Error())
		}
	}

	logic.ChangeProducers(addresses, height, adminRpcService.bm, int(kind))
	return &rpcpb.ChangeProducerResponse{}, nil
}

func (adminRpcService *AdminRpcService) RpcGetPeerInfo(ctx context.Context, in *rpcpb.GetPeerInfoRequest) (*rpcpb.GetPeerInfoResponse, error) {

	return &rpcpb.GetPeerInfoResponse{
		PeerList: getPeerInfo(adminRpcService.node),
	}, nil
}

func (adminRpcService *AdminRpcService) RpcSend(ctx context.Context, in *rpcpb.SendRequest) (*rpcpb.SendResponse, error) {
	start := time.Now().UnixNano() / 1e6
	txRequestStats.concurrentCounter.Inc(1)
	defer func() {
		if txRequestStats.responseTime.Count()/100 == 0 {
			txRequestStats.responseTime.Clear()
		}
		txRequestStats.responseTime.Update(time.Now().UnixNano()/1e6 - start)
	}()
	defer txRequestStats.requestPerSec.Mark(1)
	defer txRequestStats.concurrentCounter.Dec(1)

	sendFromAddress := account.NewAddress(in.GetFrom())
	sendToAddress := account.NewAddress(in.GetTo())
	sendAmount := common.NewAmountFromBytes(in.GetAmount())
	tip := common.NewAmountFromBytes(in.GetTip())
	gasLimit := common.NewAmountFromBytes(in.GetGasLimit())
	gasPrice := common.NewAmountFromBytes(in.GetGasPrice())

	if sendAmount.Validate() != nil || sendAmount.IsZero() {
		return nil, status.Error(codes.InvalidArgument, errval.InvalidAmount.Error())
	}
	path := in.GetAccountPath()
	if len(path) == 0 {
		path = wallet.GetAccountFilePath()
	}

	am, err := logic.GetAccountManager(path)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}

	senderAccount := am.GetAccountByAddress(sendFromAddress)
	if senderAccount == nil || senderAccount.GetKeyPair() == nil {
		return nil, status.Error(codes.NotFound, errval.AddressNotFound.Error())
	}

	adminRpcService.mutex.Lock()
	txHash, scAddress, err := logic.Send(senderAccount, sendToAddress, sendAmount, tip, gasLimit, gasPrice, in.GetData(),
		adminRpcService.bm.Getblockchain())
	adminRpcService.mutex.Unlock()

	txHashStr := hex.EncodeToString(txHash)
	if err != nil {
		switch err {
		case errval.InvalidSenderAddress, errval.InvalidRcverAddress, errval.InvalidAmount:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errval.InsufficientFund:
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		default:
			return nil, status.Error(codes.Unknown, err.Error())
		}
	}

	resp := &rpcpb.SendResponse{Txid: txHashStr}
	if scAddress != "" {
		resp.ContractAddress = scAddress
	}

	return resp, nil
}

func (adminRpcService *AdminRpcService) IsPrivate() bool { return true }

func getPeerInfo(node *network.Node) []*networkpb.PeerInfo {
	peers := node.GetNetwork().GetConnectedPeers()

	var peerPbs []*networkpb.PeerInfo
	for _, peerInfo := range peers {
		peerPbs = append(peerPbs, peerInfo.ToProto().(*networkpb.PeerInfo))
	}

	return peerPbs
}
