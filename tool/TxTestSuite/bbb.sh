#!/bin/bash
source "./AutoCLILib.sh"

bcHeight=0
setBcHeight bcHeight

newBcHeight=0
addBalanceAmount=10
#var name 'accList' must match variable name in imported lib above
me="dSTDkNcS7Ln4V7AsGoeuuzxiGsVvtyS7Wm"
accList[0]=$me
accList[1]="dThUP369noDhMw5yUwDYTx29awu4SUSM4R"
accList[2]="dJEuZE3T97MQA9ThK5PfHTwFUPS5HcejgS"

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
                newTxToAnotherNode
                bcHeight=$newBcHeight
        else
               setBcHeight newBcHeight
               sleep 1
       fi
done

