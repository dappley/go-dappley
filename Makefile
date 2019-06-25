all: dep build deploy-v8 run test check-running

testall:
	go clean -testcache
	for f in ./*/; do \
		if [ "$$f" != "./bin/" -a  "$$f" != "./vendor/" ]; then \
			cd $$f; go test -tags=integration ./... ; cd ..; \
		fi \
	done

dep:
	dep ensure -v

test:
	go clean -testcache
	for f in ./*/; do \
		if [ "$$f" != "./bin/" -a  "$$f" != "./vendor/" ]; then \
			cd $$f; go test ./...; cd ..; \
		fi \
	done

build:
	dep ensure -update
	make deploy-v8
	cd dapp; go build
	cd dapp/cli; go build
run:
	cd dapp; ./dapp > /dev/null 2>&1 &

check-running:
	pkill dapp || ( echo "dapp service not running"; exit 666)


release:
	cd dapp; go build -tags=release
	cd dapp/cli; go build -tags=release

deploy-v8:
	cd contract/v8; make build; make install
