# go-dappley

Official Go implementation of the dappley protocol.

[![Build Status](https://travis-ci.com/dappley/go-dappley.svg?branch=master)](https://travis-ci.com/dappley/go-dappley)


## Building from source

### Prerequisites
| Components | Version | Description |
|----------|-------------|-------------|
|[Golang](https://golang.org) | >= 1.9.1| The Go Programming Language |
[Dep](https://github.com/golang/dep) | >= 0.5.0 | Dep is a dependency management tool for Go. |

Please ensure ```GOPATH``` and ```GOROOT``` are set up correctly for ```dep``` to install all required dependencies. You may find [this guide](https://github.com/golang/go/wiki/SettingGOPATH) helpful.

For detailed instructions about the environment setup for go-dappley, please check out the [wiki](https://github.com/dappley/go-dappley/wiki) page.

### Build

1. Checkout repo.

```bash
cd $GOPATH/src
go get -u -v github.com/dappley/go-dappley
```

2. Import dependencies and build.

```bash
cd github.com/dappley/go-dappley
make
```

## Running dapp
The executable is located in the ```dapp``` folder. Run the following command to bring up a full node in the network.
``` bash
cd dapp
./dapp
```
