package main

import (
	"fmt"
	"bufio"
	"os"
	"flag"
	"strings"
	"google.golang.org/grpc"
	"github.com/dappley/go-dappley/rpc/pb"
	"context"
	"log"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/metadata"
)

//command names
const(
	cliGetBlockchainInfo	= "getBlockchainInfo"
	cliGetBalance 			= "getBalance"
	cliGetPeerInfo			= "getPeerInfo"
	cliExit					= "exit"
)

//flag names
const(
	flagAddress			= "address"
)

//type enum
const(
	valueTypeInt = iota
	valueTypeString
)

//list of commands
var cmdList = []string{
	cliGetBlockchainInfo,
	cliGetBalance,
	cliGetPeerInfo,
	cliExit,
}

//configure input parameters/flags for each command
var cmdFlagsMap = map[string][]flagPars{
	cliGetBlockchainInfo 	:{},
	cliGetBalance			:{flagPars{
		flagAddress,
		"",
		valueTypeString,
		"Get the balance of the input address",

	}},
	cliGetPeerInfo			:{},
}

//map the callback function to each command
var cmdHandlers = map[string]commandHandlers{
	cliGetBlockchainInfo	: getBlockchainInfoCommandHandler,
	cliGetBalance			: getBalanceCommandHandler,
	cliGetPeerInfo			: getPeerInfoCommandHandler,
}

type commandHandlers func(context.Context,rpcpb.ConnectClient, cmdFlags)

type flagPars struct{
	name 		string
	value 		interface{}
	valueType	valueType
	usage 		string
}

type valueType int
//map key: flag name   map value: flag value
type cmdFlags map[string]interface{}


func main(){

	var rpcPort int
	flag.IntVar(&rpcPort, "p", 50050, "RPC server port")
	flag.Parse()
	if rpcPort <= 0 {
		log.Panic("rpc port is invalid")
	}

	printUsage()
	conn := initRpcClient(rpcPort)
	defer conn.Close()
	client:= rpcpb.NewConnectClient(conn)

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
				cmdFlagValues[cmd][par.name] = cmdFlagSetList[cmd].Int(par.name,par.value.(int),par.usage)
			case valueTypeString:
				cmdFlagValues[cmd][par.name] = cmdFlagSetList[cmd].String(par.name,par.value.(string),par.usage)
			}
		}
	}

	for{
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		args := strings.Fields(text)
		if len(args)==0 {
			continue
		}

		cmdName := args[0]
		if cmdName == cliExit{
			return
		}

		cmd := cmdFlagSetList[cmdName]
		if cmd == nil {
			fmt.Println("\nERROR:", cmdName, "is an invalid command")
			printUsage()
		}else{
			err := cmd.Parse(args[1:])
			if err!= nil{
				cmd.Usage()
				continue
			}
			if cmd.Parsed() {
				md := metadata.Pairs("password", "password")
				ctx := metadata.NewOutgoingContext(context.Background(), md)
				cmdHandlers[cmdName](ctx, client, cmdFlagValues[cmdName])
			}
		}
	}
}

func printUsage() {
	fmt.Println("Usage:")
	for _,cmd := range cmdList{
		fmt.Println(" ", cmd)
	}
}

func getBlockchainInfoCommandHandler(ctx context.Context, client rpcpb.ConnectClient, flags cmdFlags){
	response,err  := client.RpcGetBlockchainInfo(ctx,&rpcpb.GetBlockchainInfoRequest{})
	if err!=nil {
		fmt.Println("ERROR: GetBlockchainInfo failed. ERR:", err)
		return
	}
	fmt.Println(proto.MarshalTextString(response))
}

func getBalanceCommandHandler(ctx context.Context, client rpcpb.ConnectClient, flags cmdFlags){
	//TODO
	fmt.Println("getBalance!")
	fmt.Println(*(flags[flagAddress].(*string)))
}

func getPeerInfoCommandHandler(ctx context.Context, client rpcpb.ConnectClient, flags cmdFlags){

	response,err  := client.RpcGetPeerInfo(ctx,&rpcpb.GetPeerInfoRequest{})
	if err!=nil {
		fmt.Println("ERROR: GetPeerInfo failed. ERR:", err)
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