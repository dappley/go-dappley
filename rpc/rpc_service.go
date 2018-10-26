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
	"errors"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"

	"strings"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/network/pb"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/storage"
)

const (
	ProtoVersion                  = "1.0.0"
	MaxGetBlocksCount       int32 = 500
	MinUtxoBlockHeaderCount int32 = 6
)

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

func (rpcService *RpcService) RpcGetBalance(ctx context.Context, in *rpcpb.GetBalanceRequest) (*rpcpb.GetBalanceResponse, error) {
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
			}
		} else {
			wallet = wm.GetWalletByAddress(core.NewAddress(address))
			if wallet == nil {
				return &rpcpb.GetBalanceResponse{Message: "Address not found in the wallet!"}, nil
			}
		}

		getbalanceResp := rpcpb.GetBalanceResponse{}
		amount, err := logic.GetBalance(wallet.GetAddress(), rpcService.node.GetBlockchain().GetDb())
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

func (rpcService *RpcService) RpcSend(ctx context.Context, in *rpcpb.SendRequest) (*rpcpb.SendResponse, error) {
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

	txhash, err := logic.Send(senderWallet, sendToAddress, sendAmount, 0, rpcService.node.GetBlockchain(), rpcService.node)
	txhashStr := hex.EncodeToString(txhash)
	if err != nil {
		return &rpcpb.SendResponse{Message: "Error sending [" + txhashStr + "]"}, err
	}

	return &rpcpb.SendResponse{Message: "[" + txhashStr + "] Sent"}, nil
}

func (rpcService *RpcService) RpcGetPeerInfo(ctx context.Context, in *rpcpb.GetPeerInfoRequest) (*rpcpb.GetPeerInfoResponse, error) {
	return &rpcpb.GetPeerInfoResponse{
		PeerList: rpcService.node.GetPeerList().ToProto().(*networkpb.Peerlist),
	}, nil
}

