#!/bin/bash

source .github/env.sh

chmod -R 777 build 2>/dev/null
rm -rf build 2>/dev/null

go get -v -d

# Install gomobile
if [ ! -f "$GOPATH/bin/gomobile" ]; then
    go install -v github.com/sagernet/gomobile/cmd/gomobile@v0.0.0-20210905032500-701a995ff844
    go install -v github.com/sagernet/gomobile/cmd/gobind@v0.0.0-20210905032500-701a995ff844
fi

gomobile init
