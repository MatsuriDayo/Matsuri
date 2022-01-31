#!/bin/bash
set -e

source buildScript/init/env.sh
mkdir -p $GOPATH
cd $golang

if [ ! -f "go/bin/go" ]; then
    curl -Lso go.tar.gz https://go.dev/dl/go1.18beta2.linux-amd64.tar.gz
    echo "b5dacafa59737cfb0d657902b70c2ad1b6bb4ed15e85ea2806f72ce3d4824688 go.tar.gz" | sha256sum -c -
    tar xzf go.tar.gz
fi

go version
go env
