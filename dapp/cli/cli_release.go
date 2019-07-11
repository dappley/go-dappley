// +build release

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
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	clientpkg "github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/core/pb"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/golang/protobuf/proto"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

//command names
const (
	cliGetBlocks         = "getBlocks"
	cliGetBlockchainInfo = "getBlockchainInfo"
	cliGetBalance        = "getBalance"
	cliGetPeerInfo       = "getPeerInfo"
	cliSend              = "send"
	cliAddPeer           = "addPeer"
	clicreateWallet      = "createWallet"
	cliListAddresses     = "listAddresses"
	cliaddProducer       = "addProducer"
	cliHelp              = "help"
)

//flag names
const (
	flagStartBlockHashes = "startBlockHashes"
	flagBlockMaxCount    = "maxCount"
	flagAddress          = "address"
	flagAddressBalance   = "to"
	flagAmountBalance    = "amount"
	flagTip              = "tip"
	flagToAddress        = "to"
	flagFromAddress      = "from"
	flagAmount           = "amount"
	flagData             = "data"
	flagFilePath         = "file"
	flagPeerFullAddr     = "peerFullAddr"
	flagProducerAddr     = "address"
	flagListPrivateKey   = "privateKey"
	flagGasLimit         = "gasLimit"
	flagGasPrice         = "gasPrice"
)

type valueType int

//type enum
const (
	valueTypeInt = iota
	valueTypeString
	boolType
	valueTypeUint64
)

type serviceType int

const (
	rpcService = iota
	adminRpcService
)

//list of commands
var cmdList = []string{
	cliGetBlocks,
	cliGetBlockchainInfo,
	cliGetBalance,
	cliGetPeerInfo,
	cliSend,
	cliAddPeer,
	clicreateWallet,
	cliListAddresses,
	cliHelp,
}

var (
	ErrInsufficientFund = errors.New("cli: the balance is insufficient")
)

//configure input parameters/flags for each command
var cmdFlagsMap = map[string][]flagPars{
	cliGetBlocks: {
		flagPars{
			flagBlockMaxCount,
			0,
			valueTypeInt,
			"maxCount. Eg. 500",
		},
		flagPars{
			flagStartBlockHashes,
			"",
			valueTypeString,
			"startBlockHashes. Eg. \"8334b4c19091ae7582506eec5b84bfeb4a5e101042e40b403490c4ceb33897ba, " +
				"8334b4c19091ae7582506eec5b84bfeb4a5e101042e40b403490c4ceb33897bb\"(no space)",
		},
	},
	cliGetBalance: {flagPars{
		flagAddress,
		"",
		valueTypeString,
		"Address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
	}},
	cliSend: {
		flagPars{
			flagFromAddress,
			"",
			valueTypeString,
			"Sender's wallet address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
		},
		flagPars{
			flagToAddress,
			"",
			valueTypeString,
			"Receiver's wallet address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
		},
		flagPars{
			flagAmount,
			0,
			valueTypeInt,
			"The amount to send from the sender to the receiver.",
		},
		flagPars{
			flagTip,
			uint64(0),
			valueTypeUint64,
			"Tip to miner.",
		},
		flagPars{
			flagData,
			"",
			valueTypeString,
			"Smart contract in JavaScript. Eg. helloworld!",
		},
		flagPars{
			flagFilePath,
			"",
			valueTypeString,
			"Smart contract file path. Eg. contract/smart_contract.js",
		},
		flagPars{
			flagGasLimit,
			uint64(0),
			valueTypeUint64,
			"Gas limit count of smart contract execution.",
		},
		flagPars{
			flagGasPrice,
			uint64(0),
			valueTypeUint64,
			"Gas price of smart contract execution.",
		},
	},
	cliAddPeer: {flagPars{
		flagPeerFullAddr,
		"",
		valueTypeString,
		"Full Address. Eg. /ip4/127.0.0.1/tcp/12345/ipfs/QmT5oB6xHSunc64Aojoxa6zg9uH31ajiAVyNfCdBZiwFTV",
	}},
	cliListAddresses: {flagPars{
		flagListPrivateKey,
		false,
		boolType,
		"with/without this optional argument to display the private keys or not",
	}},
}

