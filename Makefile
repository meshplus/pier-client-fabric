SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)
DISTRO = $(shell uname)
CURRENT_TAG =$(shell git describe --abbrev=0 --tags)

GO  = GO111MODULE=on go

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