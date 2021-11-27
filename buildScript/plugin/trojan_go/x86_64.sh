#!/bin/bash

source "buildScript/init/env.sh"
source "buildScript/plugin/trojan_go/build.sh"

DIR="$ROOT/x86_64"
mkdir -p $DIR
env CC=$ANDROID_X86_64_CC GOARCH=amd64 go build -x -o $DIR/$LIB_OUTPUT -tags "client" -trimpath -ldflags="-s -w -buildid="
$ANDROID_X86_64_STRIP $DIR/$LIB_OUTPUT
