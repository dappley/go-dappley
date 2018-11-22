#!/bin/bash
kill -9 $(cat script.pid)
kill -9 $(cat dapp.pid)