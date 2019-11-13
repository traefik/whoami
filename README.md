# whoami

[![Docker Pulls](https://img.shields.io/docker/pulls/containous/whoami.svg)](https://hub.docker.com/r/containous/whoami/)
[![Build Status](https://travis-ci.com/containous/whoami.svg?branch=master)](https://travis-ci.com/containous/whoami)

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
- `port`: give me a port number. (default: 80)


## Environment

Optionally run with environment variable `BLUE_GREEN`, the content will be displayed in the `/` call.

```console
$ docker run -d -e "BLUE_GREEN=blue" -p 8080:80 whoami

$ curl localhost:8080/whoami
Environment (BLUE_GREEN): blue
Hostname: 88e638611554
IP: 127.0.0.1
IP: 172.17.0.2
RemoteAddr: 172.17.0.1:49464
GET /whoami HTTP/1.1
Host: localhost:8080
User-Agent: curl/7.64.1
Accept: */*
```
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
