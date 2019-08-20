FROM ubuntu:16.04
RUN apt-get update && apt-get install make
WORKDIR /opt/go-dappley
COPY vm /opt/go-dappley/vm
WORKDIR /opt/go-dappley/vm/v8
RUN install ../lib/*.so /usr/local/lib && ldconfig
WORKDIR /opt/go-dappley
COPY bin bin
COPY dapp/dapp dapp/dapp
COPY dapp/jslib dapp/jslib
COPY client client
WORKDIR /opt/go-dappley/dapp 
EXPOSE 60054 22341 22342
ENTRYPOINT  ["./dapp"]
