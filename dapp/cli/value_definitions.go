package main

import (
	"context"
)

const version = "v0.5.0"
const (
	cliGetBlocks         = "getBlocks"
	cliGetBlockchainInfo = "getBlockchainInfo"
	cliGetBalance        = "getBalance"
	cliGetPeerInfo       = "getPeerInfo"
	cliSend              = "send"
	cliAddPeer           = "addPeer"
	cliCreateAccount     = "createAccount"
	cliAddAccount        = "addAccount"
	cliDeleteAccount     = "deleteAccount"
	cliListAddresses     = "listAddresses"
	clichangeProducer    = "changeProducer"
	cliaddProducer       = "addProducer"
	clideleteProducer    = "deleteProducer"
	cliEstimateGas       = "estimateGas"
	cliGasPrice          = "gasPrice"
	cliContractQuery     = "contractQuery"
	cliHelp              = "help"
	cliGetMetricsInfo    = "getMetricsInfo"
	cliGetBlockByHeight  = "getBlockByHeight"
	cliGenerateSeed      = "generateSeed"
	cliConfigGenerator   = "generateConfig"
	cliCreateDID         = "createDID"
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
	flagContractAddr     = "contractAddr"
	flagKey              = "key"
	flagValue            = "value"
	flagBlockHeight      = "height"
	flagGenerateConfig   = "generateConfig"
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
	metricsRpcService
)

const (
	InvalidNode = iota
	MinerNode
	FullNode
)

//list of commands
var cmdList = []string{
	cliGetBlocks,
	cliGetBlockchainInfo,
	cliGetBalance,
	cliGetPeerInfo,
	cliSend,
	cliAddPeer,
	cliCreateAccount,
	cliAddAccount,
	cliDeleteAccount,
	cliListAddresses,
	clichangeProducer,
	cliaddProducer,
	clideleteProducer,
	cliEstimateGas,
	cliGasPrice,
	cliContractQuery,
	cliHelp,
	cliGetMetricsInfo,
	cliGetBlockByHeight,
	cliGenerateSeed,
	cliConfigGenerator,
	cliCreateDID,
}

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
	clichangeProducer: {
		flagPars{
			flagProducerAddr,
			"",
			valueTypeString,
			"Producer's address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
		},
		flagPars{
			flagBlockHeight,
			uint64(0),
			valueTypeUint64,
			"height. Eg. 1",
		},
	}, cliaddProducer: {
		flagPars{
			flagProducerAddr,
			"",
			valueTypeString,
			"Producer's address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
		},
		flagPars{
			flagBlockHeight,
			uint64(0),
			valueTypeUint64,
			"height. Eg. 1",
		},
	}, clideleteProducer: {
		flagPars{
			flagBlockHeight,
			uint64(0),
			valueTypeUint64,
			"height. Eg. 1",
		},
	},
	cliSend: {
		flagPars{
			flagFromAddress,
			"",
			valueTypeString,
			"Sender's account address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
		},
		flagPars{
			flagToAddress,
			"",
			valueTypeString,
			"Receiver's account address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
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
	cliAddAccount: {flagPars{
		flagKey,
		"",
		valueTypeString,
		"Private key of account to be added.",
	}},
	cliDeleteAccount: {
		flagPars{
			flagKey,
			"",
			valueTypeString,
			"Private key of account to be deleted. Do not use alongside address.",
		},
		flagPars{
			flagAddress,
			"",
			valueTypeString,
			"Address of account to be deleted. Do not use alongside key.",
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
	cliEstimateGas: {
		flagPars{
			flagFromAddress,
			"",
			valueTypeString,
			"Sender's account address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
		},
		flagPars{
			flagToAddress,
			"",
			valueTypeString,
			"Receiver's account address. Eg. 1MeSBgufmzwpiJNLemUe1emxAussBnz7a7",
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
	cliGasPrice:       {},
	cliGetMetricsInfo: {},
	cliContractQuery: {
		flagPars{
			flagContractAddr,
			"",
			valueTypeString,
			"Contract address. Eg. cd9N6MRsYxU1ToSZjLnqFhTb66PZcePnAD",
		},
		flagPars{
			flagKey,
			"",
			valueTypeString,
			"The data key storaged in contract address.",
		},
		flagPars{
			flagValue,
			"",
			valueTypeString,
			"The data value storaged in contract address.",
		},
	},
	cliGetBlockByHeight: {
		flagPars{
			flagBlockHeight,
			0,
			valueTypeInt,
			"height. Eg. 1",
		},
	},
	cliGenerateSeed: {},
	cliCreateDID:    {},
}

//map the callback function to each command
var cmdHandlers = map[string]commandHandlersWithType{
	cliGetBlocks:         {rpcService, getBlocksCommandHandler},
	cliGetBlockchainInfo: {rpcService, getBlockchainInfoCommandHandler},
	cliGetBalance:        {rpcService, getBalanceCommandHandler},
	cliGetPeerInfo:       {adminRpcService, getPeerInfoCommandHandler},
	cliSend:              {rpcService, sendCommandHandler},
	cliAddPeer:           {adminRpcService, addPeerCommandHandler},
	cliCreateAccount:     {rpcService, createAccountCommandHandler},
	cliAddAccount:        {rpcService, addAccountCommandHandler},
	cliDeleteAccount:     {rpcService, deleteAccountCommandHandler},
	cliListAddresses:     {adminRpcService, listAddressesCommandHandler},
	clichangeProducer:    {adminRpcService, clichangeProducerCommandHandler},
	cliaddProducer:       {adminRpcService, cliaddProducerCommandHandler},
	clideleteProducer:    {adminRpcService, clideleteProducerCommandHandler},
	cliEstimateGas:       {rpcService, estimateGasCommandHandler},
	cliGasPrice:          {rpcService, gasPriceCommandHandler},
	cliHelp:              {adminRpcService, helpCommandHandler},
	cliContractQuery:     {rpcService, contractQueryCommandHandler},
	cliGetMetricsInfo:    {metricsRpcService, getMetricsInfoCommandHandler},
	cliGetBlockByHeight:  {rpcService, getBlockByHeightCommandHandler},
	cliGenerateSeed:      {adminRpcService, generateSeedCommandHandler},

	cliConfigGenerator: {adminRpcService, configGeneratorCommandHandler},

	cliCreateDID: {rpcService, createDIDCommandHandler},
}

type commandHandlersWithType struct {
	serviceType serviceType
	cmdHandler  commandHandler
}

type commandHandler func(ctx context.Context, account interface{}, flags cmdFlags)

type flagPars struct {
	name         string
	defaultValue interface{}
	valueType    valueType
	usage        string
}

type Node struct {
	NodeType      int
	nodeType      string
	fileName      string
	miner_address string
	private_key   string
	port          string
	seed          string
	db_path       string
	rpc_port      string
	key           string
	node_address  string
}

//map key: flag name   map defaultValue: flag defaultValue
type cmdFlags map[string]interface{}
