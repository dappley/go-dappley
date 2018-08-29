package main

import (
	"fmt"
	"bufio"
	"os"
	"flag"
	"strings"
)

//command names
const(
	cliGetBalance 		= "getBalance"
	cliGetPeerInfo		= "getPeerInfo"
	cliExit				= "exit"
)

//type enum
const(
	valueTypeInt = iota
	valueTypeString
)

//list of commands
var cmdList = []string{
	cliGetBalance,
	cliGetPeerInfo,
	cliExit,
}

//configure input parameters/flags for each command
var cmdFlagsMap = map[string][]flagPars{
	cliGetBalance	: {flagPars{
		"address",
		"",
		valueTypeString,
		"Get the balance of the input address",

	}},
	cliGetPeerInfo	: {},
}

//map the callback function to each command
var cmdHandlers = map[string]commandHandlers{
	cliGetBalance	:	getBalanceCommandHandler,
	cliGetPeerInfo	:	getPeerInfoCommandHandler,
}

type commandHandlers func()

type flagPars struct{
	name 		string
	value 		interface{}
	valueType	valueType
	usage 		string
}

type valueType int



func main(){

	printUsage()

	cmdFlagSetList := map[string]*flag.FlagSet{}
	//set up flagset for each command
	for _, cmd := range cmdList{
		fs := flag.NewFlagSet(cmd,flag.ContinueOnError)
		cmdFlagSetList[cmd] = fs
	}

	cmdFlagValues := map[string]map[string]interface{}{}
	//set up flags for each command
	for cmd, pars := range cmdFlagsMap {
		cmdFlagValues[cmd] = map[string]interface{}{}
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
			fmt.Println("\nERROR:", cmdName, "is an invalid command\n")
			printUsage()
		}else{
			err := cmd.Parse(args[1:])
			if err!= nil{
				cmd.Usage()
				continue
			}
			if cmd.Parsed() {
				cmdHandlers[cmdName]()
			}
		}
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  getBalance -address ADDRESS")
	fmt.Println("  getPeerInfo")
	fmt.Println("  exit")
}

func getBalanceCommandHandler(){
	fmt.Println("getBalance!")
}

func getPeerInfoCommandHandler(){
	fmt.Println("getPeerInfo!")
}