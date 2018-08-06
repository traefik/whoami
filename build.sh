#!/bin/sh
CGO_ENABLED=0 go build -a --installsuffix cgo --ldflags="-s" -o whoami
docker build -t containous/whoami .
