#!/bin/bash
nohup ./transaction_automator > tx.log &
echo $! > script.pid
