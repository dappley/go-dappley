package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"sync"

	"github.com/dappley/go-dappley/core"
	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/network"
)

// CLI responsible for processing command line arguments
type CLI struct{}

func (cli *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  createblockchain -address ADDRESS")
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
	fmt.Println("  exit")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

// Run parses command line arguments and processes commands
func (cli *CLI) Run(dep *Dep, signal chan bool, waitGroup sync.WaitGroup) {

	cli.printUsage()
	var node *network.Node
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter command: ")
		text, _ := reader.ReadString('\n')
		args := strings.Fields(text)

		getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
		createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
		createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
		listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
		addBalanceCmd := flag.NewFlagSet("addbalance", flag.ExitOnError)
		sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
		printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
		nodeSetPortCmd := flag.NewFlagSet("setListeningPort", flag.ExitOnError)
		addPeerCmd := flag.NewFlagSet("addPeer", flag.ExitOnError)
		sendMockBlockCmd := flag.NewFlagSet("sendMockBlock", flag.ExitOnError)
		syncPeersCmd := flag.NewFlagSet("syncPeers", flag.ExitOnError)

		getBalanceAddressString := getBalanceCmd.String("address", "", "The address to get balance for")
		addBalanceAddressString := addBalanceCmd.String("address", "", "The address to add balance for")
		createBlockchainAddressString := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
		sendFrom := sendCmd.String("from", "", "Source client address")
		sendTo := sendCmd.String("to", "", "Destination client address")
		sendAmount := sendCmd.Int("amount", 0, "Amount to send")
		addAmount := addBalanceCmd.Int("amount", 0, "Amount to add")
		tipAmount := sendCmd.Int("tip", 0, "Amount to tip")
		nodePort := nodeSetPortCmd.Int("port", 12345, "Port to listen")
		peerAddr := addPeerCmd.String("address", "", "peer ip4 address")

		var err error
		switch args[0] {
		case "getbalance":
			err = getBalanceCmd.Parse(args[1:])
		case "addbalance":
			err = addBalanceCmd.Parse(args[1:])
		case "createblockchain":
			err = createBlockchainCmd.Parse(args[1:])
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
		case "exit":
			signal <- true
			os.Exit(1)
		default:
			cli.printUsage()
		}
		if err != nil {
			log.Panic(err)
		}

		if nodeSetPortCmd.Parsed() {
			if *nodePort <= 0 {
				nodeSetPortCmd.Usage()
			}
			node = network.NewNode(dep.bc)
			err = node.Start(*nodePort)
		}

		if addPeerCmd.Parsed() {
			if *peerAddr == "" {
				addPeerCmd.Usage()
			}
			node.AddStreamString(*peerAddr)
		}

		if sendMockBlockCmd.Parsed() {
			b := core.GenerateMockBlock()
			node.SendBlock(b)
		}

		if syncPeersCmd.Parsed() {
			node.SyncPeers()
		}

		if getBalanceCmd.Parsed() {
			if *getBalanceAddressString == "" {
				getBalanceCmd.Usage()
			}
			getBalanceAddress := core.NewAddress(*getBalanceAddressString)
			balance, err := logic.GetBalance(getBalanceAddress, dep.db)
			if err != nil {
				log.Println(err)
			}

			fmt.Printf("Balance of '%s': %d\n", getBalanceAddress, balance)

		}

		if addBalanceCmd.Parsed() {
			if *addBalanceAddressString == "" || *addAmount <=0 {
				addBalanceCmd.Usage()
			}
			addBalanceAddress := core.NewAddress(*addBalanceAddressString)
			err := logic.AddBalance(addBalanceAddress, *addAmount, dep.db)
			if err != nil {
				log.Println(err)
			}

			fmt.Printf("Add Balance Amount %d for '%s'\n", *addAmount, addBalanceAddress, )

		}

		if createBlockchainCmd.Parsed() {
			if *createBlockchainAddressString == "" {
				createBlockchainCmd.Usage()
			}
			createBlockchainAddress := core.NewAddress(*createBlockchainAddressString)
			_, err := logic.CreateBlockchain(createBlockchainAddress, dep.db)
			if err != nil {
				log.Println(err)
			} else {
				fmt.Println("Create Blockchain Successful")
			}
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
			cli.printChain()
		}

		if sendCmd.Parsed() {
			if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
				sendCmd.Usage()
			}
			sendFromAddress := core.NewAddress(*sendFrom)
			sendToAddress := core.NewAddress(*sendTo)
			if err := logic.Send(sendFromAddress, sendToAddress, *sendAmount, int64(*tipAmount), dep.db); err != nil {
				log.Println(err)
			} else {
				fmt.Println("Send Successful")
			}
		}
	}
}
