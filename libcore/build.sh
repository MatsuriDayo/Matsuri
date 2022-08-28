#!/bin/bash

source ../buildScript/init/env_ndk.sh

[ $rel ] || sed -i "s/buildDate .*/buildDate := \"`date +'%Y%m%d'`\"/g" date.go

BUILD=".build"

rm -rf $BUILD/android \
  $BUILD/java \
  $BUILD/javac-output \
  $BUILD/src

gomobile bind -v -androidapi 21 -cache $(realpath $BUILD) -trimpath -ldflags='-s -w' . || exit 1
rm -r libcore-sources.jar

proj=../app/libs
mkdir -p $proj
cp -f libcore.aar $proj
echo ">> install $(realpath $proj)/libcore.aar"
