package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/dappley/go-dappley/logic"
	"github.com/dappley/go-dappley/storage"
	"sync"
)

// CLI responsible for processing command line arguments
type CLI struct{}

func (cli *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  createblockchain -address ADDRESS")
	fmt.Println("  createwallet")
	fmt.Println("  getbalance -address ADDRESS")
	fmt.Println("  listaddresses")
	fmt.Println("  printchain")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT")
	fmt.Println("  exit")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

// Run parses command line arguments and processes commands
func (cli *CLI) Run(db storage.LevelDB, signal chan bool, waitGroup sync.WaitGroup) {
	cli.printUsage()
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter command: ")
		text, _ := reader.ReadString('\n')
		args := strings.Fields(text)

		getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
		createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
		createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
		listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
		sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
		printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

		getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
		createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
		sendFrom := sendCmd.String("from", "", "Source client address")
		sendTo := sendCmd.String("to", "", "Destination client address")
		sendAmount := sendCmd.Int("amount", 0, "Amount to send")
		tipAmount := sendCmd.Int("tip", 0, "Amount to tip")

		var err error
		switch args[0] {
		case "getbalance":
			err = getBalanceCmd.Parse(args[1:])
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
		case "exit":
			signal <- true
			os.Exit(1)
		default:
			cli.printUsage()
		}
		if err != nil {
			log.Panic(err)
		}

		if getBalanceCmd.Parsed() {
			if *getBalanceAddress == "" {
				getBalanceCmd.Usage()
			}
			balance, err := logic.GetBalance(*getBalanceAddress, db)
			if err != nil {
				log.Println(err)
			}

			fmt.Printf("Balance of '%s': %d\n", *getBalanceAddress, balance)

		}

		if createBlockchainCmd.Parsed() {
			if *createBlockchainAddress == "" {
				createBlockchainCmd.Usage()
			}

			_, err := logic.CreateBlockchain(*createBlockchainAddress, db)
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

			if err := logic.Send(*sendFrom, *sendTo, *sendAmount, int64(*tipAmount), db); err != nil {
				log.Println(err)
			} else {
				fmt.Println("Send Successful")
			}
		}
	}
}
