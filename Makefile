.PHONY: default build image

default: build

build:
	CGO_ENABLED=0 go build -a --installsuffix cgo --ldflags="-s" -o whoami

image:
	docker build -t containous/whoami .
