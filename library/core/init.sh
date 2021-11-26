#!/bin/bash

source .github/env.sh

go mod download -x
go get -v github.com/sagernet/gomobile/cmd/gomobile@v0.0.0-20210905032500-701a995ff844
go get -v github.com/sagernet/gomobile/cmd/gobind@v0.0.0-20210905032500-701a995ff844
gomobile init
