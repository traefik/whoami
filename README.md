# whoamI

This is a fork from https://github.com/containous/whoami.

Tiny Go webserver that prints os information and HTTP request to output

```sh
$ docker run -d -P --name iamfoo bee42/whoami:1.2.0
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

## Endpoints

### 1.2.0

* `GET /` prints runtime information, ip addresses
* `GET /health` returns current healthstatus of the app
* `POST /health` sets current healthstatus of the app
* `GET /echo` prints an echo back
* `GET /api` prints runtime information, ip addresses in json
* `GET /bench` wait a long time before repsonse

### 2.0.0

* `GET /version` prints podinfo version and git commit hash

## Versions

Setting version in 2.0.0

```
$ docker run -d -p 8086:80 -e WHOAMI_VERSION=2.0.0-release bee42/whoami:2.0.0
$ curl 127.0.0.1:8086/version
```

## Metrics with prometheus

* `GET /metrics` get prometheus metrics form go process

### 2.1.0

```
$ docker run -d -p 8087:80 -e WHOAMI_VERSION=2.1.0-release bee42/whoami:2.1.0
$ curl 127.0.0.1:8087/
$ curl 127.0.0.1:8087/metrics
```

Currently the request of ´/api´ and ´/´ are measured.

### Links of metrics

* https://prometheus.io/docs/guides/go-application/
* https://povilasv.me/prometheus-go-metrics/
* https://github.com/alexellis/hash-browns
* https://blog.alexellis.io/prometheus-monitoring/
* https://alex.dzyoba.com/blog/go-prometheus-service/
* https://ordina-jworks.github.io/monitoring/2016/09/23/Monitoring-with-Prometheus.html

Regards
Peter Rossbach (peter.rossbach@bee42.com)