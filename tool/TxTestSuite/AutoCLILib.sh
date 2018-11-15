#!/bin/bash
cliPath="../../dapp/cli/./cli -f ../../dapp/cli/default.conf"
function setBcHeight()
{
	local -n ref=$1
	bcInfo=$(eval $cliPath getBlockchainInfo | sed -n 2p)
	# stdout to array delimited by " "
	arr=(${bcInfo// / })
	# get last index and return everything before ',' 
	height=${arr[-1]%,*} 
	if [[ $bcInfo == *"ERROR: GetBlockchainInfo failed."* ]]; then
  		echo "Error has occurred, make sure dapp service is running."; exit 1
	fi
	ref=$height
}

function createWallet(){
	# create wallet, respond to every command line prompt with 'y' and read stdout
	output=$(yes | eval $cliPath  createWallet)
	while read -r line; do
		arr=(${line// / })
		# get address in output
		address=${arr[-1]}
		# create list of addresses
		accList[$1]=$address	
	done <<< "$output"
}

function reviewBalancesAndQuit()
{
	for i in "${accList[@]}"
	do 
	line=$(eval $cliPath  getBalance -address $i)
	arr=(${line// / })	
	amount=${arr[-1]}
	echo "$i balance: $amount"
	done
	echo "Job done."
	exit 1
}

function sendFromMiner(){
	rand=${accList[$RANDOM % ${#accList[@]} ]}	
	# miner gives some money
	accRich[$counter]=$rand
	echo "adding $addBalanceAmount to ${accRich[$counter]}"
	eval $cliPath  sendFromMiner -to ${accRich[$counter]} -amount $addBalanceAmount	
}

function newTxBetweenLocalWallets(){
	# random amount (0-20)
	amount=$((RANDOM%(addBalanceAmount+10)))
	# from = last sendFromMiner address, to = random index in accList
	from=${accRich[$counter-1]}
	to=${accList[$RANDOM % ${#accList[@]} ]} 
	echo "sending $amount from $from to $to"
	eval $cliPath  send -from $from -to $to -amount $amount
}

function newTxToAnotherNode(){
	from=$me
	to=${accList[$RANDOM % ${#accList[@]} ]} 
	while ["$from" == "$to"]:; do
		to=${accList[$RANDOM % ${#accList[@]} ]}
	done
	amount=$((RANDOM%(addBalanceAmount+10)))
	echo "sending $amount from $from to $to"
	eval $cliPath  send -from $from -to $to -amount $amount
}


