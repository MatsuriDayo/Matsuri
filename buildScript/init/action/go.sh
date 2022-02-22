#!/bin/bash
set -e

source buildScript/init/env.sh
mkdir -p $GOPATH
cd $golang

if [ ! -f "go/bin/go" ]; then
    curl -Lso go.tar.gz https://go.dev/dl/go1.18rc1.linux-amd64.tar.gz
    echo "9ea4e6adee711e06fa95546e1a9629b63de3aaae85fac9dc752fb533f3e5be23 go.tar.gz" | sha256sum -c -
    tar xzf go.tar.gz
fi

go version
go env
