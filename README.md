# whoami

[![Docker Pulls](https://img.shields.io/docker/pulls/traefik/whoami.svg)](https://hub.docker.com/r/traefik/whoami/)
[![Build Status](https://github.com/traefik/whoami/workflows/Main/badge.svg?branch=master)](https://github.com/traefik/whoami/actions)

Tiny Go webserver that prints os information and HTTP request to output

## Usage

### Paths

- `/data?size=n[&unit=u]`: creates a response with a size `n`. The unit of measure, if specified, accepts the following values: `KB`, `MB`, `GB`, `TB` (optional, default: bytes).
- `/echo`: webSocket echo.
- `/bench`: always return the same response (`1`).
- `/[?wait=d]`: returns the whoami information (request and network information). The optional `wait` query parameter can be provided to tell the server to wait before sending the response. The duration is expected in Go's [`time.Duration`](https://golang.org/pkg/time/#ParseDuration) format (e.g. `/?wait=100ms` to wait 100 milliseconds).
- `/api`: returns the whoami information as JSON.
- `/health`: heath check
    - `GET`, `HEAD`, ...: returns a response with the status code defined by the `POST`
    - `POST`: changes the status code of the `GET` (`HEAD`, ...) response.

### Flags

- `cert`: give me a certificate.
- `key`: give me a key.
- `port`: give me a port number. (it can be also defined with `WHOAMI_PORT_NUMBER` environment variable) (default: 80)
- `name`: give me a name. (it can be also defined with `WHOAMI_NAME` environment variable)

## Examples

```console
$ docker run -d -P --name iamfoo traefik/whoami

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

```console
# updates health check status
$ curl -X POST -d '500' http://localhost:80/health

# calls the health check
$ curl -v http://localhost:80/health
*   Trying ::1:80...
* TCP_NODELAY set
* Connected to localhost (::1) port 80 (#0)
> GET /health HTTP/1.1
> Host: localhost:80
> User-Agent: curl/7.65.3
> Accept: */*
> 
* Mark bundle as not supporting multiuse
< HTTP/1.1 500 Internal Server Error
< Date: Mon, 16 Sep 2019 22:52:40 GMT
< Content-Length: 0
```

```console
docker run -d -P -v ./certs:/certs --name iamfoo traefik/whoami --cert /certs/cert.cer --key /certs/key.key
```

```compose
services:
  whoami:
    container_name: iamfoo
    image: traefik/whoami
    command: '--port 8080'
```
