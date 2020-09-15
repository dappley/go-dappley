# go-dappley
#
Official Go implementation of the dappley protocol.

[![Build Status](https://travis-ci.com/dappley/go-dappley.svg?branch=master)](https://travis-ci.com/dappley/go-dappley) [![Go Report Card](https://goreportcard.com/badge/github.com/dappley/go-dappley)](https://goreportcard.com/report/github.com/dappley/go-dappley)

## Building from source

### Prerequisites
| Components | Version | Description |
|----------|-------------|-------------|
|[Golang](https://golang.org) | >= 1.14| The Go Programming Language |

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
make build
# note: you may be asked to grant super user permission
```

## Running Dapp
The executable is located in the ```dapp``` folder. Run the following command to bring up a full node in the network.
``` bash
cd dapp
./dapp
```

## Running Multiple Nodes On A Machine
### Start node 1
``` bash
cd $GOPATH/src/github.com/dappley/go-dappley/dapp
./dapp -f conf/node1.conf
```

### Start node 2
``` bash
./dapp -f conf/node2.conf
```

## Running Mutiple Nodes On Multiple Machines
### Start node 1
1. On your first machine, run `ifconfig` to find your ip address.
``` bash
ifconfig
```

2. Run the following command to start your first node.
``` bash
cd $GOPATH/src/github.com/dappley/go-dappley/dapp
./dapp -f conf/node1.conf
```

### Start node 2 
1. On your second machine, first go to your node2.conf file
``` bash
cd $GOPATH/src/github.com/dappley/go-dappley/dapp/conf
vim node2.conf
```

2. Modify the ip address of the seed node. Replace `<node 1 ip address>` with your node 1 ip address that you have found in the previous step
```
consensus_config{
    miner_address: "dUuPPYshbBgkzUrgScEHWvdGbSxC8z4R12"
    private_key: "da9282440fae188c371165e01615a2e1b14af68b3eaae51e6608c0bd86d4e6a6"
}

node_config{
    port:   12342
    seed:   ["/ip4/<node 1 ip address>/tcp/12341/ipfs/QmNzA9rsEcM5nAzX9PzTrabJsGiifzaUU85Qe78HSDzSSE"]
    db_path: "../bin/node2.db"
    rpc_port: 50052
    key: "CAESYLUt8Dxqx0MKGZ/dF9cFei8Usm5CPBNat2GhZsv86jJD7oFAPV5Fm7GG1/enKfKAFhrMpyM3UGwvPo2tHNlIdVPugUA9XkWbsYbX96cp8oAWGsynIzdQbC8+ja0c2Uh1Uw=="
    tx_pool_limit: 102400
    blk_size_limit: 102400
    node_address: "dWNrwKvATvPNXNtNNXSj1yzMGerxRQhwUw"
    metrics_interval: 7200
    metrics_polling_interval: 5
}
```

3. Start your peer node
``` bash
cd ../
./dapp -f conf/node2.conf
```

## Contribution
Thank you for considering helping with go-dappley project. Any contributions or suggestions are welcome. Please read the following instructions to get started.

Before you make any contribution, identify if it's a bug-fix or implementation of a complex idea. Please file an [issue](https://github.com/dappley/go-dappley/issues) for bugs, then fork, fix, commit and send a pull request for the maintainers to review and merge into the main code base.
For complex ideas, you'll need to discuss with maintainers in our [Gitter](https://gitter.im/dappley/Lobby) or [Telegram](https://t.me/joinchat/HMgbi0viAbTrk7ClgIQdjw) chanel first. This helps to prevent duplicated efforts and save other contributors time.

See [Installation Instruction](https://github.com/dappley/go-dappley/wiki/Install) to configure your environment and follow [Go formatting](https://golang.org/doc/effective_go.html#formatting) to keep the coding style consistent in the project. All pull requests should be based on the `master` branch. 
Please refer to [Contribution Guideline](https://github.com/dappley/go-dappley/wiki/Contribution-guideline) for development practices in Dappley.

### License
The go-dappley project is licensed under the [GNU Lesser General Public License Version 3.0 (“LGPL v3”)](https://www.gnu.org/licenses/lgpl-3.0.en.html).
