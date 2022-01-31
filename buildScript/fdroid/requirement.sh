#!/bin/bash

# run as root

apt-get update || apt-get update
apt-get install -y openjdk-11-jdk-headless wget
update-alternatives --auto java

buildScript/init/action/go.sh
