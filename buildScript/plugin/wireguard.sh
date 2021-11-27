#!/usr/bin/env bash

buildScript/plugin/wireguard/init.sh &&
  buildScript/plugin/wireguard/armeabi-v7a.sh &&
  buildScript/plugin/wireguard/arm64-v8a.sh &&
  buildScript/plugin/wireguard/x86.sh &&
  buildScript/plugin/wireguard/x86_64.sh &&
  buildScript/plugin/wireguard/end.sh
