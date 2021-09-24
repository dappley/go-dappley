package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"

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

func testSigVerCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {
	script, _ := ioutil.ReadFile("contracts/sig_tester.js")
	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	contract := string(script)
	toAddress := account.NewContractTransactionAccount().GetAddress()
	fromAddress := *flags[flagFromAddress].(*string)
	sender := am.GetAccountByAddress(account.NewAddress(fromAddress))
	if !sender.IsValid() {
		fmt.Println("Error: 'from' address is not valid!")
		return
	}

	response, err := logic.GetUtxoStream(a.(rpcpb.RpcServiceClient), &rpcpb.GetUTXORequest{
		Address: sender.GetAddress().String(),
	})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", status.Convert(err).Message())
		}
		return
	}
	utxos := response.GetUtxos()
	var inputUtxos []*utxo.UTXO
	for _, u := range utxos {
		utxo := utxo.UTXO{}
		utxo.FromProto(u)
		inputUtxos = append(inputUtxos, &utxo)
	}
	sort.Sort(utxoSlice(inputUtxos))
	amount := common.NewAmount(0)
	tip := common.NewAmount(0)
	gasLimit := common.NewAmount(100000)
	gasPrice := common.NewAmount(1)
	tx_utxos, err := getUTXOsfromAmount(inputUtxos, amount, tip, gasLimit, gasPrice)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	sendTxParam := transaction.NewSendTxParam(account.NewAddress(sender.GetAddress().String()), sender.GetKeyPair(),
		toAddress, amount, tip, gasLimit, gasPrice, contract)

	tx, err := ltransaction.NewNormalUTXOTransaction(tx_utxos, sendTxParam)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	sendTransactionRequest := &rpcpb.SendTransactionRequest{Transaction: tx.ToProto().(*transactionpb.Transaction)}
	_, err = a.(rpcpb.RpcServiceClient).RpcSendTransaction(ctx, sendTransactionRequest)

	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", status.Convert(err).Message())
		}
		return
	}

	fmt.Println("Contract sent successfully")
}
