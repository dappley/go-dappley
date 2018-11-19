#### Recommended workflow 
0. `cd $DAP/dapp;`
1. Run ./dapp in detached background mode redirecting stdout and stderr to a file (current using blockchain.log)
`nohup ./dapp -f conf/seed.conf > blockchain.log 2>&1 &`
2. `cd $DAP/tool/transaction_automator`
3. start the test automation script
`./start.sh`
4. If you feel like staring at logs updating you may do so with `tail -f /path/to/logfile`
5. If you need to stop the scripts 
`./stop.sh`