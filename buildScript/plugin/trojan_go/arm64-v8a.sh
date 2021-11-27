#!/bin/bash

source "buildScript/init/env.sh"
source "buildScript/plugin/trojan_go/build.sh"

DIR="$ROOT/arm64-v8a"
mkdir -p $DIR
env CC=$ANDROID_ARM64_CC GOARCH=arm64 go build -x -o $DIR/$LIB_OUTPUT -tags "client" -trimpath -ldflags="-s -w -buildid="
$ANDROID_ARM64_STRIP $DIR/$LIB_OUTPUT
