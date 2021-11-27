#!/bin/bash

source "buildScript/init/env.sh"

bash "buildScript/zipVersion/downloadZip.sh" library/shadowsocks/src/main/rust buildScript/zipVersion/shadowsocks_name buildScript/zipVersion/shadowsocks_status

rm -rf library/shadowsocks/build/outputs/aar
./gradlew :library:shadowsocks:assembleRelease || exit 1
mkdir -p app/libs
cp library/shadowsocks/build/outputs/aar/shadowsocks-release.aar app/libs/shadowsocks.aar
