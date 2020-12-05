FROM golang:1-alpine as builder

RUN apk --no-cache --no-progress add git ca-certificates tzdata make \
    && update-ca-certificates \
    && rm -rf /var/cache/apk/*

WORKDIR /go/whoami

# Download go modules
COPY go.mod .
COPY go.sum .
RUN GO111MODULE=on GOPROXY=https://proxy.golang.org go mod download

COPY . .

RUN GOARCH={{ .GoARCH }} GOARM={{ .GoARM }} make build

FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/whoami/whoami .

ENTRYPOINT ["/whoami"]
EXPOSE 80
