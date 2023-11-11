# syntax=docker/dockerfile:1.2
FROM golang:1-alpine as builder

RUN apk --no-cache --no-progress add git ca-certificates tzdata make \
    && update-ca-certificates \
    && rm -rf /var/cache/apk/*

# syntax=docker/dockerfile:1.2
# Create a minimal container to run a Golang static binary
FROM scratch

COPY --from=tarampampam/curl /bin/curl /curl
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY whoami /

HEALTHCHECK --interval=10s --start-period=2s CMD ["/curl", "--fail", "http://127.0.0.1:80/health"]
ENTRYPOINT ["/whoami"]
EXPOSE 80
