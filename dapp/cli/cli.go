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
	"flag"
	"fmt"
	"os"

	"github.com/dappley/go-dappley/core/utxo"

	"github.com/dappley/go-dappley/common"
	"github.com/dappley/go-dappley/config"
	configpb "github.com/dappley/go-dappley/config/pb"
	"github.com/dappley/go-dappley/core/account"
	"github.com/dappley/go-dappley/logic"
	rpcpb "github.com/dappley/go-dappley/rpc/pb"
	"github.com/dappley/go-dappley/util"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

//command names

func main() {

	var filePath string
	flag.StringVar(&filePath, "f", "default.conf", "CLI config file path")
	var ver bool
	flag.BoolVar(&ver, "v", false, "display version")
	flag.Parse()

	if ver {
		println(version)
		return
	}

	cliConfig := &configpb.CliConfig{}
	config.LoadConfig(filePath, cliConfig)

	conn := initRpcClient(int(cliConfig.GetPort()))
	defer conn.Close()
	clients := map[serviceType]interface{}{
		rpcService:        rpcpb.NewRpcServiceClient(conn),
		adminRpcService:   rpcpb.NewAdminServiceClient(conn),
		metricsRpcService: rpcpb.NewMetricServiceClient(conn),
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

func createAccount(ctx context.Context, c interface{}, flags cmdFlags) *account.Account {
	var acc *account.Account
	empty, err := logic.IsAccountEmpty()
	prompter := util.NewTerminalPrompter()
	passphrase := ""

	if err != nil {
		fmt.Println("Error:", err.Error())
		return nil
	}
	if empty {
		passphrase = prompter.GetPassPhrase("Please input the password for the new account: ", true)

		if passphrase == "" {
			fmt.Println("Error: password cannot be empty!")
			return nil
		}
		account, err := logic.CreateAccountWithPassphrase(passphrase)
		if err != nil {
			fmt.Println("Error:", err.Error())
			return nil
		}
		acc = account
	} else {
		passphrase = prompter.GetPassPhrase("Please input the password: ", false)
		if passphrase == "" {
			fmt.Println("Error: password should not be empty!")
			return nil
		}
		account, err := logic.CreateAccountWithPassphrase(passphrase)
		if err != nil {
			fmt.Println("Error:", err.Error())
			return nil
		}

		acc = account
	}
	return acc
}

func getUTXOsfromAmount(inputUTXOs []*utxo.UTXO, amount *common.Amount, tip *common.Amount, gasLimit *common.Amount, gasPrice *common.Amount) ([]*utxo.UTXO, error) {

	if tip != nil {
		amount = amount.Add(tip)
	}
	if gasLimit != nil {
		limitedFee := gasLimit.Mul(gasPrice)
		amount = amount.Add(limitedFee)
	}
	var retUtxos []*utxo.UTXO
	sum := common.NewAmount(0)

	vinRulesCheck := false
	for i := 0; i < len(inputUTXOs); i++ {
		retUtxos = append(retUtxos, inputUTXOs[i])
		sum = sum.Add(inputUTXOs[i].Value)
		if vinRules(sum, amount, i, len(inputUTXOs)) {
			vinRulesCheck = true
			break
		}
	}
	if vinRulesCheck {
		return retUtxos, nil
	}
	if sum.Cmp(amount) > 0 {
		return nil, ErrTooManyUtxoFund
	}
	return nil, ErrInsufficientFund
}

func vinRules(utxoSum, amount *common.Amount, utxoNum, remainUtxoNum int) bool {
	return utxoSum.Cmp(amount) >= 0 && (utxoNum == 50 || remainUtxoNum < 100)
}

func initRpcClient(port int) *grpc.ClientConn {
	//prepare grpc account
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(fmt.Sprint(":", port), grpc.WithInsecure())
	if err != nil {
		logger.Panic("Error:", err.Error())
	}
	return conn
}
