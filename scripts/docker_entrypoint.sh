#!/usr/bin/env sh
set -e

APPCHAIN_NAME=$1

pier --repo=/root/.pier appchain register --name=${APPCHAIN_NAME} --type=fabric --validators=/root/.pier/fabric/fabric.validators --desc="appchain for test" --version=1.4.3
pier --repo=/root/.pier rule deploy --path=/root/.pier/validating.wasm
pier --repo=/root/.pier start