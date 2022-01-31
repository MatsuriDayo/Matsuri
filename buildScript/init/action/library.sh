#!/bin/bash

rm -rf external
mkdir -p external
cd external

# Download "external" from Internet

wget -q -O tmp.zip https://github.com/SagerNet/preferencex-android/archive/2bdf5a06bc242f5d5f01aa66b88ea640c938f243.zip
unzip tmp.zip > /dev/null 2>&1
mv preferencex-android-* preferencex

rm tmp.zip
