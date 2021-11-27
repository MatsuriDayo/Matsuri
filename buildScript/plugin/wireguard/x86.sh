#!/bin/bash

source "buildScript/init/env.sh"
source "buildScript/plugin/wireguard/build.sh"

DIR="$ROOT/x86"
mkdir -p $DIR
env CC=$ANDROID_X86_CC GOARCH=386 go build -x -o $DIR/$LIB_OUTPUT -trimpath -ldflags="-s -w -buildid="
$ANDROID_X86_STRIP $DIR/$LIB_OUTPUT