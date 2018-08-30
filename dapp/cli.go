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
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/dappley/go-dappley/client"
	"github.com/dappley/go-dappley/consensus"
	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
	"github.com/sirupsen/logrus"
)

// CLI responsible for processing command line arguments
type CLI struct{}

func (cli *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  createwallet")
	fmt.Println("  getbalance -address ADDRESS")
	fmt.Println("  addbalance -address ADDRESS -amount AMOUNT")
	fmt.Println("  listaddresses")
	fmt.Println("  printchain")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT")
	fmt.Println("  setListeningPort -port PORT")
	fmt.Println("  addPeer -address FULLADDRESS")
	fmt.Println("  sendMockBlock")
	fmt.Println("  syncPeers")
	fmt.Println("  setLoggerLevel -level LEVEL")
	fmt.Println("  addProducer -address PRODUCERADDRESS")
	fmt.Println("  setMaxProducers -max MAXPRODUCERS")
	fmt.Println("  exit")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

// Run parses command line arguments and processes commands
func (cli *CLI) Run(bc *core.Blockchain, node *network.Node, wallets *client.Wallets, dynasty *consensus.Dynasty) {

	cli.printUsage()
loop:
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter command: ")
		text, _ := reader.ReadString('\n')
		args := strings.Fields(text)

		getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
		createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
		listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
		addBalanceCmd := flag.NewFlagSet("addbalance", flag.ExitOnError)
		sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
		printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
		nodeSetPortCmd := flag.NewFlagSet("setListeningPort", flag.ExitOnError)
		addPeerCmd := flag.NewFlagSet("addPeer", flag.ExitOnError)
		sendMockBlockCmd := flag.NewFlagSet("sendMockBlock", flag.ExitOnError)
		syncPeersCmd := flag.NewFlagSet("syncPeers", flag.ExitOnError)
		broadcastMockTxCmd := flag.NewFlagSet("broadcastTx", flag.ExitOnError)
		setLoggerLevelCmd := flag.NewFlagSet("setLoggerLevel", flag.ExitOnError)
		addProducerCmd := flag.NewFlagSet("addProducer", flag.ExitOnError)
		setMaxProducersCmd := flag.NewFlagSet("setMaxProducers", flag.ExitOnError)

		getBalanceAddressString := getBalanceCmd.String("address", "", "The address to get balance for")
		addBalanceAddressString := addBalanceCmd.String("address", "", "The address to add balance for")
		sendFrom := sendCmd.String("from", "", "Source client address")
		sendTo := sendCmd.String("to", "", "Destination client address")
		sendAmount := sendCmd.Int("amount", 0, "Amount to send")
		addAmount := addBalanceCmd.Int("amount", 0, "Amount to add")
		tipAmount := sendCmd.Int("tip", 0, "Amount to tip")
		nodePort := nodeSetPortCmd.Int("port", 12345, "Port to listen")
		peerAddr := addPeerCmd.String("address", "", "peer ip4 address")
		loggerLevel := setLoggerLevelCmd.Int("level", 4, "0:Panic 1:Fatal 2:Error 3:Warning 4:Info 5:Debug")
		producerAddr := addProducerCmd.String("address", "", "producer address")
		maxProducers := setMaxProducersCmd.Int("max", 3, "maximum producers")

		var err error
		switch args[0] {
		case "getbalance":
			err = getBalanceCmd.Parse(args[1:])
		case "addbalance":
			err = addBalanceCmd.Parse(args[1:])
		case "createwallet":
			err = createWalletCmd.Parse(args[1:])
		case "listaddresses":
			err = listAddressesCmd.Parse(args[1:])
		case "printchain":
			err = printChainCmd.Parse(args[1:])
		case "send":
			err = sendCmd.Parse(args[1:])
		case "setListeningPort":
			err = nodeSetPortCmd.Parse(args[1:])
		case "addPeer":
			err = addPeerCmd.Parse(args[1:])
		case "sendMockBlock":
			err = sendMockBlockCmd.Parse(args[1:])
		case "syncPeers":
			err = syncPeersCmd.Parse(args[1:])
		case "broadcastTx":
			err = broadcastMockTxCmd.Parse(args[1:])
		case "setLoggerLevel":
			err = setLoggerLevelCmd.Parse(args[1:])
		case "addProducer":
			err = addProducerCmd.Parse(args[1:])
		case "setMaxProducers":
			err = setMaxProducersCmd.Parse(args[1:])
		case "exit":
			break loop
		default:
			cli.printUsage()
		}
		if err != nil {
			log.Panic(err)
		}

		if setMaxProducersCmd.Parsed() {
			dynasty.SetMaxProducers(*maxProducers)
		}

		if addProducerCmd.Parsed() {
			dynasty.AddProducer(*producerAddr)
		}

		if setLoggerLevelCmd.Parsed() {
			if *loggerLevel < 0 || *loggerLevel > 5 {
				nodeSetPortCmd.Usage()
			}
			logrus.SetLevel((logrus.Level)(*loggerLevel))
		}

		if nodeSetPortCmd.Parsed() {
			if *nodePort <= 0 {
				nodeSetPortCmd.Usage()
			}
			err = node.Start(*nodePort)
		}

		if addPeerCmd.Parsed() {
			if *peerAddr == "" {
				addPeerCmd.Usage()
			}
			node.AddStreamByString(*peerAddr)
		}

		if sendMockBlockCmd.Parsed() {
			b := core.GenerateMockBlock()
			node.SendBlock(b)
		}

		if syncPeersCmd.Parsed() {
			node.SyncPeersBlockcast()
		}

		if broadcastMockTxCmd.Parsed() {
			tx := core.MockTransaction()
			node.BroadcastTxCmd(tx)
		}

		if getBalanceCmd.Parsed() {
			if *getBalanceAddressString == "" {
				getBalanceCmd.Usage()
			}
			getBalanceAddress := core.NewAddress(*getBalanceAddressString)
			balance, err := logic.GetBalance(getBalanceAddress, bc.GetDb())
			if err != nil {
				log.Println(err)
			}

			fmt.Printf("Balance of '%s': %d\n", getBalanceAddress, balance)

		}

		if addBalanceCmd.Parsed() {
			if *addBalanceAddressString == "" || *addAmount <= 0 {
				addBalanceCmd.Usage()
			}
			addBalanceAddress := core.NewAddress(*addBalanceAddressString)
			err := logic.AddBalance(addBalanceAddress, *addAmount, bc)
			if err != nil {
				log.Println(err)
			}

			fmt.Printf("Add Balance Amount %d for '%s'\n", *addAmount, addBalanceAddress)

		}

		if createWalletCmd.Parsed() {
			walletAddr, err := logic.CreateWallet()
			if err != nil {
				log.Println(err)
			}
			fmt.Printf("Your new address: %s\n", walletAddr)
		}

		if listAddressesCmd.Parsed() {
			addrs, err := logic.GetAllAddresses()
			if err != nil {
				log.Println(err)
			}
			for _, address := range addrs {
				fmt.Println(address)
			}
		}

		if printChainCmd.Parsed() {
			fmt.Println(bc)
		}

		if sendCmd.Parsed() {
			if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
				sendCmd.Usage()
			}
			sendFromAddress := core.NewAddress(*sendFrom)
			sendToAddress := core.NewAddress(*sendTo)
			senderWallet := wallets.GetWalletByAddress(sendFromAddress)
			if len(senderWallet.Addresses) == 0 {
				logrus.Warn("Sender address could not be found in local wallet")
			} else {
				if err := logic.Send(senderWallet, sendToAddress, *sendAmount, uint64(*tipAmount), bc); err != nil {
					log.Println(err)
				} else {
					fmt.Println("Send Successful")
				}
			}
		}
	}
}
