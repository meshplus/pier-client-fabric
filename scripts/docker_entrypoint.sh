#!/usr/bin/env sh
set -e

APPCHAIN_NAME=$1

if [ -z "$CONFIG_FILE" ]
then
else
    echo $CONFIG_FILE | base64 -d > /root/.pier/config.tar.gz
    cd /root/.pier && tar xvf config.tar.gz && rm -f config.tar.gz
fi

pier --repo=/root/.pier appchain register --name=${APPCHAIN_NAME} --type=fabric --validators=/root/.pier/fabric/fabric.validators --desc="appchain for test" --version=1.4.3
pier --repo=/root/.pier rule deploy --path=/root/.pier/validating.wasm
pier --repo=/root/.pier start