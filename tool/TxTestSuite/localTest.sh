#!/bin/bash
source "./AutoCLILib.sh"

bcHeight=0
setBcHeight bcHeight

newBcHeight=0
addBalanceAmount=10

if [ -z "$1" ]
  then
    echo "Error: Missing argument. How many wallets do you want?"; exit 1
fi
# check input argument $1 is number
re='^[0-9]+$'
if ! [[ $1 =~ $re ]] ; then
   echo "Error: Not a number"; exit 1
fi
# check input arg $1 >= 2
if [ $1 -le 1 ];  then
        echo "Error: At least 2 wallets should be created"; exit 1
fi

# check input arg $2 >= 1
if [ $2 -le 1 ];  then
        echo "Error: Too few transactions"; exit 1
fi

# loop $1 times
for (( c=0; c<$1; c++ )); do
        createWallet "$c"
done

counter=0
#before next block is mined, send some money to addr from miner
sendFromMiner
while [ $counter -le $2 ]; do
        if [ $bcHeight -lt $newBcHeight ]; then
		date +"%m-%d-%Y %T" 
                if [ $counter -eq $2 ]; then
                        reviewBalancesAndQuit
                fi
                ((counter++))
                sendFromMiner
                newTxBetweenLocalWallets
		bcHeight=$newBcHeight
        else
               setBcHeight newBcHeight
               sleep 1
       fi
done

