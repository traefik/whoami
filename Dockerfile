FROM golang:1.10 as builder
WORKDIR /go/src/github.com/containous/whoami
COPY . .
RUN go get -u github.com/golang/dep/cmd/dep
RUN make dependencies
RUN make build

# Create a minimal container to run a Golang static binary
FROM scratch
COPY --from=builder /go/src/github.com/containous/whoami/whoami .
ENTRYPOINT ["/whoami"]
EXPOSE 80
