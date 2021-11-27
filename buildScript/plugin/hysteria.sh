#!/usr/bin/env bash

buildScript/plugin/hysteria/init.sh &&
  buildScript/plugin/hysteria/armeabi-v7a.sh &&
  buildScript/plugin/hysteria/arm64-v8a.sh &&
  buildScript/plugin/hysteria/x86.sh &&
  buildScript/plugin/hysteria/x86_64.sh &&
  buildScript/plugin/hysteria/end.sh
