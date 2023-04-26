SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)
DISTRO = $(shell uname)
CURRENT_TAG =$(shell git describe --abbrev=0 --tags)

GO  = GO111MODULE=on go
GREEN=\033[0;32m
NC=\033[0m

ifndef (${TAG})
  TAG = latest
endif

help: Makefile
	@echo "Choose a command run:"
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'

prepare:
	cd scripts && bash prepare.sh

## make test-coverage: Test project with cover
test-coverage: prepare
	@go test -short -coverprofile cover.out -covermode=atomic ${TEST_PKGS}
	@cat cover.out >> coverage.txt

## make fabric1.4: build fabric(1.4) client plugin
fabric1.4:
	@packr2
	mkdir -p build
	$(GO) build -o build/fabric-client-1.4 ./*.go
	@printf "${GREEN}Build fabric-client-1.4 successfully!${NC}\n"

## make build-docker: docker build the project
build-docker:
	docker build -t meshplus/pier-fabric:${TAG} .
	@printf "${GREEN}Build images meshplus/pier-fabric:${TAG} successfully!${NC}\n"

fabric1.4-linux:
	cd scripts && sh cross_compile.sh linux-amd64 ${CURRENT_PATH}

release-binary:
	mkdir -p build
	$(GO) build -o build/fabric-client-${CURRENT_TAG}-${DISTRO} ./*.go

## make linter: Run golanci-lint
linter:
	golangci-lint run


contract-zip:
	@rm -rf tmp && mkdir tmp && cd tmp && unzip -q ../example/contracts.zip
	@cd example/contracts/src/broker && cp -r ../../../../tmp/contracts/src/broker/vendor ./
	@cd example/contracts/src/data_swapper && cp -r ../../../../tmp/contracts/src/data_swapper/vendor ./
	@cd example/contracts/src/transfer && cp -r ../../../../tmp/contracts/src/transfer/vendor ./
	@cd example/contracts/src/transaction && cp -r ../../../../tmp/contracts/src/transaction/vendor ./
	@rm -rf tmp example/contracts.zip
	@cd example && zip -r -q contracts.zip contracts
	@printf "${GREEN}Build contracts.zip successfully!${NC}\n"
