#!/usr/bin/env bash

source "buildScript/init/env.sh"
export CGO_ENABLED=1
export GO386=softfloat

cd library/core
./build.sh || exit 1

mkdir -p "$PROJECT/app/libs"
/bin/cp -f libcore.aar "$PROJECT/app/libs"

# copy v2ray soucre to sagernet build dir.
mkdir -p "$PROJECT/build"
TEMP_V2RAY_PATH="$PROJECT/build/v2ray-core"
chmod -R 777 $TEMP_V2RAY_PATH 2>/dev/null
rm -rf $TEMP_V2RAY_PATH 2>/dev/null
/bin/cp -r build/pkg/mod/github.com/sagernet/v2ray-core/* $TEMP_V2RAY_PATH
chmod -R 777 $TEMP_V2RAY_PATH
