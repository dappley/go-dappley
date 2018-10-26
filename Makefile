all: dep build test

testall:
	for f in ./*/; do \
        if [ "$$f" != "./bin/" -a  "$$f" != "./vendor/" ]; then \
            cd $$f; go test -tags=integration ./... ; cd ..; \
        fi \
    done

dep:
	dep ensure -v

test:
	for f in ./*/; do \
		if [ "$$f" != "./bin/" -a  "$$f" != "./vendor/" ]; then \
			cd $$f; go test ./...; cd ..; \
		fi \
	done

build:
	cd dapp; go build
	cd dapp/cli; go build

release:
	cd dapp; go build -tags=release
	cd dapp/cli; go build -tags=release

