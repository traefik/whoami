FROM golang:1.11 as builder
WORKDIR /go/whoami
COPY . .
RUN make build

# Create a minimal container to run a Golang static binary
FROM scratch
COPY --from=builder /go/whoami/whoami .
ENTRYPOINT ["/whoami"]
EXPOSE 80
