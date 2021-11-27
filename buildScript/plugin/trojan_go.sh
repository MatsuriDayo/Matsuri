#!/usr/bin/env bash

buildScript/plugin/trojan_go/init.sh &&
  buildScript/plugin/trojan_go/armeabi-v7a.sh &&
  buildScript/plugin/trojan_go/arm64-v8a.sh &&
  buildScript/plugin/trojan_go/x86.sh &&
  buildScript/plugin/trojan_go/x86_64.sh &&
  buildScript/plugin/trojan_go/end.sh
