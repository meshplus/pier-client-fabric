#!/usr/bin/env sh
set -e

APPCHAIN_NAME=$1

if [ !-f /root/config/pier-fabric.property ]; then
  echo "Start pier with default config"
else
  echo "Start pier with custom config"
  base64 -d /root/config/pier-fabric.property > /root/config.tar.gz
  tar xvf /root/config.tar.gz -C /root/.pier
fi

pier --repo=/root/.pier appchain register --name=${APPCHAIN_NAME} --type=fabric --validators=/root/.pier/fabric/fabric.validators --desc="appchain for test" --version=1.4.3
pier --repo=/root/.pier rule deploy --path=/root/.pier/validating.wasm
export CONFIG_PATH=/root/.pier/fabric
pier --repo=/root/.pier start

tail -f /dev/null
