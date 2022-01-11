#!/usr/bin/env bash

source "buildScript/init/env.sh"
export CGO_ENABLED=1
export GO386=softfloat

cd libcore
rel=1 ./build.sh || exit 1

# copy v2ray soucre to build/ for generating proto
TEMP_V2RAY_PATH="$PROJECT/build/v2ray-core"
chmod -R 777 $TEMP_V2RAY_PATH 2>/dev/null
rm -rf $TEMP_V2RAY_PATH 2>/dev/null
mkdir -p $TEMP_V2RAY_PATH
/bin/cp -r ../../v2ray-core/* $TEMP_V2RAY_PATH
chmod -R 777 $TEMP_V2RAY_PATH
