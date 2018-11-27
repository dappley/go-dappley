#!/bin/bash
#start up dapp
nohup ../../dapp/dapp  > dapp.log &
echo $! > dapp.pid
#wait 5 seconds
echo "Dapp starting... 5 Seconds until test script starts..."
sleep 5
#start up test script
nohup ./transaction_automator > script.log &
echo $! > script.pid

echo "use command 'tail -f dapp.log' to monitor dappley blockchain status"
echo "use command 'tail -f script.log' to monitor test script status"
