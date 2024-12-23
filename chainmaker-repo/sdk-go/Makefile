build:
	go build ./...

lint:
	golangci-lint run ./...

gomod:
	go get chainmaker.org/chainmaker/pb-go/v2@develop
	go get chainmaker.org/chainmaker/common/v2@develop
.PHONY: build lint
