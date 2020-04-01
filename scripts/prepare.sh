#!/usr/bin/env bash

BLUE='\033[0;34m'
NC='\033[0m'

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

print_blue "===> 1. Install golangci-lint"
if ! type golanci-lint >/dev/null 2>&1; then
  go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.23.0
fi