//map the callback function to each command
var cmdHandlers = map[string]commandHandlersWithType{
	cliGetBlocks:         {rpcService, getBlocksCommandHandler},
	cliGetBlockchainInfo: {rpcService, getBlockchainInfoCommandHandler},
	cliGetBalance:        {rpcService, getBalanceCommandHandler},
	cliGetPeerInfo:       {adminRpcService, getPeerInfoCommandHandler},
	cliSend:              {rpcService, sendCommandHandler},
	cliAddPeer:           {adminRpcService, addPeerCommandHandler},
	clicreateWallet:      {adminRpcService, createWalletCommandHandler},
	cliListAddresses:     {adminRpcService, listAddressesCommandHandler},
	cliHelp:              {adminRpcService, helpCommandHandler},
}

type commandHandlersWithType struct {
	serviceType serviceType
	cmdHandler  commandHandler
}

type commandHandler func(ctx context.Context, client interface{}, flags cmdFlags)

type flagPars struct {
	name         string
	defaultValue interface{}
	valueType    valueType
	usage        string
}

//map key: flag name   map defaultValue: flag defaultValue
type cmdFlags map[string]interface{}

func main() {

	var filePath string
	flag.StringVar(&filePath, "f", "default.conf", "CLI config file path")
	flag.Parse()

	cliConfig := &configpb.CliConfig{}
	config.LoadConfig(filePath, cliConfig)

	conn := initRpcClient(int(cliConfig.GetPort()))
	defer conn.Close()
	clients := map[serviceType]interface{}{
		rpcService:      rpcpb.NewRpcServiceClient(conn),
		adminRpcService: rpcpb.NewAdminServiceClient(conn),
	}
	args := os.Args[1:]

	if len(args) < 1 {
		printUsage()
		return
	}

	if args[0] == "-f" {
		args = args[2:]
	}

	cmdFlagSetList := map[string]*flag.FlagSet{}
	//set up flagset for each command
	for _, cmd := range cmdList {
		fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
		cmdFlagSetList[cmd] = fs
	}

	cmdFlagValues := map[string]cmdFlags{}
	//set up flags for each command
	for cmd, pars := range cmdFlagsMap {
		cmdFlagValues[cmd] = cmdFlags{}
		for _, par := range pars {
			switch par.valueType {
			case valueTypeInt:
				cmdFlagValues[cmd][par.name] = cmdFlagSetList[cmd].Int(par.name, par.defaultValue.(int), par.usage)
			case valueTypeString:
				cmdFlagValues[cmd][par.name] = cmdFlagSetList[cmd].String(par.name, par.defaultValue.(string), par.usage)
			case boolType:
				cmdFlagValues[cmd][par.name] = cmdFlagSetList[cmd].Bool(par.name, par.defaultValue.(bool), par.usage)
			case valueTypeUint64:
				cmdFlagValues[cmd][par.name] = cmdFlagSetList[cmd].Uint64(par.name, par.defaultValue.(uint64), par.usage)
			}
		}
	}

	cmdName := args[0]

	cmd := cmdFlagSetList[cmdName]
	if cmd == nil {
		fmt.Println("\nError:", cmdName, "is an invalid command")
		printUsage()
	} else {
		err := cmd.Parse(args[1:])
		if err != nil {
			return
		}
		if cmd.Parsed() {
			md := metadata.Pairs("password", cliConfig.GetPassword())
			ctx := metadata.NewOutgoingContext(context.Background(), md)
			cmdHandlers[cmdName].cmdHandler(ctx, clients[cmdHandlers[cmdName].serviceType], cmdFlagValues[cmdName])
		}
	}

}

func printUsage() {
	fmt.Println("Usage:")
	for _, cmd := range cmdList {
		fmt.Println(" ", cmd)
	}
	fmt.Println("Note: Use the command 'cli help' to get the command usage in details")
}

func getBlocksCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	maxCount := int32(*(flags[flagBlockMaxCount].(*int)))
	if maxCount <= 0 {
		fmt.Println("\n Example: cli getBlocks -startBlockHashes 10 -maxCount 5")
		fmt.Println()
		return
	}

	getBlocksRequest := &rpcpb.GetBlocksRequest{MaxCount: maxCount}

	// set startBlockHashes of getBlocksRequest if specified in flag
	startBlockHashesString := string(*(flags[flagStartBlockHashes].(*string)))
	if len(startBlockHashesString) > 0 {
		var startBlockHashes [][]byte
		for _, startBlockHash := range strings.Split(startBlockHashesString, ",") {
			startBlockHashInByte, err := hex.DecodeString(startBlockHash)
			if err != nil {
				fmt.Println("Error:", err.Error())
				return
			}
			startBlockHashes = append(startBlockHashes, startBlockHashInByte)
		}
		getBlocksRequest = &rpcpb.GetBlocksRequest{MaxCount: maxCount, StartBlockHashes: startBlockHashes}
	}

	response, err := client.(rpcpb.RpcServiceClient).RpcGetBlocks(ctx, getBlocksRequest)
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error:", status.Convert(err).Message())
		}
		return
	}

	var encodedBlocks []map[string]interface{}
	for _, block := range response.Blocks {

		var encodedTransactions []map[string]interface{}

		for _, transaction := range block.GetTransactions() {

			var encodedVin []map[string]interface{}
			for _, vin := range transaction.GetVin() {
				encodedVin = append(encodedVin, map[string]interface{}{
					"Vout":      vin.GetVout(),
					"Signature": hex.EncodeToString(vin.GetSignature()),
					"PubKey":    string(vin.GetPublicKey()),
				})
			}

			var encodedVout []map[string]interface{}
			for _, vout := range transaction.GetVout() {
				encodedVout = append(encodedVout, map[string]interface{}{
					"Value":      string(vout.GetValue()),
					"PubKeyHash": hex.EncodeToString(vout.GetPublicKeyHash()),
					"Contract":   vout.GetContract(),
				})
			}

			encodedTransaction := map[string]interface{}{
				"ID":   hex.EncodeToString(transaction.GetId()),
				"Vin":  encodedVin,
				"Vout": encodedVout,
			}
			encodedTransactions = append(encodedTransactions, encodedTransaction)
		}

		encodedBlock := map[string]interface{}{
			"Header": map[string]interface{}{
				"Hash":      hex.EncodeToString(block.GetHeader().GetHash()),
				"Prevhash":  hex.EncodeToString(block.GetHeader().GetPreviousHash()),
				"Timestamp": block.GetHeader().GetTimestamp(),
				"Sign":      hex.EncodeToString(block.GetHeader().GetSignature()),
				"height":    block.GetHeader().GetHeight(),
			},
			"Transactions": encodedTransactions,
		}

		encodedBlocks = append(encodedBlocks, encodedBlock)
	}

	blocks, err := json.MarshalIndent(encodedBlocks, "", "  ")
	if err != nil {
		fmt.Println("Error:", err.Error())
	}

	fmt.Println(string(blocks))
}

func getBlockchainInfoCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	response, err := client.(rpcpb.RpcServiceClient).RpcGetBlockchainInfo(ctx, &rpcpb.GetBlockchainInfoRequest{})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error:", status.Convert(err).Message())
		}
		return
	}
	encodedResponse := map[string]interface{}{
		"TailBlockHash": hex.EncodeToString(response.TailBlockHash),
		"BlockHeight":   response.BlockHeight,
		"Producers":     response.Producers,
	}

	blockchainInfo, err := json.MarshalIndent(encodedResponse, "", "  ")
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}

	fmt.Println(string(blockchainInfo))
}

func getBalanceCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	if len(*(flags[flagAddress].(*string))) == 0 {
		printUsage()
		fmt.Println("\n Example: cli getBalance -address 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7")
		fmt.Println()
		return
	}

	address := *(flags[flagAddress].(*string))
	if core.NewAddress(address).IsValid() == false {
		fmt.Println("Error: address is not valid")
		return
	}

	response, err := client.(rpcpb.RpcServiceClient).RpcGetBalance(ctx, &rpcpb.GetBalanceRequest{Address: address})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error:", status.Convert(err).Message())
		}
		return
	}
	fmt.Printf("The balance is: %d\n", response.GetAmount())
}

func createWalletCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	empty, err := logic.IsWalletEmpty()
	prompter := util.NewTerminalPrompter()
	passphrase := ""
	if empty {
		passphrase = prompter.GetPassPhrase("Please input the password for the new wallet: ", true)
		if passphrase == "" {
			fmt.Println("Error: password cannot be empty!")
			return
		}
		wallet, err := logic.CreateWalletWithpassphrase(passphrase)
		if err != nil {
			fmt.Println("Error:", err.Error())
			return
		}
		if wallet != nil {
			fmt.Printf("Wallet is created. The address is %s \n", wallet.GetAddress().Address)
			return
		}
	}

	locked, err := logic.IsWalletLocked()
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}

	if locked {
		passphrase = prompter.GetPassPhrase("Please input the password: ", false)
		if passphrase == "" {
			fmt.Println("Error: password should not be empty!")
			return
		}
		wallet, err := logic.CreateWalletWithpassphrase(passphrase)
		if err != nil {
			fmt.Println("Error:", err.Error())
			return
		}
		if wallet != nil {
			fmt.Printf("Wallet is created. The address is %s\n", wallet.GetAddress().Address)
		}
		//unlock the wallet
		_, err = client.(rpcpb.AdminServiceClient).RpcUnlockWallet(ctx, &rpcpb.UnlockWalletRequest{})

		if err != nil {
			switch status.Code(err) {
			case codes.Unavailable:
				fmt.Println("Error: server is not reachable!")
			default:
				fmt.Println("Error:", status.Convert(err).Message())
			}
			return
		}
	} else {
		wallet, err := logic.AddWallet()
		if err != nil {
			fmt.Println("Error:", err.Error())
			return
		}
		if wallet != nil {
			fmt.Printf("Wallet is created. The address is %s\n", wallet.GetAddress().Address)
		}
	}

	return
}

func listAddressesCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	listPriv := false
	if flags[flagListPrivateKey] == nil {
		return
	} else if *(flags[flagListPrivateKey].(*bool)) {
		listPriv = true
	} else {
		listPriv = false
	}

	passphrase := ""
	prompter := util.NewTerminalPrompter()

	empty, err := logic.IsWalletEmpty()
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}
	if empty {
		fmt.Println("Please use cli createWallet to generate a wallet first!")
		return
	}

	locked, err := logic.IsWalletLocked()
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}
	if locked {
		passphrase = prompter.GetPassPhrase("Please input the password: ", false)
		if passphrase == "" {
			fmt.Println("Password should not be empty!")
			return
		}
		fl := storage.NewFileLoader(clientpkg.GetWalletFilePath())
		wm := clientpkg.NewWalletManager(fl)
		err := wm.LoadFromFile()
		addressList, err := wm.GetAddressesWithPassphrase(passphrase)
		if err != nil {
			fmt.Println("Error:", err.Error())
			return
		}
		//unlock the wallet
		_, err = client.(rpcpb.AdminServiceClient).RpcUnlockWallet(ctx, &rpcpb.UnlockWalletRequest{})
		if err != nil {
			switch status.Code(err) {
			case codes.Unavailable:
				fmt.Println("Error: server is not reachable!")
			default:
				fmt.Println("Error:", status.Convert(err).Message())
			}
		}
		if !listPriv {
			if len(addressList) == 0 {
				fmt.Println("The addresses in the wallet is empty!")
			} else {
				i := 1
				fmt.Println("The address list:")
				for _, addr := range addressList {
					fmt.Printf("Address[%d]: %s\n", i, addr)
					i++
				}
				fmt.Println()
				fmt.Println("Use the command 'cli listAddresses -privateKey' to list the addresses with private keys")
			}
		} else {
			privateKeyList := []string{}
			for _, addr := range addressList {
				keyPair := wm.GetKeyPairByAddress(core.NewAddress(addr))
				privateKey, err1 := secp256k1.FromECDSAPrivateKey(&keyPair.PrivateKey)
				if err1 != nil {
					err = err1
					return
				}
				privateKeyList = append(privateKeyList, hex.EncodeToString(privateKey))
				err = err1
			}
			if len(addressList) == 0 {
				fmt.Println("The addresses in the wallet is empty!")
			} else {
				i := 1
				fmt.Println("The address list with private keys:")
				for _, addr := range addressList {
					fmt.Println("--------------------------------------------------------------------------------")
					fmt.Printf("Address[%d]: %s \nPrivate Key[%d]: %s", i, addr, i, privateKeyList[i-1])
					fmt.Println()
					i++
				}
				fmt.Println("--------------------------------------------------------------------------------")
			}

		}
	} else {
		fl := storage.NewFileLoader(clientpkg.GetWalletFilePath())
		wm := clientpkg.NewWalletManager(fl)
		err := wm.LoadFromFile()
		if err != nil {
			fmt.Println("Error:", err.Error())
			return
		}
		addressList := wm.GetAddresses()
		if !listPriv {
			if len(addressList) == 0 {
				fmt.Println("The addresses in the wallet is empty!")
			} else {
				i := 1
				fmt.Println("The address list:")
				for _, addr := range addressList {
					fmt.Printf("Address[%d]: %s\n", i, addr.Address)
					i++
				}
				fmt.Println()
				fmt.Println("Use the command 'cli listAddresses -privateKey' to list the addresses with private keys")
			}
		} else {
			privateKeyList := []string{}
			for _, addr := range addressList {
				keyPair := wm.GetKeyPairByAddress(addr)
				privateKey, err1 := secp256k1.FromECDSAPrivateKey(&keyPair.PrivateKey)
				if err1 != nil {
					err = err1
					return
				}
				privateKeyList = append(privateKeyList, hex.EncodeToString(privateKey))
				err = err1
			}
			if len(addressList) == 0 {
				fmt.Println("The addresses in the wallet is empty!")
			} else {
				i := 1
				fmt.Println("The address list with private keys:")
				for _, addr := range addressList {
					fmt.Println("--------------------------------------------------------------------------------")
					fmt.Printf("Address[%d]: %s \nPrivate Key[%d]: %s", i, addr.Address, i, privateKeyList[i-1])
					fmt.Println()
					i++
				}
				fmt.Println("--------------------------------------------------------------------------------")
			}

		}

	}
	return
}

func getPeerInfoCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	response, err := client.(rpcpb.AdminServiceClient).RpcGetPeerInfo(ctx, &rpcpb.GetPeerInfoRequest{})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error:", err.Error())
		}
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}

func sendCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
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

	if core.NewAddress(*(flags[flagFromAddress].(*string))).IsValid() == false {
		fmt.Println("Error: 'from' address is not valid!")
		return
	}

	//Contract deployment transaction does not need to validate to address
	if data == "" && core.NewAddress(*(flags[flagToAddress].(*string))).IsValid() == false {
		fmt.Println("Error: 'to' address is not valid!")
		return
	}

	response, err := client.(rpcpb.RpcServiceClient).RpcGetUTXO(ctx, &rpcpb.GetUTXORequest{
		Address: core.NewAddress(*(flags[flagFromAddress].(*string))).Address,
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
	var InputUtxos []*core.UTXO
	for _, u := range utxos {
		uu := core.UTXO{}
		uu.Value = common.NewAmountFromBytes(u.Amount)
		uu.Txid = u.Txid
		uu.PubKeyHash = core.PubKeyHash(u.PublicKeyHash)
		uu.TxIndex = int(u.TxIndex)
		InputUtxos = append(InputUtxos, &uu)
	}
	tip := common.NewAmount(0)
	gasLimit := common.NewAmount(0)
	gasPrice := common.NewAmount(0)
	if flags[flagTip] != nil {
		tip = common.NewAmount(*(flags[flagTip].(*uint64)))
	}
	if flags[flagGasLimit] != nil {
		gasLimit = common.NewAmount(*(flags[flagGasLimit].(*uint64)))
	}
	if flags[flagGasPrice] != nil {
		gasPrice = common.NewAmount(*(flags[flagGasPrice].(*uint64)))
	}
	tx_utxos, err := GetUTXOsfromAmount(InputUtxos, common.NewAmount(uint64(*(flags[flagAmount].(*int)))), tip, gasLimit, gasPrice)
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}

	wm, err := logic.GetWalletManager(clientpkg.GetWalletFilePath())
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}
	senderWallet := wm.GetWalletByAddress(core.NewAddress(*(flags[flagFromAddress].(*string))))

	if senderWallet == nil {
		fmt.Println("Error: invalid wallet address.")
		return
	}
	sendTxParam := core.NewSendTxParam(core.NewAddress(*(flags[flagFromAddress].(*string))), senderWallet.GetKeyPair(),
		core.NewAddress(*(flags[flagToAddress].(*string))), common.NewAmount(uint64(*(flags[flagAmount].(*int)))), tip, gasLimit, gasPrice, data)
	tx, err := core.NewUTXOTransaction(tx_utxos, sendTxParam)

	sendTransactionRequest := &rpcpb.SendTransactionRequest{Transaction: tx.ToProto().(*corepb.Transaction)}
	_, err = client.(rpcpb.RpcServiceClient).RpcSendTransaction(ctx, sendTransactionRequest)

	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error:", status.Convert(err).Message())
		}
		return
	}

	if *(flags[flagToAddress].(*string)) == "" {
		fmt.Println("Contract address:", tx.Vout[0].PubKeyHash.GenerateAddress().String())
	}

	fmt.Println("Transaction is sent! Pending approval from network.")
}

