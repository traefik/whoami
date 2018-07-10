# whoamI

Tiny Go webserver that prints os information and HTTP request to output

```sh
$ docker run -d -P --name iamfoo emilevauge/whoami
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
docker run -it --rm -p 8086:80 -e WHOAMI_VERSION=2.0.0-release emilevauge/whoami:2.0.0
```
