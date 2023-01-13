#!/bin/bash
set -e

source buildScript/init/env.sh
mkdir -p $GOPATH
cd $golang

if [ ! -f "go/bin/go" ]; then
    curl -Lso go.tar.gz https://go.dev/dl/go1.19.5.linux-amd64.tar.gz
    echo "36519702ae2fd573c9869461990ae550c8c0d955cd28d2827a6b159fda81ff95 go.tar.gz" | sha256sum -c -
    tar xzf go.tar.gz
fi

go version
go env
