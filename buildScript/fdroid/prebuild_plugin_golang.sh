#!/bin/bash

git submodule update --init "plugin/$1"

buildScript/fdroid/install_golang.sh
