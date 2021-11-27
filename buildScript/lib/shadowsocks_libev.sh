#!/bin/bash

source "buildScript/init/env.sh"

bash "buildScript/zipVersion/downloadZip.sh" library/shadowsocks-libev/src/main/jni buildScript/zipVersion/shadowsocks_libev_name buildScript/zipVersion/shadowsocks_libev_status

rm -rf library/shadowsocks-libev/build/outputs/aar
./gradlew :library:shadowsocks-libev:assembleRelease || exit 1
mkdir -p app/libs
cp library/shadowsocks-libev/build/outputs/aar/shadowsocks-libev-release.aar app/libs/shadowsocks-libev.aar
