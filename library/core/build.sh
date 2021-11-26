#!/bin/bash

source .github/env.sh

chmod -R 777 build 2>/dev/null
rm -rf build 2>/dev/null

gomobile bind -v -cache $(realpath build) -trimpath -ldflags='-s -w' . || exit 1
rm -r libcore-sources.jar

proj=../SagerNet/app/libs
if [ -d $proj ]; then
  cp -f libcore.aar $proj
  echo ">> install $(realpath $proj)/libcore.aar"
fi