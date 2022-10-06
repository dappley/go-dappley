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

type utxoSlice []*utxo.UTXO

func (u utxoSlice) Len() int {
	return len(u)
}

func (u utxoSlice) Less(i, j int) bool {
	if u[i].Value.Cmp(u[j].Value) == -1 {
		return false
	} else {
		return true
	}
}

func (u utxoSlice) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

func sendCommandHandler(ctx context.Context, c interface{}, flags cmdFlags) {
	var data string
	path := *(flags[flagFilePath].(*string))
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
	fromAddress := *(flags[flagFromAddress].(*string))
	if fromAddress == "" {
		fmt.Println("Error: from address is missing!")
		return
	}
	toAddress := *(flags[flagToAddress].(*string))
	if toAddress == "" && data == "" {
		fmt.Println("Error: to address is missing!")
		return
	}

	addressAccount := account.NewTransactionAccountByAddress(account.NewAddress(fromAddress))
	if !addressAccount.IsValid() {
		fmt.Println("Error: 'from' address is not valid!")
		return
	}

	addressAccount = account.NewTransactionAccountByAddress(account.NewAddress(toAddress))
	if !addressAccount.IsValid() && data == "" {
		fmt.Println("Error: 'to' address is not valid!")
		return
	}

	amount := common.NewAmount(0)
	tip := common.NewAmount(0)
	gasLimit := common.NewAmount(0)
	gasPrice := common.NewAmount(0)
	if flags[flagAmount] != nil {
		amount = common.NewAmount(uint64(*(flags[flagAmount].(*int))))
	}
	if flags[flagTip] != nil {
		tip = common.NewAmount(*(flags[flagTip].(*uint64)))
	}
	if flags[flagGasLimit] != nil {
		gasLimit = common.NewAmount(*(flags[flagGasLimit].(*uint64)))
	}
	if flags[flagGasPrice] != nil {
		gasPrice = common.NewAmount(*(flags[flagGasPrice].(*uint64)))
	}

	/*
		response, err := logic.GetUtxoStream(c.(rpcpb.RpcServiceClient), &rpcpb.GetUTXORequest{
			Address: account.NewAddress(*(flags[flagFromAddress].(*string))).String(),
		})
	*/

	targetAmount := amount.Uint64() + tip.Uint64() + gasLimit.Uint64()*gasPrice.Uint64()
	response, err := logic.GetUtxoStreamWithAmount(c.(rpcpb.RpcServiceClient), &rpcpb.GetUTXOWithAmountRequest{
		Address: account.NewAddress(*(flags[flagFromAddress].(*string))).String(),
		Amount:  targetAmount,
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
	tx_utxos, err := getUTXOsfromAmount(inputUtxos, amount, tip, gasLimit, gasPrice)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	senderAccount := am.GetAccountByAddress(account.NewAddress(*(flags[flagFromAddress].(*string))))

	if senderAccount == nil {
		fmt.Println("Error: invalid account address.")
		return
	}
	sendTxParam := transaction.NewSendTxParam(account.NewAddress(*(flags[flagFromAddress].(*string))), senderAccount.GetKeyPair(),
		account.NewAddress(*(flags[flagToAddress].(*string))), amount, tip, gasLimit, gasPrice, data)

	tx, err := ltransaction.NewNormalUTXOTransaction(tx_utxos, sendTxParam)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	sendTransactionRequest := &rpcpb.SendTransactionRequest{Transaction: tx.ToProto().(*transactionpb.Transaction)}
	_, err = c.(rpcpb.RpcServiceClient).RpcSendTransaction(ctx, sendTransactionRequest)

	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", status.Convert(err).Message())
		}
		return
	}

	if *(flags[flagToAddress].(*string)) == "" {
		fmt.Println("Contract address:", tx.Vout[0].GetAddress().String())
	}

	fmt.Println("Transaction is sent! Pending approval from network.")
}
