package main

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/core/transaction"
	transactionpb "github.com/dappley/go-dappley/core/transaction/pb"
	"github.com/dappley/go-dappley/core/utxo"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/logic/ltransaction"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/wallet"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func estimateGasCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {
	var data string
	path := *(flags[flagFilePath].(*string))
	fromAddress := *(flags[flagFromAddress].(*string))
	fromAccount := account.NewTransactionAccountByAddress(account.NewAddress(fromAddress))
	toAddress := *(flags[flagToAddress].(*string))
	toAccount := account.NewTransactionAccountByAddress(account.NewAddress(toAddress))
	if path == "" {
		data = *(flags[flagData].(*string))
	} else {
		script, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("Error: smart contract path \"%s\" is invalid.\n", path)
			return
		}
		data = string(script)
	}

	if !fromAccount.IsValid() {
		fmt.Println("Error: 'from' address is not valid!")
		return
	}

	//Contract deployment transaction does not need to validate to address
	if data == "" && !toAccount.IsValid() {
		fmt.Println("Error: 'to' address is not valid!")
		return
	}
	response, err := logic.GetUtxoStream(c.(rpcpb.RpcServiceClient), &rpcpb.GetUTXORequest{
		Address: account.NewAddress(*(flags[flagFromAddress].(*string))).String(),
	})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error:", status.Convert(err).Message())
		}
		return
	}
	utxos := response.GetUtxos()
	var InputUtxos []*utxo.UTXO
	for _, u := range utxos {
		uu := utxo.UTXO{}
		uu.Value = common.NewAmountFromBytes(u.Amount)
		uu.Txid = u.Txid
		uu.PubKeyHash = account.PubKeyHash(u.PublicKeyHash)
		uu.TxIndex = int(u.TxIndex)
		InputUtxos = append(InputUtxos, &uu)
	}
	tip := common.NewAmount(0)
	gasLimit := common.NewAmount(0)
	gasPrice := common.NewAmount(0)
	tx_utxos, err := getUTXOsfromAmount(InputUtxos, common.NewAmount(uint64(*(flags[flagAmount].(*int)))), tip, gasLimit, gasPrice)
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}

	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}
	senderAccount := am.GetAccountByAddress(account.NewAddress(*(flags[flagFromAddress].(*string))))

	if senderAccount == nil {
		fmt.Println("Error: invalid account address.")
		return
	}
	sendTxParam := transaction.NewSendTxParam(account.NewAddress(*(flags[flagFromAddress].(*string))), senderAccount.GetKeyPair(),
		account.NewAddress(*(flags[flagToAddress].(*string))), common.NewAmount(uint64(*(flags[flagAmount].(*int)))), tip, gasLimit, gasPrice, data)
	tx, err := ltransaction.NewNormalUTXOTransaction(tx_utxos, sendTxParam)
	estimateGasRequest := &rpcpb.EstimateGasRequest{Transaction: tx.ToProto().(*transactionpb.Transaction)}
	gasResponse, err := c.(rpcpb.RpcServiceClient).RpcEstimateGas(ctx, estimateGasRequest)

	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", status.Convert(err).Message())
		}
		return
	}

	gasCount := gasResponse.GasCount

	fmt.Println("Gas estimation: ", common.NewAmountFromBytes(gasCount).String())
}
