#!/bin/bash

# Setup go & external library
buildScript/init/action/go.sh
buildScript/init/action/library.sh

# Build libcore
buildScript/lib/core.sh
