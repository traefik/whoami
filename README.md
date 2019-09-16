# whoami

[![Docker Pulls](https://img.shields.io/docker/pulls/containous/whoami.svg)](https://hub.docker.com/r/containous/whoami/)
[![Build Status](https://travis-ci.com/containous/whoami.svg?branch=master)](https://travis-ci.com/containous/whoami)

Tiny Go webserver that prints os information and HTTP request to output

## Usage

### Paths

- `/data?size=n`: creates a response with a size `n`.
- `/echo`: webSocket echo.
- `/bench`: always return the same response (`1`).
- `/`: returns the whoami information (request and network information).
- `/api`: returns the whoami information as JSON.
- `/health`: heath check
    - `GET`, `HEAD`, ...: returns a response with the status code defined by the `POST`
    - `POST`: changes the status code of the `GET` (`HEAD`, ...) response.

### Flags

- `cert`: give me a certificate.
- `key`: give me a key.
- `port`: give me a port number. (default: 80)

## Examples

```console
$ docker run -d -P --name iamfoo containous/whoami
$ docker inspect --format '{{ .NetworkSettings.Ports }}'  iamfoo
map[80/tcp:[{0.0.0.0 32769}]]
$ curl "http://0.0.0.0:32769"
Hostname :  6e0030e67d6a
IP :  127.0.0.1
IP :  ::1
IP :  172.17.0.27
IP :  fe80::42:acff:fe11:1b
GET / HTTP/1.1
Host: 0.0.0.0:32769
User-Agent: curl/7.35.0
Accept: */*
```
