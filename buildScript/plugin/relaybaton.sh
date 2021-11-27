#!/usr/bin/env bash

buildScript/plugin/relaybaton/init.sh &&
  buildScript/plugin/relaybaton/armeabi-v7a.sh &&
  buildScript/plugin/relaybaton/arm64-v8a.sh &&
  buildScript/plugin/relaybaton/x86.sh &&
  buildScript/plugin/relaybaton/x86_64.sh &&
  buildScript/plugin/relaybaton/end.sh
