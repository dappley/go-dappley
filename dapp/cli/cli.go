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
	"flag"
	"fmt"
	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/rpc/pb"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
	"os"
	"github.com/dappley/go-dappley/config"
	"github.com/dappley/go-dappley/util"
	"github.com/dappley/go-dappley/config/pb"
)

//command names
const(
	cliGetBlockchainInfo	= "getBlockchainInfo"
	cliGetBalance 			= "getBalance"
	cliGetPeerInfo			= "getPeerInfo"
	cliSend 				= "send"
	cliAddPeer 				= "addPeer"
	clicreateWallet			= "createWallet"
)

//flag names
const(
	flagAddress			= "address"
	flagToAddress		= "to"
	flagFromAddress		= "from"
	flagAmount			= "amount"
	flagPeerFullAddr    = "peerFullAddr"
)

type valueType int
//type enum
const(
	valueTypeInt = iota
	valueTypeString
)

type serviceType int
const(
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
}

//configure input parameters/flags for each command
var cmdFlagsMap = map[string][]flagPars{
	cliGetBalance	:{	flagPars{
		flagAddress,
		"",
		valueTypeString,
		"Address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",

	}},
	cliSend	: {
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
	cliAddPeer		:{flagPars{
		flagPeerFullAddr,
		"",
		valueTypeString,
		"Full Address. Eg. /ip4/127.0.0.1/tcp/12345/ipfs/QmT5oB6xHSunc64Aojoxa6zg9uH31ajiAVyNfCdBZiwFTV",
	}},
}

//map the callback function to each command
var cmdHandlers = map[string]commandHandlersWithType{
	cliGetBlockchainInfo	: {rpcService, getBlockchainInfoCommandHandler},
	cliGetBalance			: {rpcService, getBalanceCommandHandler},
	cliGetPeerInfo			: {rpcService, getPeerInfoCommandHandler},
	cliSend					: {rpcService, sendCommandHandler},
	cliAddPeer				: {adminRpcService, addPeerCommandHandler},
	clicreateWallet			:{rpcService, createWalletCommandHandler},
}

type commandHandlersWithType struct {
	serviceType		serviceType
	cmdHandler 		commandHandler
}

type commandHandler func(ctx context.Context, client interface{},flags cmdFlags)

type flagPars struct{
	name         string
	defaultValue interface{}
	valueType    valueType
	usage        string
}


//map key: flag name   map defaultValue: flag defaultValue
type cmdFlags map[string]interface{}



func main(){

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
	for _, cmd := range cmdList{
		fs := flag.NewFlagSet(cmd,flag.ContinueOnError)
		cmdFlagSetList[cmd] = fs
	}

	cmdFlagValues := map[string]cmdFlags{}
	//set up flags for each command
	for cmd, pars := range cmdFlagsMap {
		cmdFlagValues[cmd] = cmdFlags{}
		for _,par := range pars{
			switch par.valueType{
			case valueTypeInt:
				cmdFlagValues[cmd][par.name] = cmdFlagSetList[cmd].Int(par.name,par.defaultValue.(int),par.usage)
			case valueTypeString:
				cmdFlagValues[cmd][par.name] = cmdFlagSetList[cmd].String(par.name,par.defaultValue.(string),par.usage)
			}
		}
	}

	cmdName := args[0]

	cmd := cmdFlagSetList[cmdName]
	if cmd == nil {
		fmt.Println("\nERROR:", cmdName, "is an invalid command")
		printUsage()
	}else{
		err := cmd.Parse(args[1:])
		if err!= nil{
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
	for _,cmd := range cmdList{
		fmt.Println(" ", cmd)
	}
}

func getBlockchainInfoCommandHandler(ctx context.Context, client interface{}, flags cmdFlags){
	response,err  := client.(rpcpb.RpcServiceClient).RpcGetBlockchainInfo(ctx,&rpcpb.GetBlockchainInfoRequest{})
	if err!=nil {
		fmt.Println("ERROR: GetBlockchainInfo failed. ERR:", err)
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}

func getBalanceCommandHandler(ctx context.Context, client interface{}, flags cmdFlags){
	//TODO
	fmt.Println("getBalance!")
	fmt.Println(*(flags[flagAddress].(*string)))
}

func createWalletCommandHandler(ctx context.Context, client interface{}, flags cmdFlags){
	prompter := util.NewTerminalPrompter()
	passphrase:= prompter.GetPassPhrase("Please input the password: ",true)
	fmt.Println(passphrase)
	walletRequest := rpcpb.CreateWalletRequest{}
	walletRequest.SetPassphrase(passphrase)
	response,err  := client.(rpcpb.RpcServiceClient).RpcCreateWallet(ctx,&walletRequest)
	if err!=nil {
		fmt.Println("ERROR: Create Wallet failed. ERR:", err)
		return
	}
	if (response.Message == "Create Wallet: Error") {
		fmt.Println("Error: Create Wallet failed. ERR: Fail to create address!")
	}
	fmt.Println("Create Wallet, the address is ",response.Address)
}

func getPeerInfoCommandHandler(ctx context.Context, client interface{}, flags cmdFlags){
	response,err  := client.(rpcpb.RpcServiceClient).RpcGetPeerInfo(ctx,&rpcpb.GetPeerInfoRequest{})
	if err!=nil {
		fmt.Println("ERROR: GetPeerInfo failed. ERR:", err)
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}

func sendCommandHandler(ctx context.Context, client interface{}, flags cmdFlags){
	response, err  := client.(rpcpb.RpcServiceClient).RpcSend(ctx, &rpcpb.SendRequest{
		From: *(flags[flagFromAddress].(*string)),
		To: *(flags[flagToAddress].(*string)),
		Amount: common.NewAmount(uint64(*(flags[flagAmount].(*int)))).Bytes(),
	})
	if err!=nil {
		fmt.Println("ERROR: Send failed. ERR:", err)
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}

func addPeerCommandHandler(ctx context.Context, client interface{}, flags cmdFlags){
	req := &rpcpb.AddPeerRequest{
		FullAddress:  *(flags[flagPeerFullAddr].(*string)),
	}
	response,err  := client.(rpcpb.AdminServiceClient).RpcAddPeer(ctx,req)
	if err!=nil {
		fmt.Println("ERROR: AddPeer failed. ERR:", err)
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}

func initRpcClient(port int)*grpc.ClientConn{
	//prepare grpc client
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(fmt.Sprint(":",port), grpc.WithInsecure())
	if err != nil{
		log.Panic("ERROR: Not able to connect to RPC server. ERR:",err)
	}
	return conn
}