#!/bin/bash

if [ -z "$ANDROID_HOME" ]; then
  if [ -d "$HOME/Android/Sdk" ]; then
    export ANDROID_HOME="$HOME/Android/Sdk"
  elif [ -d "$HOME/.local/lib/android/sdk" ]; then
    export ANDROID_HOME="$HOME/.local/lib/android/sdk"
  elif [ -d "$HOME/Library/Android/sdk" ]; then
    export ANDROID_HOME="$HOME/Library/Android/sdk"
  fi
fi

_NDK="$ANDROID_HOME/ndk/23.1.7779620"
[ -f "$_NDK/source.properties" ] || _NDK="$ANDROID_NDK_HOME"
[ -f "$_NDK/source.properties" ] || _NDK="$NDK"
[ -f "$_NDK/source.properties" ] || _NDK="$ANDROID_HOME/ndk-bundle"
[ -f "$_NDK/source.properties" ] || _NDK="$ANDROID_HOME/23.1.7779620"

if [ ! -f "$_NDK/source.properties" ]; then
  echo "Error: NDK not found."
  exit 1
fi

export ANDROID_NDK_HOME=$_NDK
export NDK=$_NDK

if [ ! $(command -v go) ]; then
  if [ -d /usr/lib/go ]; then
    export PATH="$PATH:/usr/lib/go/bin"
  elif [ /usr/lib/go-1.17 ]; then
    export PATH="$PATH:/usr/lib/go-1.17/bin"
  elif [ -d $HOME/.go ]; then
    export PATH="$PATH:$HOME/.go/bin"
  fi
fi

if [ $(command -v go) ]; then
  export PATH="$PATH:$(go env GOPATH)/bin"
fi
