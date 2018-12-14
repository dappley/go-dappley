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
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"errors"
	clientpkg "github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"github.com/dappley/go-dappley/core/pb"
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
	flagContract         = "contract"
	flagPeerFullAddr     = "peerFullAddr"
)

type valueType int

//type enum
const (
	valueTypeInt = iota
	valueTypeString
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
			flagContract,
			"",
			valueTypeString,
			"Smart contract in JavaScript. Eg. helloworld!",
		},
	},
	cliAddPeer: {flagPars{
		flagPeerFullAddr,
		"",
		valueTypeString,
		"Full Address. Eg. /ip4/127.0.0.1/tcp/12345/ipfs/QmT5oB6xHSunc64Aojoxa6zg9uH31ajiAVyNfCdBZiwFTV",
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
			}
		}
	}

	cmdName := args[0]

	cmd := cmdFlagSetList[cmdName]
	if cmd == nil {
		fmt.Println("\nERROR:", cmdName, "is an invalid command")
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

	getBlocksRequest := &rpcpb.GetBlocksRequest{}
	getBlocksRequest.MaxCount = maxCount

	// set startBlockHashes of getBlocksRequest if specified in flag
	startBlockHashesString := string(*(flags[flagStartBlockHashes].(*string)))
	if len(startBlockHashesString) > 0 {
		var startBlockHashes [][]byte
		for _, startBlockHash := range strings.Split(startBlockHashesString, ",") {
			startBlockHashInByte, err := hex.DecodeString(startBlockHash)
			if err != nil {
				fmt.Println("ERROR: get blocks failed. ERR:", err)
				return
			}
			startBlockHashes = append(startBlockHashes, startBlockHashInByte)
		}
		getBlocksRequest.StartBlockHashes = startBlockHashes
	}

	response, err := client.(rpcpb.RpcServiceClient).RpcGetBlocks(ctx, getBlocksRequest)
	if err != nil {
		fmt.Println("ERROR: get blocks failed. ERR:", err)
		return
	}

	var encodedBlocks []map[string]interface{}
	for i := 0; i < len(response.Blocks); i++ {
		block := response.Blocks[i]

		var encodedTransactions []map[string]interface{}

		for j := 0; j < len(block.Transactions); j++ {
			transaction := block.Transactions[j]

			var encodedVin []map[string]interface{}
			for k := 0; k < len(transaction.Vin); k++ {
				vin := transaction.Vin[k]
				encodedVin = append(encodedVin, map[string]interface{}{
					"Vout":      vin.Vout,
					"Signature": hex.EncodeToString(vin.Signature),
					"PubKey":    string(vin.PubKey),
				})
			}

			var encodedVout []map[string]interface{}
			for l := 0; l < len(transaction.Vout); l++ {
				vout := transaction.Vout[l]
				encodedVout = append(encodedVout, map[string]interface{}{
					"Value":      string(vout.Value),
					"PubKeyHash": hex.EncodeToString(vout.PubKeyHash),
					"Contract":   vout.Contract,
				})
			}

			encodedTransaction := map[string]interface{}{
				"ID":   hex.EncodeToString(transaction.ID),
				"Vin":  encodedVin,
				"Vout": encodedVout,
			}
			encodedTransactions = append(encodedTransactions, encodedTransaction)
		}

		encodedBlock := map[string]interface{}{
			"Header": map[string]interface{}{
				"Hash":      hex.EncodeToString(block.Header.Hash),
				"Prevhash":  hex.EncodeToString(block.Header.Prevhash),
				"Timestamp": block.Header.Timestamp,
				"Sign":      hex.EncodeToString(block.Header.Sign),
				"height":    block.Header.Height,
			},
			"Transactions": encodedTransactions,
		}

		encodedBlocks = append(encodedBlocks, encodedBlock)
	}

	blocks, err := json.MarshalIndent(encodedBlocks, "", "  ")
	if err != nil {
		fmt.Println("Print blocks failed. ERR: ", err)
	}

	fmt.Println(string(blocks))
}

func getBlockchainInfoCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	response, err := client.(rpcpb.RpcServiceClient).RpcGetBlockchainInfo(ctx, &rpcpb.GetBlockchainInfoRequest{})
	if err != nil {
		fmt.Println("ERROR: GetBlockchainInfo failed. ERR:", err)
		return
	}
	encodedResponse := map[string]interface{}{
		"TailBlockHash": hex.EncodeToString(response.TailBlockHash),
		"BlockHeight":   response.BlockHeight,
		"Producers":     response.Producers,
	}

	blockchainInfo, err := json.MarshalIndent(encodedResponse, "", "  ")
	if err != nil {
		if strings.Contains(err.Error(), "connection error") {
			fmt.Println("ERROR: GetBlockchainInfo failed. The server is not reachable!")
		} else {
			fmt.Printf("ERROR: GetBlockchainInfo failed. %v \n", err.Error())
		}
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
	if core.NewAddress(address).ValidateAddress() == false {
		fmt.Println("Error: Get balance failed: the address is not valid")
		return
	}

	getBalanceRequest := rpcpb.GetBalanceRequest{}
	getBalanceRequest.Name = "getBalance"
	getBalanceRequest.Address = address
	response, err := client.(rpcpb.RpcServiceClient).RpcGetBalance(ctx, &getBalanceRequest)
	if err != nil {
		if strings.Contains(err.Error(), "connection error") {
			fmt.Println("Error: Get balance failed. The server is not reachable!")
		} else {
			fmt.Printf("Error: Get balance failed. %v \n", err.Error())
		}
		return
	}
	if response.Message == "Succeed" {
		fmt.Printf("The balance is: %d\n", response.Amount)
	} else {
		fmt.Println(response.Message)
	}
}

func createWalletCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	empty, err := logic.IsWalletEmpty()
	prompter := util.NewTerminalPrompter()
	passphrase := ""
	if empty {
		passphrase = prompter.GetPassPhrase("Please input the password for generating a new wallet: ", true)
		if passphrase == "" {
			fmt.Println("Password Empty!")
			return
		}
		wallet, err := logic.CreateWalletWithpassphrase(passphrase)
		if err != nil {
			fmt.Printf("Error: Create Wallet Failed. %v \n", err.Error())
			return
		}
		if wallet != nil {
			fmt.Printf("Create Wallet, the address is %s \n", wallet.GetAddress().Address)
			return
		}
	}

	locked, err := logic.IsWalletLocked()
	if err != nil {
		fmt.Printf("Error: Create Wallet Failed. %v \n", err.Error())
		return
	}

	if locked {
		passphrase = prompter.GetPassPhrase("Please input the password: ", false)
		if passphrase == "" {
			fmt.Println("Password Empty!")
			return
		}
		wallet, err := logic.CreateWalletWithpassphrase(passphrase)
		if err != nil {
			fmt.Printf("Error: Create Wallet Failed. %v \n", err.Error())
			return
		}
		if wallet != nil {
			fmt.Printf("Create Wallet, the address is %s\n", wallet.GetAddress().Address)
		}
		//unlock the wallet
		client.(rpcpb.AdminServiceClient).RpcUnlockWallet(ctx, &rpcpb.UnlockWalletRequest{
			Name: "unlock",
		})

		if err != nil {
			fmt.Printf("Error: Unlock Wallet Failed. %v \n", err.Error())
			return
		}
	} else {
		wallet, err := logic.AddWallet()
		if err != nil {
			fmt.Printf("Error: Create Wallet Failed. %v \n", err.Error())
			return
		}
		if wallet != nil {
			fmt.Println("Create Wallet, the address is ", wallet.GetAddress().Address)
		}
	}

	return
}

func listAddressesCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	passphrase := ""
	prompter := util.NewTerminalPrompter()

	empty, err := logic.IsWalletEmpty()
	if err != nil {
		fmt.Printf("Error: List addresses failed. %v \n", err.Error())
		return
	}
	if empty {
		fmt.Println("Please use cli createWallet to generate a wallet first!")
		return
	}

	locked, err := logic.IsWalletLocked()
	if err != nil {
		fmt.Printf("Error: List addresses failed. %v \n", err.Error())
		return
	}
	if locked {
		passphrase = prompter.GetPassPhrase("Please input the password: ", false)
		if passphrase == "" {
			fmt.Println("Password Empty!")
			return
		}
		fl := storage.NewFileLoader(clientpkg.GetWalletFilePath())
		wm := clientpkg.NewWalletManager(fl)
		err := wm.LoadFromFile()
		addressList, err := wm.GetAddressesWithPassphrase(passphrase)
		if err != nil {
			fmt.Printf("Error: List addresses failed. %v \n", err.Error())
			return
		}
		//unlock the wallet
		client.(rpcpb.AdminServiceClient).RpcUnlockWallet(ctx, &rpcpb.UnlockWalletRequest{
			Name: "unlock",
		})

		if len(addressList) == 0 {
			fmt.Println("The addresses in the wallet is empty!")
		} else {
			i := 1
			fmt.Println("The address list:")
			for _, addr := range addressList {
				fmt.Printf("Address[%d]: %s\n", i, addr)
				i++
			}
		}

	} else {
		fl := storage.NewFileLoader(clientpkg.GetWalletFilePath())
		wm := clientpkg.NewWalletManager(fl)
		err := wm.LoadFromFile()
		if err != nil {
			fmt.Printf("Error: List addresses failed. %v \n", err.Error())
			return
		}
		addressList := wm.GetAddresses()
		if len(addressList) == 0 {
			fmt.Println("The addresses in the wallet is empty!")
		} else {
			i := 1
			fmt.Println("The address list:")
			for _, addr := range addressList {
				fmt.Printf("Address[%d]: %s\n", i, addr.Address)
				i++
			}
		}

	}
	return
}

func getPeerInfoCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	response, err := client.(rpcpb.AdminServiceClient).RpcGetPeerInfo(ctx, &rpcpb.GetPeerInfoRequest{})
	if err != nil {
		if strings.Contains(err.Error(), "connection error") {
			fmt.Println("Error: Get peer failed. The server is not reachable!")
		} else {
			fmt.Printf("Error: Get peer failed. %v \n", err.Error())
		}
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}

func sendCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {

	if core.NewAddress(*(flags[flagFromAddress].(*string))).ValidateAddress() == false {
		fmt.Println("the 'from' address is not valid!")
		return
	}

	if core.NewAddress(*(flags[flagToAddress].(*string))).ValidateAddress() == false {
		fmt.Println("the 'to' address is not valid!")
		return
	}

	response, err := client.(rpcpb.RpcServiceClient).RpcGetUTXO(ctx, &rpcpb.GetUTXORequest{
		Address: core.NewAddress(*(flags[flagFromAddress].(*string))).Address,
	})
	if err != nil {
		fmt.Println("ERROR: Send failed. ERR:", err)
		return
	}
	utxos := response.GetUtxos()
	var InputUtxos []*core.UTXO
	for _, u := range utxos {
		uu := core.UTXO{}
		uu.Value = common.NewAmountFromBytes(u.Amount)
		uu.Txid = u.Txid
		uu.PubKeyHash = core.PubKeyHash{u.PublicKeyHash}
		if err != nil {
			fmt.Println("ERROR: Send failed. ERR:", err)
			return
		}
		uu.TxIndex = int(u.TxIndex)
		InputUtxos = append(InputUtxos, &uu)
	}

	tx_utxos, err := GetUTXOsfromAmount(InputUtxos, common.NewAmount(uint64(*(flags[flagAmount].(*int)))))
	if err != nil {
		fmt.Println("ERROR: Send failed. ERR:", err)
		return
	}

	wm, err := logic.GetWalletManager(clientpkg.GetWalletFilePath())
	if err != nil {
		fmt.Println("ERROR: Send failed. ERR:", err)
		return
	}
	senderWallet := wm.GetWalletByAddress(core.NewAddress(*(flags[flagFromAddress].(*string))))


	tx, err := core.NewUTXOTransaction(tx_utxos, core.NewAddress(*(flags[flagFromAddress].(*string))), core.NewAddress(*(flags[flagToAddress].(*string))),
		common.NewAmount(uint64(*(flags[flagAmount].(*int)))), *senderWallet.GetKeyPair(), common.NewAmount(*(flags[flagTip].(*uint64))), *(flags[flagContract].(*string)))

	sendTransactionRequest := rpcpb.SendTransactionRequest{}
	sendTransactionRequest.Transaction = tx.ToProto().(*corepb.Transaction)
	response1, err := client.(rpcpb.RpcServiceClient).RpcSendTransaction(ctx, &sendTransactionRequest)

	if err != nil {
		fmt.Println("ERROR: Send failed. ERR:", err)
		return
	}
	fmt.Println(response1)
	fmt.Println(proto.MarshalTextString(response1))
}

func GetUTXOsfromAmount(inputUTXOs []*core.UTXO, amount *common.Amount) ([]*core.UTXO, error) {
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
	fmt.Printf("Usage Example: cli createWallet\n")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("Command: cli ", "listAddresses")
	fmt.Printf("Usage Example: cli listAddresses\n")
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
			if par.name == flagContract {
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
		fmt.Println("ERROR: AddPeer failed. ERR:", err)
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}

func initRpcClient(port int) *grpc.ClientConn {
	//prepare grpc client
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(fmt.Sprint(":", port), grpc.WithInsecure())
	if err != nil {
		log.Panic("ERROR: Not able to connect to RPC server. ERR:", err)
	}
	return conn
}
