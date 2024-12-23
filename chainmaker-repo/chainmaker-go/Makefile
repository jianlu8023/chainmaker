ifeq ($(OS),Windows_NT)
    PLATFORM="Windows"
else
    ifeq ($(shell uname),Darwin)
        PLATFORM="MacOS"
    else
        PLATFORM="Linux"
    endif
endif
DATETIME=$(shell date "+%Y%m%d%H%M%S")
VERSION=v2.3.0
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT = $(shell git log --pretty=format:'%h' -n 1)

LOCALCONF_HOME=chainmaker.org/chainmaker-go/module/blockchain
GOLDFLAGS += -X "${LOCALCONF_HOME}.CurrentVersion=${VERSION}"
GOLDFLAGS += -X "${LOCALCONF_HOME}.BuildDateTime=${DATETIME}"
GOLDFLAGS += -X "${LOCALCONF_HOME}.GitBranch=${GIT_BRANCH}"
GOLDFLAGS += -X "${LOCALCONF_HOME}.GitCommit=${GIT_COMMIT}"

chainmaker:
    ifeq ($(PLATFORM),"Windows")
		@echo "build for windows"
		@rm -rf go.sum && cd main && go mod tidy && go build -ldflags '${GOLDFLAGS}' -o ../bin/chainmaker.exe
    else
		@echo "build for linux or mac"
		@rm -rf go.sum && cd main && go mod tidy && go build -ldflags '${GOLDFLAGS}' -o ../bin/chainmaker
    endif

chainmaker-vendor:
	@cd main && go build -mod=vendor -o ../bin/chainmaker

package:
	@cd main && go mod tidy && GOPATH=${GOPATH} go build -ldflags '${GOLDFLAGS}' -o ../bin/chainmaker
	@mkdir -p ./release
	@rm -rf ./tmp/chainmaker/
	@mkdir -p ./tmp/chainmaker/
	@mkdir ./tmp/chainmaker/bin
	@mkdir ./tmp/chainmaker/config
	@mkdir ./tmp/chainmaker/log
	@cp bin/chainmaker ./tmp/chainmaker/bin
	@cp -r config ./tmp/chainmaker/
	@cd ./tmp;tar -zcvf chainmaker-$(VERSION).$(DATETIME).$(PLATFORM).tar.gz chainmaker; mv chainmaker-$(VERSION).$(DATETIME).$(PLATFORM).tar.gz ../release
	@rm -rf ./tmp/

compile:
	@cd main && go mod tidy && go build -ldflags '${GOLDFLAGS}' -o ../bin/chainmaker

cmc:
	@cd tools/cmc && go mod tidy && go build -ldflags '${GOLDFLAGS}' -o ../../bin/cmc

send-tool:
	cd test/send_proposal_request_tool && go build -o ../../bin/send_proposal_request_tool

scanner:
	@cd tools/scanner && GOPATH=${GOPATH} go build -o ../../bin/scanner

dep:
	@go get golang.org/x/tools/cmd/stringer

generate:
	go generate ./...

docker-build:
	rm -rf build/ data/ log/ bin/
	docker build -t chainmaker -f ./DOCKER/Dockerfile .
	docker tag chainmaker chainmaker:${VERSION}

docker-build-dev: chainmaker
	docker build -t chainmaker -f ./DOCKER/dev.Dockerfile .
	docker tag chainmaker chainmaker:${VERSION}

docker-compose-start: docker-compose-stop
	docker-compose up -d

docker-compose-stop:
	docker-compose down

ut:
	cd scripts && ./ut_cover.sh

lint:
	cd main && golangci-lint run ./...
	cd module/accesscontrol && golangci-lint run .
	cd module/blockchain && golangci-lint run .
	cd module/core && golangci-lint run ./...
	cd module/consensus && golangci-lint run ./...
	cd module/net && golangci-lint run ./...
	cd module/rpcserver && golangci-lint run ./...
	cd module/snapshot && golangci-lint run ./...
	cd module/subscriber && golangci-lint run ./...
	cd module/sync && golangci-lint run ./...
	cd module/txfilter && golangci-lint run ./...
	golangci-lint run ./tools/cmc/...
	cd tools/scanner && golangci-lint run ./...

gomod:
	cd scripts && ./gomod_update.sh

test-deploy:
	cd scripts/test/ && ./quick_deploy.sh

sql-qta:
	echo "clear environment"
	cd test/chain2 && ./stop.sh
	cd test/chain2 && ./clean.sh
	echo "start new sql-qta test"
	cd test/chain2 && ./build.sh
	cd test/chain2 && ./start.sh
	cd test/scenario0_native && python3 chain2.py
	cd test/scenario1_evm && python3 chain2.py
	cd test/scenario2_rust && python3 chain2.py
	cd test/scenario4_wasmer_sql && python3 chain2.py
	cd test/chain2 && ./stop.sh
	cd test/chain2 && ./clean.sh

qta: cert-qta pub-qta docker-qta

cert-qta:
	echo "clear environment"
	cd test/chain1 && ./stop.sh
	cd test/chain1 && ./clean.sh
	echo "start new qta test"
	cd test/chain1 && ./build.sh
	cd test/chain1 && ./start.sh
	cd test/scenario0_native && python3 chain1.py
	cd test/scenario1_evm && python3 chain1.py
	cd test/scenario2_rust && python3 chain1.py
	cd test/chain1 && ./stop.sh
	cd test/chain1 && ./clean.sh

pub-qta:
	echo "clear environment"
	cd test/chain3 && ./stop.sh
	cd test/chain3 && ./clean.sh
	echo "start new qta test"
	cd test/chain3 && ./build.sh
	cd test/chain3 && ./start.sh
	cd test/scenario0_native && python3 chain3.py
	cd test/scenario1_evm && python3 chain3.py
	#cd test/scenario2_rust && python3 chain3.py  #Rust合约不能启用Gas
	cd test/chain3 && ./stop.sh
	cd test/chain3 && ./clean.sh

docker-qta:
	echo "clear environment"
	cd test/chain1 && ./stop.sh
	cd test/chain1 && ./clean.sh
	echo "start new docker-qta test"
	cd scripts/docker && ./build-dockergo.sh
	cd test/chain1 && ./build.sh
	cd test/chain1 && ./docker-start.sh
	cd test/chain1 && ./start.sh
	cd test/scenario3_dockergo && python3 chain1.py
	cd test/chain1 && ./stop.sh
	cd test/chain1 && ./clean.sh
	docker rm -f  `docker ps -aq -f name=ci-chain1`