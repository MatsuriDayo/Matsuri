#!/bin/bash

buildScript/plugin/naive/init.sh &&
  buildScript/plugin/naive/armeabi-v7a.sh &&
  buildScript/plugin/naive/arm64-v8a.sh &&
  buildScript/plugin/naive/x86.sh &&
  buildScript/plugin/naive/x86_64.sh &&
  buildScript/plugin/naive/end.sh
