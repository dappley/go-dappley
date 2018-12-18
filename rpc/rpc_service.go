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
	"strings"

	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
	"github.com/dappley/go-dappley/rpc/pb"
)

const (
	ProtoVersion                   = "1.0.0"
	MaxGetBlocksCount       int32  = 500
	MinUtxoBlockHeaderCount uint64 = 6
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

	if in.Name == "getBalance" {
		getbalanceResp := rpcpb.GetBalanceResponse{}
		address := in.Address
		if core.NewAddress(address).ValidateAddress() == false {
			getbalanceResp.Message = "The address is not valid"
			return &getbalanceResp, nil
		}

		amount, err := logic.GetBalance(core.NewAddress(address), rpcService.node.GetBlockchain().GetDb())
		if err != nil {
			getbalanceResp.Message = "Failed to get balance from blockchain"
			return &getbalanceResp, nil
		}
		getbalanceResp.Amount = amount.Int64()
		getbalanceResp.Message = "Succeed"
		return &getbalanceResp, nil
	}
	return &rpcpb.GetBalanceResponse{Message: "Error: Get balance failded. not recognize the command!"}, nil

}

func (rpcService *RpcService) RpcGetBlockchainInfo(ctx context.Context, in *rpcpb.GetBlockchainInfoRequest) (*rpcpb.GetBlockchainInfoResponse, error) {
	tailBlock, err := rpcService.node.GetBlockchain().GetTailBlock()
	if err != nil {
		return nil, status.Error(codes.Internal, "Internal error")
	}

	return &rpcpb.GetBlockchainInfoResponse{
		TailBlockHash: rpcService.node.GetBlockchain().GetTailBlockHash(),
		BlockHeight:   rpcService.node.GetBlockchain().GetMaxHeight(),
		Producers:     rpcService.node.GetBlockchain().GetConsensus().GetProducers(),
		Timestamp:     tailBlock.GetTimestamp(),
	}, nil
}

func (rpcService *RpcService) RpcGetUTXO(ctx context.Context, in *rpcpb.GetUTXORequest) (*rpcpb.GetUTXOResponse, error) {
	utxoIndex := core.LoadUTXOIndex(rpcService.node.GetBlockchain().GetDb())
	publicKeyHash, err := core.NewAddress(in.Address).GetPubKeyHash()
	if err == false {
		return &rpcpb.GetUTXOResponse{ErrorCode: InvalidAddress}, nil
	}

	utxos := utxoIndex.GetAllUTXOsByPubKeyHash(publicKeyHash)
	response := rpcpb.GetUTXOResponse{ErrorCode: OK}
	for _, utxo := range utxos {
		response.Utxos = append(
			response.Utxos,
			&rpcpb.UTXO{
				Amount:        utxo.Value.Bytes(),
				PublicKeyHash: utxo.PubKeyHash.GetPubKeyHash(),
				Txid:          utxo.Txid,
				TxIndex:       uint32(utxo.TxIndex),
			},
		)
	}

	//TODO Race condition Blockchain update after GetUTXO
	getHeaderCount := MinUtxoBlockHeaderCount
	if int(getHeaderCount) < len(rpcService.node.GetBlockchain().GetConsensus().GetProducers()) {
		getHeaderCount = uint64(len(rpcService.node.GetBlockchain().GetConsensus().GetProducers()))
	}

	tailHeight := rpcService.node.GetBlockchain().GetMaxHeight()
	if getHeaderCount > tailHeight {
		getHeaderCount = tailHeight
	}

	for i := uint64(0); i < getHeaderCount; i++ {
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
	block := rpcService.findBlockInRequestHash(in.StartBlockHashes)

	// Reach the blockchain's tail
	if block.GetHeight() >= rpcService.node.GetBlockchain().GetMaxHeight() {
		return &rpcpb.GetBlocksResponse{ErrorCode: OK}, nil
	}

	var blocks []*core.Block
	maxBlockCount := in.MaxCount
	if maxBlockCount > MaxGetBlocksCount {
		return &rpcpb.GetBlocksResponse{ErrorCode: GetBlocksCountOverflow}, nil
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

func (rpcService *RpcService) findBlockInRequestHash(startBlockHashes [][]byte) *core.Block {
	for _, hash := range startBlockHashes {
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

// RpcSendTransaction Send transaction to blockchain created by wallet client
func (rpcService *RpcService) RpcSendTransaction(ctx context.Context, in *rpcpb.SendTransactionRequest) (*rpcpb.SendTransactionResponse, error) {
	tx := core.Transaction{nil, nil, nil, 0}
	tx.FromProto(in.Transaction)

	if tx.IsCoinbase() {
		return &rpcpb.SendTransactionResponse{ErrorCode: InvalidTransaction}, nil
	}

	utxoIndex := core.LoadUTXOIndex(rpcService.node.GetBlockchain().GetDb())

	if tx.Verify(*utxoIndex, core.NewTransactionPool(128), 0) == false {
		return &rpcpb.SendTransactionResponse{ErrorCode: InvalidTransaction}, nil
	}

	rpcService.node.GetBlockchain().GetTxPool().Push(tx)
	rpcService.node.TxBroadcast(&tx)

	return &rpcpb.SendTransactionResponse{ErrorCode: OK}, nil
}

// RpcSendBatchTransaction sends a batch of transactions to blockchain created by wallet client
func (rpcService *RpcService) RpcSendBatchTransaction(ctx context.Context, in *rpcpb.SendBatchTransactionRequest) (*rpcpb.SendBatchTransactionResponse, error) {
	respErrorCode := []uint32{}
	for _, txInReq := range in.Transaction {
		tx := core.Transaction{nil, nil, nil, 0}
		tx.FromProto(txInReq)

		if tx.IsCoinbase() {
			respErrorCode = append(respErrorCode, InvalidTransaction)
			continue
		}

		utxoIndex := core.LoadUTXOIndex(rpcService.node.GetBlockchain().GetDb())

		if tx.Verify(*utxoIndex, core.NewTransactionPool(128), 0) == false {
			respErrorCode = append(respErrorCode, InvalidTransaction)
			continue
		}

		rpcService.node.GetBlockchain().GetTxPool().Push(tx)
		rpcService.node.TxBroadcast(&tx)

		respErrorCode = append(respErrorCode, OK)
	}
	return &rpcpb.SendBatchTransactionResponse{ErrorCode: respErrorCode}, nil
}

func (rpcService *RpcService) RpcGetNewTransactions(in *rpcpb.GetNewTransactionsRequest, stream rpcpb.RpcService_RpcGetNewTransactionsServer) error {
	var txHandler interface{}

	quitCh := make(chan bool, 1)

	txHandler = func(tx *core.Transaction) {
		response := &rpcpb.GetNewTransactionsResponse{Transaction: tx.ToProto().(*corepb.Transaction)}
		err := stream.Send(response)
		if err != nil {
			logger.WithError(err).Info("RPCService: failed to send transaction to client.")
			rpcService.node.GetBlockchain().GetTxPool().EventBus.Unsubscribe(core.NewTransactionTopic, txHandler)
			quitCh <- true
		}
	}

	rpcService.node.GetBlockchain().GetTxPool().EventBus.SubscribeAsync(core.NewTransactionTopic, txHandler, false)
	<-quitCh
	return nil
}

func (rpcService *RpcService) IsPrivate() bool { return false }
