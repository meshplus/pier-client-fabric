SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)
DISTRO = $(shell uname)
CURRENT_TAG =$(shell git describe --abbrev=0 --tags)

GO  = GO111MODULE=on go

help: Makefile
	@echo "Choose a command run:"
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'

prepare:
	cd scripts && bash prepare.sh

## make test-coverage: Test project with cover
test-coverage:
	@go test -short -coverprofile cover.out -covermode=atomic ${TEST_PKGS}
	@cat cover.out >> coverage.txt

## make fabric1.4: build fabric(1.4) client plugin
fabric1.4:
	mkdir -p build
	$(GO) build -o build/fabric-client-1.4 ./*.go

docker:
	mkdir -p build
	cd build && rm -rf pier && cp -r ../../pier pier
	cd ${CURRENT_PATH}
	docker build -t meshplus/pier-fabric .

fabric1.4-linux:
	cd scripts && sh cross_compile.sh linux-amd64 ${CURRENT_PATH}

release-binary:
	mkdir -p build
	$(GO) build -o build/fabric-client-${CURRENT_TAG}-${DISTRO} ./*.go

## make linter: Run golanci-lint
linter:
	golangci-lint run