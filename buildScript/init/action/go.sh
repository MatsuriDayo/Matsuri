#!/bin/bash
set -e

source buildScript/init/env.sh
mkdir -p $GOPATH
cd $golang

if [ ! -f "go/bin/go" ]; then
    curl -Lso go.tar.gz https://go.dev/dl/go1.18.3.linux-amd64.tar.gz
    echo "956f8507b302ab0bb747613695cdae10af99bbd39a90cae522b7c0302cc27245 go.tar.gz" | sha256sum -c -
    tar xzf go.tar.gz
fi

go version
go env
