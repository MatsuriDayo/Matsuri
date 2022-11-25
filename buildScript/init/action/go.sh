#!/bin/bash
set -e

source buildScript/init/env.sh
mkdir -p $GOPATH
cd $golang

if [ ! -f "go/bin/go" ]; then
    curl -Lso go.tar.gz https://go.dev/dl/go1.18.8.linux-amd64.tar.gz
    echo "4d854c7bad52d53470cf32f1b287a5c0c441dc6b98306dea27358e099698142a go.tar.gz" | sha256sum -c -
    tar xzf go.tar.gz
fi

go version
go env
