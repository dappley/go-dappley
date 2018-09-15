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

## Contribution
Thank you for considering to help with go-dappley project. Any contributions or suggestions are welcome. Please read the following instructions to get started.

Before making your contribution, identify if it's a bug or complex idea. Please file an [issue](https://github.com/dappley/go-dappley/issues) for bugs, then fork, fix, commit and send a pull request for the maintainers to review and merge into the main code base.
For more complex ideas, you'll need to discuss with maintainers in our **Gitter** or **Slack** chanel first. This helps to prevent duplicated efforts and save other contributors time.

See [Installation Instruction](https://github.com/dappley/go-dappley/wiki/Install) to configure your environment and follow [Go formatting](https://golang.org/doc/effective_go.html#formatting) to keep the coding style consistent in the project. All pull requests should be based on the `master` branch. 
Please refer to [Contribution Guideline](https://github.com/dappley/go-dappley/wiki/Contribution-guideline) for development practices in Dappley.

### License
The go-dappley project is licensed under the [GNU Lesser General Public License Version 3.0 (“LGPL v3”)](https://www.gnu.org/licenses/lgpl-3.0.en.html).
