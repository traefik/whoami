# whoami

[![Docker Pulls](https://img.shields.io/docker/pulls/traefik/whoami.svg)](https://hub.docker.com/r/traefik/whoami/)
[![Build Status](https://github.com/traefik/whoami/workflows/Main/badge.svg?branch=master)](https://github.com/traefik/whoami/actions)

Tiny Go webserver that prints OS information and HTTP request to output.

## Usage

### Paths

#### `/[?wait=d]`

Returns the whoami information (request and network information).

The optional `wait` query parameter can be provided to tell the server to wait before sending the response.
The duration is expected in Go's [`time.Duration`](https://golang.org/pkg/time/#ParseDuration) format (e.g. `/?wait=100ms` to wait 100 milliseconds).

#### `/api`

Returns the whoami information as JSON.

#### `/bench`

Always return the same response (`1`).

#### `/data?size=n[&unit=u]`

Creates a response with a size `n`.

The unit of measure, if specified, accepts the following values: `KB`, `MB`, `GB`, `TB` (optional, default: bytes).

#### `/echo`

WebSocket echo.

#### `/health`

Heath check.

- `GET`, `HEAD`, ...: returns a response with the status code defined by the `POST`
- `POST`: changes the status code of the `GET` (`HEAD`, ...) response.

### Flags

| Flag      | Env var              | Description                             |
|-----------|----------------------|-----------------------------------------|
| `cert`    |                      | Give me a certificate.                  |
| `key`     |                      | Give me a key.                          |
| `cacert`  |                      | Give me a CA chain, enforces mutual TLS |
| `port`    | `WHOAMI_PORT_NUMBER` | Give me a port number. (default: `80`)  |
| `name`    | `WHOAMI_NAME`        | Give me a name.                         |
| `verbose` |                      | Enable verbose logging.                 |

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
docker run -d -P -v ./certs:/certs --name iamfoo traefik/whoami --cert /certs/example.cert --key /certs/example.key
```

```yml
version: '3.9'

services:
  whoami:
    image: traefik/whoami
    command:
       # It tells whoami to start listening on 2001 instead of 80
       - --port 2001
       - --name iamfoo
```
