FROM ubuntu:16.04
RUN apt-get update && apt-get install make
WORKDIR /opt/go-dappley
COPY vm /opt/go-dappley/vm
WORKDIR /opt/go-dappley/vm/v8
RUN install ../lib/x86_64/*.so /usr/local/lib
RUN install ../lib/*.so /usr/local/lib
RUN ldconfig
WORKDIR /opt/go-dappley
COPY dapp/dapp dapp/dapp
COPY dapp/jslib dapp/jslib
COPY core/account/account.conf core/account/account.conf
WORKDIR /opt/go-dappley/dapp
EXPOSE 50051 12341
ENTRYPOINT  ["./dapp"]
