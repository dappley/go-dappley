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

while :; do
	# if block is mined, then miner has some money to send 
        if [ $bcHeight -lt $newBcHeight ]; then
		date +"%m-%d-%Y %T"
		#send amount is random[0,20) for testing purposes
                newTxToAnotherNode
                bcHeight=$newBcHeight
        else
               setBcHeight newBcHeight
               sleep 1
       fi
done

