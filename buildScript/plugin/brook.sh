#!/usr/bin/env bash

buildScript/plugin/brook/init.sh &&
  buildScript/plugin/brook/armeabi-v7a.sh &&
  buildScript/plugin/brook/arm64-v8a.sh &&
  buildScript/plugin/brook/x86.sh &&
  buildScript/plugin/brook/x86_64.sh &&
  buildScript/plugin/brook/end.sh
