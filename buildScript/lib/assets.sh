#!/bin/bash

set -e

DIR=app/src/main/assets/v2ray
rm -rf $DIR
mkdir -p $DIR
cd $DIR

get_latest_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" | # Get latest release from GitHub api
    grep '"tag_name":' |                                            # Get tag line
    sed -E 's/.*"([^"]+)".*/\1/'                                    # Pluck JSON value
}

####
VERSION_V2RAY=`get_latest_release "v2fly/v2ray-core"`
echo VERSION_V2RAY=$VERSION_V2RAY
echo -n $VERSION_V2RAY > core.version.txt
curl -fLSsO https://github.com/v2fly/v2ray-core/releases/download/$VERSION_V2RAY/v2ray-extra.zip
unzip v2ray-extra.zip
mv browserforwarder/* .
xz index.js

####
VERSION_GEOIP=`get_latest_release "v2fly/geoip"`
echo VERSION_GEOIP=$VERSION_GEOIP
echo -n $VERSION_GEOIP > geoip.version.txt
curl -fLSsO https://github.com/v2fly/geoip/releases/download/$VERSION_GEOIP/geoip.dat
xz geoip.dat

####
VERSION_GEOSITE=`get_latest_release "v2fly/domain-list-community"`
echo VERSION_GEOSITE=$VERSION_GEOSITE
echo -n $VERSION_GEOSITE > geosite.version.txt
curl -fLSs -o geosite.dat.xz https://github.com/v2fly/domain-list-community/releases/download/$VERSION_GEOSITE/dlc.dat.xz

####
rm -rf browserforwarder *.zip
