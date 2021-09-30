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

func launchVCContractCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {
	script, _ := ioutil.ReadFile("contracts/vc.js")
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

	fmt.Println("Contract sent successfully to ", toAddress)
}
func testCreateSchemaCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {

	script := `{"function": "createSchema", "args": ["{'context': 'stuff', 'id': 'test', 'type': 'good', 'credentialSubject': {'id': 'did:dappley:123456789'}, 'credentialSchema': {'id': 'test','type': 'good'}, 'issuer': 'me', 'issuanceDate': 'now', 'proof': [{'type': 'stillgood', 'created': 'earlier', 'proofPurpose': 'testing', 'verificationMethod': 'whoknows', 'hex': 'f8d99846ca74161de2847041ed3bfd5b79ae0d4f20febfaa8075e0366d6abce42b838306ffac2d8bdbdad3fb943e2431cbf568950ca59da3c218b96795077b6600'}]}"]}`
	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	contract := string(script)
	toAddress := account.NewAddress(*(flags[flagToAddress].(*string)))
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

	fmt.Println("Contract invoked successfully")
}

func testAddVCCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {

	script := `{"function": "addVC", "args": ["{'context': 'vcstuff', 'id': 'test', 'type': 'good', 'credentialSubject': {'id': 'placeholder'}, 'credentialSchema': {'id': 'test','type': 'good'}, 'issuer': 'did:dappley:123456789', 'issuanceDate': 'now', 'proof': [{'type': 'stillgood', 'created': 'earlier', 'proofPurpose': 'testing', 'verificationMethod': 'whoknows', 'hex': 'f8d99846ca74161de2847041ed3bfd5b79ae0d4f20febfaa8075e0366d6abce42b838306ffac2d8bdbdad3fb943e2431cbf568950ca59da3c218b96795077b6600'}]}", "2021-09-27T10:25:04.697700166-07:00"]}`
	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	contract := string(script)
	toAddress := account.NewAddress(*(flags[flagToAddress].(*string)))
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

	fmt.Println("Contract invoked successfully")
}

func testUpdateVCCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {

	script := `{"function": "updateVC", "args": ["{'context': 'vcstuffmodified', 'id': 'test', 'type': 'good', 'credentialSubject': {'id': 'placeholder'}, 'credentialSchema': {'id': 'test','type': 'good'}, 'issuer': 'did:dappley:123456789', 'issuanceDate': 'now', 'proof': [{'type': 'stillgood', 'created': 'earlier', 'proofPurpose': 'testing', 'verificationMethod': 'whoknows', 'hex': 'f8d99846ca74161de2847041ed3bfd5b79ae0d4f20febfaa8075e0366d6abce42b838306ffac2d8bdbdad3fb943e2431cbf568950ca59da3c218b96795077b6600'}]}", "2021-09-27T10:25:04.697700166-07:00"]}`
	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	contract := string(script)
	toAddress := account.NewAddress(*(flags[flagToAddress].(*string)))
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

	fmt.Println("Contract invoked successfully")
}

func testDeleteVCCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {

	script := `{"function": "deleteVC", "args": ["test", "f8d99846ca74161de2847041ed3bfd5b79ae0d4f20febfaa8075e0366d6abce42b838306ffac2d8bdbdad3fb943e2431cbf568950ca59da3c218b96795077b6600", "2021-09-27T10:25:04.697700166-07:00"]}`
	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	contract := string(script)
	toAddress := account.NewAddress(*(flags[flagToAddress].(*string)))
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

	fmt.Println("Contract invoked successfully")
}

func testAddDidCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {

	script := `{"function": "addDID", "args": ["{'id': 'did:dappley:123456789', 'verificationMethod': [{'id': 'did:dappley:123456789#verification', 'controller': 'did:dappley:1234567890', 'type': 'Secp256k1', 'publicKeyHex': '11daebdd3b879b3552da81479f893447338234623c70a2e4c343a419395f7428c87350f52964dc6ef21497f51ee38c7dae83d08e84c6aa9ed3ec35744bc8ac18'}]}"]}`
	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	contract := string(script)
	toAddress := account.NewAddress(*(flags[flagToAddress].(*string)))
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

	fmt.Println("Contract invoked successfully")
}

func testUpdateDidCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {
	script := `{"function": "updateDID", "args": ["did:dappley:123456789", "{'update' : 'successful', 'id': 'did:dappley:123456789', 'verificationMethod': [{'id': 'did:dappley:123456789#verification', 'controller': 'did:dappley:1234567890', 'type': 'Secp256k1', 'publicKeyHex': '11daebdd3b879b3552da81479f893447338234623c70a2e4c343a419395f7428c87350f52964dc6ef21497f51ee38c7dae83d08e84c6aa9ed3ec35744bc8ac18'}]}", "2021-09-27T10:25:04.697700166-07:00", "f8d99846ca74161de2847041ed3bfd5b79ae0d4f20febfaa8075e0366d6abce42b838306ffac2d8bdbdad3fb943e2431cbf568950ca59da3c218b96795077b6600"]}`
	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	contract := string(script)
	toAddress := account.NewAddress(*(flags[flagToAddress].(*string)))
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

	fmt.Println("Contract invoked successfully")
}

func testDeleteDidCommandHandler(ctx context.Context, a interface{}, flags cmdFlags) {
	script := `{"function": "deleteDID", "args": ["did:dappley:123456789", "2021-09-27T10:25:04.697700166-07:00", "f8d99846ca74161de2847041ed3bfd5b79ae0d4f20febfaa8075e0366d6abce42b838306ffac2d8bdbdad3fb943e2431cbf568950ca59da3c218b96795077b6600"]}`
	am, err := logic.GetAccountManager(wallet.GetAccountFilePath())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	contract := string(script)
	toAddress := account.NewAddress(*(flags[flagToAddress].(*string)))
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

	fmt.Println("Contract invoked successfully")
}