func (rpcService *RpcService) RpcAddProducer(ctx context.Context, in *rpcpb.AddProducerRequest) (*rpcpb.AddProducerResponse, error) {
	if len(in.Address) == 0 {
		return &rpcpb.AddProducerResponse{
			Message: "Error: Address is empty!",
		}, nil
	}
	if in.Name == "addProducer" {
		err := rpcService.node.GetBlockchain().GetConsensus().AddProducer(in.Address)
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

func (rpcService *RpcService) RpcGetBlockchainInfo(ctx context.Context, in *rpcpb.GetBlockchainInfoRequest) (*rpcpb.GetBlockchainInfoResponse, error) {
	return &rpcpb.GetBlockchainInfoResponse{
		TailBlockHash: rpcService.node.GetBlockchain().GetTailBlockHash(),
		BlockHeight:   rpcService.node.GetBlockchain().GetMaxHeight(),
		Producers:     rpcService.node.GetBlockchain().GetConsensus().GetProducers(),
	}, nil
}

func (rpcService *RpcService) RpcAddBalance(ctx context.Context, in *rpcpb.AddBalanceRequest) (*rpcpb.AddBalanceResponse, error) {
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
		err = logic.AddBalance(sendToAddress, sendAmount, rpcService.node.GetBlockchain())
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

	//TODO Race condition Blockchain update after GetUTXO
	getHeaderCount := MaxGetBlocksCount
	if int(getHeaderCount) < len(rpcService.node.GetBlockchain().GetConsensus().GetProducers()) {
		getHeaderCount = int32(len(rpcService.node.GetBlockchain().GetConsensus().GetProducers()))
	}

	tailHeight := rpcService.node.GetBlockchain().GetMaxHeight()
	for i := int32(0); i < getHeaderCount; i++ {
		block, err := rpcService.node.GetBlockchain().GetBlockByHeight(tailHeight - uint64(i))
		if err != nil {
			break
		}

		response.BlockHeaders = append(response.BlockHeaders, block.GetHeader().ToProto().(*corepb.BlockHeader))
	}

	return &response, nil
}

// RpcGetBlocks Get blocks in blockchain from head to tail
func (rpcService *RpcService) RpcGetBlocks(ctx context.Context, in *rpcpb.GetBlocksRequest) (*rpcpb.GetBlocksResponse, error) {
	block := rpcService.findBlockInRequestHash(in.StartBlockHashs)

	// Reach the blockchain's tail
	if block.GetHeight() >= rpcService.node.GetBlockchain().GetMaxHeight() {
		return &rpcpb.GetBlocksResponse{ErrorCode: OK}, nil
	}

	var blocks []*core.Block
	maxBlockCount := int32(rpcService.node.GetBlockchain().GetMaxHeight())
	if maxBlockCount > MaxGetBlocksCount {
		maxBlockCount = MaxGetBlocksCount
	}

	block, err := rpcService.node.GetBlockchain().GetBlockByHeight(block.GetHeight() + 1)
	for i := int32(0); i < maxBlockCount && err == nil; i++ {
		blocks = append(blocks, block)
		block, err = rpcService.node.GetBlockchain().GetBlockByHeight(block.GetHeight() + 1)
	}

	result := &rpcpb.GetBlocksResponse{ErrorCode: OK}

	for _, block = range blocks {
		result.Blocks = append(result.Blocks, block.ToProto().(*corepb.Block))
	}

	return result, nil
}

func (rpcService *RpcService) findBlockInRequestHash(startBlockHashs [][]byte) *core.Block {
	for _, hash := range startBlockHashs {
		// hash in blockchain, return
		if block, err := rpcService.node.GetBlockchain().GetBlockByHash(hash); err == nil {
			return block
		}
	}

	// Return Genesis Block
	block, _ := rpcService.node.GetBlockchain().GetBlockByHeight(0)
	return block
}

// RpcGetBlockByHash Get single block in blockchain by hash
func (rpcService *RpcService) RpcGetBlockByHash(ctx context.Context, in *rpcpb.GetBlockByHashRequest) (*rpcpb.GetBlockByHashResponse, error) {
	block, err := rpcService.node.GetBlockchain().GetBlockByHash(in.Hash)

	if err != nil {
		return &rpcpb.GetBlockByHashResponse{ErrorCode: BlockNotFound}, nil
	}

	return &rpcpb.GetBlockByHashResponse{ErrorCode: OK, Block: block.ToProto().(*corepb.Block)}, nil
}

// RpcGetBlockByHeight Get single block in blockchain by height
func (rpcService *RpcService) RpcGetBlockByHeight(ctx context.Context, in *rpcpb.GetBlockByHeightRequest) (*rpcpb.GetBlockByHeightResponse, error) {
	block, err := rpcService.node.GetBlockchain().GetBlockByHeight(in.Height)

	if err != nil {
		return &rpcpb.GetBlockByHeightResponse{ErrorCode: BlockNotFound}, nil
	}

	return &rpcpb.GetBlockByHeightResponse{ErrorCode: OK, Block: block.ToProto().(*corepb.Block)}, nil
}

func (rpcService *RpcService) RpcSendTransaction(ctx context.Context, in *rpcpb.SendTransactionRequest) (*rpcpb.SendTransactionResponse, error) {
	tx := core.Transaction{nil, nil, nil, 0}
	tx.FromProto(in.Transaction)

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

//unlock the wallet through rpc service
func (rpcService *RpcService) RpcUnlockWallet(ctx context.Context, in *rpcpb.UnlockWalletRequest) (*rpcpb.UnlockWalletResponse, error) {
	msg := "failed"
	if in.Name == "unlock" {
		err := logic.SetUnLockWallet()
		if err != nil {
			msg = err.Error()
		} else {
			msg = "succeed"
		}
	}
	return &rpcpb.UnlockWalletResponse{
		Message: msg,
	}, nil
}
