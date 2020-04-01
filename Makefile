SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)

GO  = GO111MODULE=on go

help: Makefile
	@echo "Choose a command run:"
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'

prepare:
	cd scripts && bash prepare.sh

## make fabric1.4: build fabric(1.4) client plugin
fabric1.4:
	mkdir -p build
	$(GO) build --buildmode=plugin -o build/fabric-client-1.4.so ./*.go

fabric1.4-linux:
	cd scripts && sh cross_compile.sh linux-amd64 ${CURRENT_PATH}

## make linter: Run golanci-lint
linter:
	golangci-lint run

