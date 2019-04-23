FROM golang:1.11
WORKDIR $GOPATH/src/github.com/dappley/go-dappley
COPY . .
RUN make build
WORKDIR $GOPATH/src/github.com/dappley/go-dappley/dapp
CMD ["./dapp"]
