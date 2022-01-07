#!/usr/bin/env bash

source "buildScript/init/env.sh"

# fetch v2ray-core matsuri soucre
bash buildScript/lib/core/clone.sh

[ -f libcore/go.mod ] || exit 1
cd libcore

./init.sh || exit 1
