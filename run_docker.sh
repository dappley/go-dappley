#!/usr/bin/env bash
printPrompt(){
    echo ""
    echo "Please follow the format"
    echo "  ./run_docker.sh <configFilePath> <port>"
    echo ""
}

if [ -z "$1" ]
  then
    printPrompt
    exit 1
fi

if [ -z "$2" ]
  then
    printPrompt
    exit 1
fi

sudo docker run -d -p "$2":"$2" -v $(realpath "$1"):/go/src/github.com/dappley/go-dappley/dapp/conf/default.conf dappley/go-dappley