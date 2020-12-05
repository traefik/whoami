.PHONY: default build image check publish-images

TAG_NAME := $(shell git tag -l --contains HEAD)

IMAGE_NAME := traefik/whoami

default: check test build

build:
	CGO_ENABLED=0 go build -a --trimpath --installsuffix cgo --ldflags="-s" -o whoami

test:
	go test -v -cover ./...

check:
	golangci-lint run

image:
	docker build -t $(IMAGE_NAME) .

publish-images:
	seihon publish -v "$(TAG_NAME)" -v "latest" --image-name $(IMAGE_NAME) --dry-run=false
