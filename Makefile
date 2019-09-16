.PHONY: default build image check

default: check test build

test:
	go test -v -cover ./...

build:
	CGO_ENABLED=0 go build -a --installsuffix cgo --ldflags="-s" -o whoami

image:
	docker build -t containous/whoami .

check:
	golangci-lint run
