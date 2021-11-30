#!/usr/bin/env bash

source "buildScript/init/env.sh"

# fetch v2ray-core matsuri soucre
bash buildScript/lib/core/clone.sh

[ -f library/core/go.mod ] || exit 1
cd library/core

./init.sh || exit 1
