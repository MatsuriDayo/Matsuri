#!/bin/bash

rm -rf external
mkdir -p external
cd external

# Download "external" from Internet

wget -q -O tmp.zip https://github.com/SagerNet/editorkit/archive/e7d1d0dca2c3e9b313f28dc34df1e12282f89206.zip
unzip tmp.zip > /dev/null 2>&1
mv editorkit-* editorkit

wget -q -O tmp.zip https://github.com/SagerNet/preferencex-android/archive/2bdf5a06bc242f5d5f01aa66b88ea640c938f243.zip
unzip tmp.zip > /dev/null 2>&1
mv preferencex-android-* preferencex

# wget -q -O tmp.zip https://github.com/SagerNet/termux-view/archive/c4127827ef013970bcd0f930f65b991bb12878f4.zip
# unzip tmp.zip > /dev/null 2>&1
# mv termux-view-* termux-view

rm tmp.zip
