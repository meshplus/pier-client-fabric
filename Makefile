SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)

GO  = GO111MODULE=on go

ifeq (docker,$(firstword $(MAKECMDGOALS)))
  DOCKER_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(DOCKER_ARGS):;@:)
endif

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
	$(GO) build --buildmode=plugin -o build/fabric-client-1.4.so ./*.go

docker:
	mkdir -p build
	cd build && rm -rf pier && git clone https://github.com/meshplus/pier.git && cd pier && git checkout $(DOCKER_ARGS)
	cd ${CURRENT_PATH}
	docker build -t meshplus/pier-fabric .

fabric1.4-linux:
	cd scripts && sh cross_compile.sh linux-amd64 ${CURRENT_PATH}

## make linter: Run golanci-lint
linter:
	golangci-lint run
