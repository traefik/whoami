#!/bin/sh
CGO_ENABLED=0 go build -a --installsuffix cgo --ldflags="-s" -o whoamI
docker build -t emilevauge/whoami .
