# syntax=docker/dockerfile:1
FROM golang:1-alpine AS builder

RUN apk --no-cache --no-progress add git ca-certificates tzdata make \
    && update-ca-certificates \
    && rm -rf /var/cache/apk/*

# Create a minimal container to run a Golang static binary
FROM scratch

ARG TARGETPLATFORM

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY $TARGETPLATFORM/whoami /

ENTRYPOINT ["/whoami"]
EXPOSE 80
