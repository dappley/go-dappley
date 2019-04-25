package tool

import (
	"context"
	"fmt"
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/storage"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"time"
)

const (
	fundTimeout = time.Duration(time.Minute * 5)
)

func FundFromMiner(adminClient rpcpb.AdminServiceClient, rpcClient rpcpb.RpcServiceClient, fundAddr string, amount *common.Amount) {
	logger.Info("Requesting fund from miner...")

	if fundAddr == "" {
		logger.Panic("There is no wallet to receive fund.")
	}

	requestFundFromMiner(adminClient, fundAddr, amount)
	bal, isSufficient := checkSufficientInitialAmount(rpcClient, fundAddr, amount)
	if isSufficient {
		//continue if the initial amount is sufficient
		return
	}
	logger.WithFields(logger.Fields{
		"address":     fundAddr,
		"balance":     bal,
		"target_fund": amount.String(),
	}).Info("Current wallet balance is insufficient. Waiting for more funds...")
	waitTilInitialAmountIsSufficient(adminClient, rpcClient, fundAddr, amount)
}

func checkSufficientInitialAmount(rpcClient rpcpb.RpcServiceClient, addr string, amount *common.Amount) (uint64, bool) {
	balance, err := GetBalance(rpcClient, addr)
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"address": addr,
		}).Panic("Failed to get balance.")
	}
	return uint64(balance), uint64(balance) >= amount.Uint64()
}

func waitTilInitialAmountIsSufficient(adminClient rpcpb.AdminServiceClient, rpcClient rpcpb.RpcServiceClient, addr string, amount *common.Amount) {
	checkBalanceTicker := time.NewTicker(time.Second * 5).C
	timeout := time.NewTicker(fundTimeout).C
	for {
		select {
		case <-checkBalanceTicker:
			bal, isSufficient := checkSufficientInitialAmount(rpcClient, addr, amount)
			if isSufficient {
				//continue if the initial amount is sufficient
				return
			}
			logger.WithFields(logger.Fields{
				"address":     addr,
				"balance":     bal,
				"target_fund": amount,
			}).Info("Current wallet balance is insufficient. Waiting for more funds...")
			requestFundFromMiner(adminClient, addr, amount)
		case <-timeout:
			logger.WithFields(logger.Fields{
				"target_fund": amount,
			}).Panic("Timed out while waiting for sufficient fund from miner!")
		}
	}
}

func requestFundFromMiner(adminClient rpcpb.AdminServiceClient, fundAddr string, amount *common.Amount) {

	sendFromMinerRequest := &rpcpb.SendFromMinerRequest{To: fundAddr, Amount: amount.Bytes()}
	adminClient.RpcSendFromMiner(context.Background(), sendFromMinerRequest)
}

func GetBalance(rpcClient rpcpb.RpcServiceClient, address string) (int64, error) {
	response, err := rpcClient.RpcGetBalance(context.Background(), &rpcpb.GetBalanceRequest{Address: address})
	return response.Amount, err
}

func GetBlockHeight(rpcClient rpcpb.RpcServiceClient) uint64 {
	resp, err := rpcClient.RpcGetBlockchainInfo(
		context.Background(),
		&rpcpb.GetBlockchainInfoRequest{})
	if err != nil {
		logger.WithError(err).Panic("Cannot get block height.")
	}
	return resp.BlockHeight
}

func InitRpcClient(port int) *grpc.ClientConn {
	//prepare grpc client
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(fmt.Sprint(":", port), grpc.WithInsecure())
	if err != nil {
		logger.WithError(err).Panic("Connection to RPC server failed.")
	}
	return conn
}

func UpdateUtxoIndex(serviceClient rpcpb.RpcServiceClient, addrs []core.Address) *core.UTXOIndex {
	wm, err := logic.GetWalletManager(client.GetWalletFilePath())
	if err != nil {
		logger.WithError(err).Panic("updateUtxoIndex: Unable to get wallet")
	}

	utxoIndex := core.NewUTXOIndex(core.NewUTXOCache(storage.NewRamStorage()))

	for _, addr := range addrs {
		kp := wm.GetKeyPairByAddress(addr)
		_, err := core.NewUserPubKeyHash(kp.PublicKey)
		if err != nil {
			logger.WithError(err).Panic("updateUtxoIndex: Unable to get public key hash")
		}

		utxos := getUtxoByAddr(serviceClient, addr)
		for _, utxoPb := range utxos {
			utxo := core.UTXO{}
			utxo.FromProto(utxoPb)
			utxoIndex.AddUTXO(utxo.TXOutput, utxo.Txid, utxo.TxIndex)
		}
	}
	return utxoIndex
}

func getUtxoByAddr(serviceClient rpcpb.RpcServiceClient, addr core.Address) []*corepb.Utxo {
	resp, err := serviceClient.RpcGetUTXO(context.Background(), &rpcpb.GetUTXORequest{
		Address: addr.String(),
	})
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"addr": addr.String(),
		}).Error("Can not update utxo")
	}
	return resp.Utxos
}
