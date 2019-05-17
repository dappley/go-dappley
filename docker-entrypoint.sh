#!/bin/bash

cd ${GOPATH}/src/github.com/dappley/go-dappley/dapp

service rsyslog start
service collectd start
service cron start

./dapp
