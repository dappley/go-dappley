// +build !release

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
	"flag"
	"fmt"
	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/crypto/keystore/secp256k1"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/rpc/pb"
	storage "github.com/dappley/go-dappley/storage"
	"github.com/dappley/go-dappley/util"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
	"os"
	"strings"
)

//command names
const (
	cliGetBlockchainInfo = "getBlockchainInfo"
	cliGetBalance        = "getBalance"
	cliGetPeerInfo       = "getPeerInfo"
	cliSend              = "send"
	cliAddPeer           = "addPeer"
	clicreateWallet      = "createWallet"
	cliListAddresses     = "listAddresses"
	cliaddBalance        = "addBalance"
	cliaddProducer       = "addProducer"
)

//flag names
const (
	flagAddress        = "address"
	flagAddressBalance = "address"
	flagAmountBalance  = "amount"
	flagToAddress      = "to"
	flagFromAddress    = "from"
	flagAmount         = "amount"
	flagPeerFullAddr   = "peerFullAddr"
	flagProducerAddr   = "address"
	flagListPrivateKey = "privateKey"
)

type valueType int

//type enum
const (
	valueTypeInt = iota
	valueTypeString
	boolType
)

type serviceType int

const (
	rpcService = iota
	adminRpcService
)

//list of commands
var cmdList = []string{
	cliGetBlockchainInfo,
	cliGetBalance,
	cliGetPeerInfo,
	cliSend,
	cliAddPeer,
	clicreateWallet,
	cliListAddresses,
	cliaddBalance,
	cliaddProducer,
}

//configure input parameters/flags for each command
var cmdFlagsMap = map[string][]flagPars{
	cliGetBalance: {flagPars{
		flagAddress,
		"",
		valueTypeString,
		"Address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
	}},
	cliaddProducer: {flagPars{
		flagProducerAddr,
		"",
		valueTypeString,
		"Address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
	}},
	cliaddBalance: {
		flagPars{
			flagAddressBalance,
			"",
			valueTypeString,
			"Address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7"},
		flagPars{
			flagAmountBalance,
			0,
			valueTypeInt,
			"The amount to add to the receiver.",
		},
	},
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
		"privateKey",
	}},
}

//map the callback function to each command
var cmdHandlers = map[string]commandHandlersWithType{
	cliGetBlockchainInfo: {rpcService, getBlockchainInfoCommandHandler},
	cliGetBalance:        {rpcService, getBalanceCommandHandler},
	cliGetPeerInfo:       {rpcService, getPeerInfoCommandHandler},
	cliSend:              {rpcService, sendCommandHandler},
	cliAddPeer:           {adminRpcService, addPeerCommandHandler},
	clicreateWallet:      {rpcService, createWalletCommandHandler},
	cliListAddresses:     {rpcService, listAddressesCommandHandler},
	cliaddBalance:        {rpcService, addBalanceCommandHandler},
	cliaddProducer:       {rpcService, cliaddProducerCommandHandler},
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
}

func getBlockchainInfoCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	response, err := client.(rpcpb.RpcServiceClient).RpcGetBlockchainInfo(ctx, &rpcpb.GetBlockchainInfoRequest{})
	if err != nil {
		fmt.Println("ERROR: GetBlockchainInfo failed. ERR:", err)
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}

func getBalanceCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	if len(*(flags[flagAddress].(*string))) == 0 {
		printUsage()
		fmt.Println("\n Example: cli getBalance -address 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7")
		fmt.Println()
		return
	}

	getBalanceRequest := rpcpb.GetBalanceRequest{}
	getBalanceRequest.Name = "getWallet"

	response, err := client.(rpcpb.RpcServiceClient).RpcGetBalance(ctx, &getBalanceRequest)
	if err != nil {
		if strings.Contains(err.Error(), "connection error") {
			fmt.Printf("Error: Get Balance failed. Network Connection Error!\n")
		} else {
			fmt.Printf("Error: Get Balance failed. %v\n", err.Error())
		}
		return
	}

	passphrase := ""
	if response.Message == "WalletExistsLocked" {
		prompter := util.NewTerminalPrompter()
		passphrase = prompter.GetPassPhrase("Please input the wallet password: ", false)
		if passphrase == "" {
			fmt.Println("Password Empty!")
			return
		}
	} else if response.Message == "WalletExistsNotLocked" {
		passphrase = ""
	} else if response.Message == "NoWallet" {
		fmt.Println("Please use cli createWallet to generate a wallet first!")
		return
	} else {
		fmt.Printf("Error: Create Wallet Failed! %v\n", response.Message)
		return
	}

	getBalanceRequest = rpcpb.GetBalanceRequest{}
	getBalanceRequest.Name = "getBalance"
	getBalanceRequest.Address = *(flags[flagAddress].(*string))
	getBalanceRequest.Passphrase = passphrase
	response, err = client.(rpcpb.RpcServiceClient).RpcGetBalance(ctx, &getBalanceRequest)
	if err != nil {
		if strings.Contains(err.Error(), "Password does not match!") {
			fmt.Printf("ERROR: Get balance failed. Password does not match!\n")
		} else if strings.Contains(err.Error(), "Address not in the wallets") {
			fmt.Printf("ERROR: Get balance failed. Address not found in the wallet!\n")
		} else {
			fmt.Printf("ERROR: Get balance failed. ERR: %v\n", err)
		}
		return
	}
	if response.Message == "Get Balance" {
		fmt.Printf("The balance is: %d\n", response.Amount)
	} else {
		fmt.Println(response.Message)
	}

	return
}

func createWalletCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	empty, err := logic.IsWalletEmpty()
	//if err != nil {
	//	fmt.Printf("Error: Create Wallet Failed. %v \n", err.Error())
	//}
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
		client.(rpcpb.RpcServiceClient).RpcUnlockWallet(ctx, &rpcpb.UnlockWalletRequest{
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

func listAddressesCommandHandler(ctx context.Context, client1 interface{}, flags cmdFlags) {

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
		fl := storage.NewFileLoader(client.GetWalletFilePath())
		wm := client.NewWalletManager(fl)
		err := wm.LoadFromFile()
		addressList, err := wm.GetAddressesWithPassphrase(passphrase)
		if err != nil {
			fmt.Printf("Error: List addresses failed. %v \n", err.Error())
			return
		}
		//unlock the wallet
		client1.(rpcpb.RpcServiceClient).RpcUnlockWallet(ctx, &rpcpb.UnlockWalletRequest{
			Name: "unlock",
		})

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
		fl := storage.NewFileLoader(client.GetWalletFilePath())
		wm := client.NewWalletManager(fl)
		err := wm.LoadFromFile()
		if err != nil {
			fmt.Printf("Error: List addresses failed. %v \n", err.Error())
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

func addBalanceCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	if len(*(flags[flagAddressBalance].(*string))) == 0 {
		printUsage()
		fmt.Println("\n Example: cli addBalance -address 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7 -amount 15")
		fmt.Println()
		return
	}
	amount := int64(*(flags[flagAmountBalance].(*int)))
	if amount <= 0 {
		fmt.Println("Add balance error! The amount must be greater than zero!")
		return
	}

	if len(*(flags[flagAddressBalance].(*string))) != 34 {
		fmt.Println("Add balance error!The length of address must be 34!")
		return
	}

	addBalanceRequest := rpcpb.AddBalanceRequest{}
	addBalanceRequest.Address = *(flags[flagAddressBalance].(*string))
	addBalanceRequest.Amount = common.NewAmount(uint64(*(flags[flagAmountBalance].(*int)))).Bytes()

	response, err := client.(rpcpb.RpcServiceClient).RpcAddBalance(ctx, &addBalanceRequest)
	if err != nil {
		fmt.Println("Add balance error!: ERR:", err)
		return
	}
	fmt.Println(response.Message)
}

func getPeerInfoCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	response, err := client.(rpcpb.RpcServiceClient).RpcGetPeerInfo(ctx, &rpcpb.GetPeerInfoRequest{})
	if err != nil {
		fmt.Println("ERROR: GetPeerInfo failed. ERR:", err)
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}

func cliaddProducerCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {

	if len(*(flags[flagProducerAddr].(*string))) == 0 {
		printUsage()
		fmt.Println("\n Example: cli addProducer -address 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7")
		fmt.Println()
		return
	}

	if len(*(flags[flagProducerAddr].(*string))) != 34 {
		fmt.Println("The length of address must be 34!")
		return
	}

	response, err := client.(rpcpb.RpcServiceClient).RpcAddProducer(ctx, &rpcpb.AddProducerRequest{
		Name:    "addProducer",
		Address: *(flags[flagProducerAddr].(*string)),
	})

	if err != nil {
		fmt.Println("ERROR: Add producer failed. ERR:", err)
		return
	}
	fmt.Println(response.Message)
}

func sendCommandHandler(ctx context.Context, client interface{}, flags cmdFlags) {
	response, err := client.(rpcpb.RpcServiceClient).RpcSend(ctx, &rpcpb.SendRequest{
		From:   *(flags[flagFromAddress].(*string)),
		To:     *(flags[flagToAddress].(*string)),
		Amount: common.NewAmount(uint64(*(flags[flagAmount].(*int)))).Bytes(),
	})
	if err != nil {
		fmt.Println("ERROR: Send failed. ERR:", err)
		return
	}
	fmt.Println(proto.MarshalTextString(response))
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