func GetUTXOsfromAmount(inputUTXOs []*core.UTXO, amount *common.Amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount) ([]*core.UTXO, error) {
	if tip != nil {
		amount = amount.Add(tip)
	}
	if gasLimit != nil {
		limitedFee := gasLimit.Mul(gasPrice)
		amount = amount.Add(limitedFee)
	}
	var retUtxos []*core.UTXO
	sum := common.NewAmount(0)
	for _, u := range inputUTXOs {
		sum = sum.Add(u.Value)
		retUtxos = append(retUtxos, u)
		if sum.Cmp(amount) >= 0 {
			break
		}
	}

	if sum.Cmp(amount) < 0 {
		return nil, ErrInsufficientFund
	}

	return retUtxos, nil
}

func helpCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("Command: cli ", "createWallet")
	fmt.Println("Usage Example: cli createWallet")
	for cmd, pars := range cmdFlagsMap {
		fmt.Println("-----------------------------------------------------------------")
		fmt.Println("Command: cli ", cmd)
		fmt.Printf("Usage Example: cli %s", cmd)
		for _, par := range pars {
			fmt.Printf(" -%s", par.name)
			if par.name == flagFromAddress {
				fmt.Printf(" dWRFRFyientRqAbAmo6bskp9sBCTyFHLqF ")
				continue
			}
			if par.name == flagData {
				fmt.Printf(" helloworld! ")
				continue
			}
			if par.name == flagStartBlockHashes {

				fmt.Printf(" 8334b4c19091ae7582506eec5b84bfeb4a5e101042e40b403490c4ceb33897ba, 8334b4c19091ae7582506eec5b84bfeb4a5e101042e40b403490c4ceb33897bb ")
				continue
			}
			if par.name == flagPeerFullAddr {
				fmt.Printf(" /ip4/127.0.0.1/tcp/12345/ipfs/QmT5oB6xHSunc64Aojoxa6zg9uH31ajiAVyNfCdBZiwFTV ")
				continue
			}
			switch par.valueType {
			case valueTypeInt:
				fmt.Printf(" 10 ")
			case valueTypeString:
				fmt.Printf(" 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7 ")
			case valueTypeUint64:
				fmt.Printf(" 50 ")
			}

		}
		fmt.Println()
		fmt.Println("Arguments:")
		for _, par := range pars {
			fmt.Println(par.name, "\t", par.usage)
		}
		fmt.Println()
	}
}

func addPeerCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	req := &rpcpb.AddPeerRequest{
		FullAddress: *(flags[flagPeerFullAddr].(*string)),
	}
	response, err := client.(rpcpb.AdminServiceClient).RpcAddPeer(ctx, req)
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			fmt.Println("Error: server is not reachable!")
		default:
			fmt.Println("Error:", status.Convert(err).Message())
		}
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}

func initRpcClient(port int) *grpc.ClientConn {
	//prepare grpc client
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(fmt.Sprint(":", port), grpc.WithInsecure())
	if err != nil {
		logger.Panic("Error:", err.Error())
	}
	return conn
}
