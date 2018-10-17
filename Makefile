all: dep build run test checkRunning

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
	cd dapp; go build;
	cd dapp/cli; go build
run:
	cd dapp; ./dapp > /dev/null 2>&1 &

checkRunning:
	pkill dapp || ( echo "dapp service not running"; exit 666)


release:
	cd dapp; go build -tags=release
	cd dapp/cli; go build -tags=release

deploy-v8:
	sudo install scEngine/lib/libdappleyv8.so /usr/local/lib/
	sudo /sbin/ldconfig
