#!/bin/bash
kill -9 $(cat pid/script.pid)
kill -9 $(cat pid/dapp.pid)