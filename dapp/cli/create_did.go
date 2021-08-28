package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"time"

	"github.com/dappley/go-dappley/common"
	acc "github.com/dappley/go-dappley/core/account"
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

func launchDIDSystem(sender *acc.Account, account interface{}, ctx context.Context, dm *wallet.DIDManager) {
	script, _ := ioutil.ReadFile("contracts/did.js")

	contract := string(script)
	toAddress := acc.NewContractTransactionAccount().GetAddress()

	if sender == nil {
		fmt.Println("Error: invalid account address.")
		return
	}
	response, err := logic.GetUtxoStream(account.(rpcpb.RpcServiceClient), &rpcpb.GetUTXORequest{
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
	sendTxParam := transaction.NewSendTxParam(acc.NewAddress(sender.GetAddress().String()), sender.GetKeyPair(),
		toAddress, amount, tip, gasLimit, gasPrice, contract)

	tx, err := ltransaction.NewNormalUTXOTransaction(tx_utxos, sendTxParam)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	sendTransactionRequest := &rpcpb.SendTransactionRequest{Transaction: tx.ToProto().(*transactionpb.Transaction)}
	_, err = account.(rpcpb.RpcServiceClient).RpcSendTransaction(ctx, sendTransactionRequest)

	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", status.Convert(err).Message())
		}
		return
	}

	dm.SystemAddress = toAddress
	dm.SaveDIDsToFile()
	time.Sleep(10 * time.Second)
	fmt.Println("Launched the system! Address is ", dm.SystemAddress)
}

func createDIDCommandHandler(ctx context.Context, account interface{}, flags cmdFlags) {
	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	senderAccount := am.GetAccountByAddress(acc.NewAddress(*(flags[flagFromAddress].(*string))))
	if senderAccount == nil {
		fmt.Println("Error: invalid account address.")
		return
	}

	dm, err := logic.GetDIDManager(wallet.GetDIDFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	if dm.SystemAddress.String() == "" {
		fmt.Println("DID system not found, launching...")
		launchDIDSystem(senderAccount, account, ctx, dm)
	}
	fmt.Println("system address: ", dm.SystemAddress.String())
	response, err := logic.GetUtxoStream(account.(rpcpb.RpcServiceClient), &rpcpb.GetUTXORequest{
		Address: acc.NewAddress(*(flags[flagFromAddress].(*string))).String(),
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

	tx_utxos, err := getUTXOsfromAmount(inputUtxos, common.NewAmount(0), tip, gasLimit, gasPrice)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	didSet := acc.NewDID()
	if !acc.CheckDIDFormat(didSet.DID) {
		fmt.Println("DID formatted incorrectly")
		return
	}
	toAddress := dm.SystemAddress
	didDocument := acc.CreateDIDDocument(didSet)
	fmt.Println("New did is", didSet.DID)

	contract := `{"function": "new_DIDDocument", "args": ["` + didDocument.ID + `", "[`
	for _, vm := range didDocument.VerificationMethods {
		contract += `['` + vm.ID + `','` + vm.MethodType + `','` + vm.Controller + `','` + vm.Key + `', ]`
	}
	contract += `]", "[`
	for _, am := range didDocument.AuthenticationMethods {
		contract += `['` + am.ID + `','` + am.MethodType + `','` + am.Controller + `','` + am.Key + `', ]`
	}
	contract += `]"]}`

	fmt.Println(toAddress)

	sendTxParam := transaction.NewSendTxParam(acc.NewAddress(*(flags[flagFromAddress].(*string))), senderAccount.GetKeyPair(),
		toAddress, amount, tip, gasLimit, gasPrice, contract)

	tx, err := ltransaction.NewNormalUTXOTransaction(tx_utxos, sendTxParam)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	sendTransactionRequest := &rpcpb.SendTransactionRequest{Transaction: tx.ToProto().(*transactionpb.Transaction)}
	_, err = account.(rpcpb.RpcServiceClient).RpcSendTransaction(ctx, sendTransactionRequest)

	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error: ", status.Convert(err).Message())
		}
		return
	}

	dm.AddDID(didSet)
	dm.SaveDIDsToFile()
	fmt.Println("Operation complete!")
}
