#!/bin/bash
set -e

source buildScript/init/env.sh
mkdir -p $GOPATH
cd $golang

if [ ! -f "go/bin/go" ]; then
    curl -Lso go.tar.gz https://go.dev/dl/go1.18.5.linux-amd64.tar.gz
    echo "9e5de37f9c49942c601b191ac5fba404b868bfc21d446d6960acc12283d6e5f2 go.tar.gz" | sha256sum -c -
    tar xzf go.tar.gz
fi

go version
go env
