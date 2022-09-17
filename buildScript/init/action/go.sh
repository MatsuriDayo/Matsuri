#!/bin/bash
set -e

source buildScript/init/env.sh
mkdir -p $GOPATH
cd $golang

if [ ! -f "go/bin/go" ]; then
    curl -Lso go.tar.gz https://go.dev/dl/go1.18.6.linux-amd64.tar.gz
    echo "bb05f179a773fed60c6a454a24141aaa7e71edfd0f2d465ad610a3b8f1dc7fe8 go.tar.gz" | sha256sum -c -
    tar xzf go.tar.gz
fi

go version
go env
