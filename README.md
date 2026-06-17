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

The optional `env` query parameter can be set to `true` to add the environment variables to the response.

#### `/api`

Returns the whoami information (and some extra information) as JSON.

The optional `env` query parameter can be set to `true` to add the environment variables to the response.

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

## Observability

whoami is instrumented with [OpenTelemetry](https://opentelemetry.io). It can emit
**traces**, **metrics**, and **logs** (including per-request access logs), configured
entirely through the standard `OTEL_*` environment variables.

Each signal is controlled independently by its standard exporter variable:

| Variable                | Default   | Description                                                              |
|-------------------------|-----------|--------------------------------------------------------------------------|
| `OTEL_LOGS_EXPORTER`    | `console` | Where logs go. `console` prints to stdout, `otlp` ships them, `none` off. |
| `OTEL_TRACES_EXPORTER`  | `none`    | Set to `otlp` (or `console`) to emit traces. Off by default.             |
| `OTEL_METRICS_EXPORTER` | `none`    | Set to `otlp` (or `console`) to emit metrics. Off by default.            |

So **logs print to stdout out of the box**, while traces and metrics are opt-in — you
turn them on when you have somewhere to send them. (`stdout` is accepted as an alias for
`console`.)

Endpoint, protocol, headers, service name, and resource attributes use the usual OTel
variables, for example:

| Variable                       | Description                                                       |
|--------------------------------|-------------------------------------------------------------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT`  | OTLP collector endpoint, e.g. `http://collector:4318`.            |
| `OTEL_EXPORTER_OTLP_PROTOCOL`  | `grpc`, `http/protobuf`, or `http/json`.                          |
| `OTEL_SERVICE_NAME`            | Service name reported to the backend (default: `whoami`).         |
| `OTEL_RESOURCE_ATTRIBUTES`     | Extra resource attributes, e.g. `deployment.environment=prod`.    |

What gets emitted:

- **Traces** — a server span per HTTP request (via `otelhttp`) and per gRPC call (via
  `otelgrpc`).
- **Metrics** — standard HTTP server metrics (`http.server.*`), gRPC server metrics
  (`rpc.server.*`), and a custom `whoami.requests` counter labelled by method and status.
- **Logs** — application logs plus a structured **access log** per request (method, path,
  status, response size, duration). Access logs are emitted within the request span, so
  each carries the active trace and span IDs.

### Example: ship everything to a collector

```console
$ docker run -d -p 8080:80 --name iamfoo \
    -e OTEL_TRACES_EXPORTER=otlp \
    -e OTEL_METRICS_EXPORTER=otlp \
    -e OTEL_LOGS_EXPORTER=otlp \
    -e OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318 \
    -e OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf \
    -e OTEL_SERVICE_NAME=whoami \
    traefik/whoami
```

With no `OTEL_*` variables set, whoami simply prints structured logs (including access
logs) to stdout and ships nothing.

## Examples

```console
$ docker run -d -p 8080:80 --name iamfoo traefik/whoami

$ curl http://localhost:8080
Hostname: 9c9c93da54b5
IP: 127.0.0.1
IP: ::1
IP: 172.17.0.2
RemoteAddr: 172.17.0.1:41040
GET / HTTP/1.1
Host: localhost:8080
User-Agent: curl/8.5.0
Accept: */*
```

```console
# updates health check status
$ curl -X POST -d '500' http://localhost:8080/health

# calls the health check
$ curl -v http://localhost:8080/health
* Host localhost:8080 was resolved.
* IPv6: ::1
* IPv4: 127.0.0.1
*   Trying [::1]:8080...
* Connected to localhost (::1) port 8080
> GET /health HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/8.5.0
> Accept: */*
> 
< HTTP/1.1 500 Internal Server Error
< Date: Fri, 18 Apr 2025 13:36:02 GMT
< Content-Length: 0
```

```console
$ openssl req -newkey rsa:4096 \
    -x509 \
    -sha256 \
    -days 3650 \
    -nodes \
    -out ./certs/example.crt \
    -keyout ./certs/example.key

$ docker run -d -p 8080:80 -v ./certs:/certs --name iamfoo traefik/whoami --cert /certs/example.crt --key /certs/example.key

$ curl https://localhost:8080 -k --cert certs/example.crt  --key certs/example.key
Hostname: 25bc0df47b95
IP: 127.0.0.1
IP: ::1
IP: 172.17.0.2
RemoteAddr: 172.17.0.1:50278
Certificate[0] Subject: CN=traefik.io,O=TraefikLabs,L=Lyon,ST=France,C=FR
GET / HTTP/1.1
Host: localhost:8080
User-Agent: curl/8.5.0
Accept: */*
```

```console
$ docker run -d -p 8080:80 --name iamfoo traefik/whoami

$ grpcurl -plaintext -proto grpc.proto localhost:8080 whoami.Whoami/Whoami
{
  "hostname": "5a45e21984b4",
  "iface": [
    "127.0.0.1",
    "::1",
    "172.17.0.2"
  ]
}

$ grpcurl -plaintext -proto grpc.proto localhost:8080 whoami.Whoami/Bench
{
  "data": 1
}
```

```yml
version: '3.9'

services:
  whoami:
    image: traefik/whoami
    command:
       # It tells whoami to start listening on 2001 instead of 80
       - --port=2001
       - --name=iamfoo
```