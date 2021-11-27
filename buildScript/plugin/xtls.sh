#!/usr/bin/env bash

buildScript/plugin/xtls/init.sh &&
  buildScript/plugin/xtls/armeabi-v7a.sh &&
  buildScript/plugin/xtls/arm64-v8a.sh &&
  buildScript/plugin/xtls/x86.sh &&
  buildScript/plugin/xtls/x86_64.sh &&
  buildScript/plugin/xtls/end.sh
