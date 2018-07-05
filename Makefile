all: dep build test

dep:
	dep ensure -v

test:
	go test ./...

build:
	cd dapp; go build
