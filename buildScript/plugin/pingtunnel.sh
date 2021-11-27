#!/bin/bash

buildScript/plugin/pingtunnel/init.sh &&
  buildScript/plugin/pingtunnel/armeabi-v7a.sh &&
  buildScript/plugin/pingtunnel/arm64-v8a.sh &&
  buildScript/plugin/pingtunnel/x86.sh &&
  buildScript/plugin/pingtunnel/x86_64.sh &&
  buildScript/plugin/pingtunnel/end.sh
