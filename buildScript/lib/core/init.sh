#!/usr/bin/env bash

source "buildScript/init/env.sh"

[ -f library/core/go.mod ] || exit 1
cd library/core

./init.sh || exit 1
