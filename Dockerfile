ARG GOLANG_TARGET=${GOLANG_TARGET:-golang:1.11.0-stretch}
ARG TARGET=${TARGET:-alpine:3.8}
FROM ${GOLANG_TARGET} as build

ENV REPO=github.com/bee42/whoamI \
    OUTPUT_PATH=/output

RUN apt-get update && apt-get install -y --no-install-recommends git make curl && \
    curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

ARG ARCH=${ARCH:-amd64}
ARG OS=${OS:-linux}
WORKDIR $GOPATH/src/${REPO}
COPY . .
RUN mkdir -p vendor && dep ensure
RUN mkdir -p $OUTPUT_PATH && \
    GOOS=${OS} GOARCH=${ARCH} CGO_ENABLED=0 go build -a --installsuffix cgo --ldflags="-s" -o $OUTPUT_PATH/whoamI

FROM ${TARGET}
LABEL maintainer nicals.mietz@bee42.com
LABEL maintainer peter.rossbach@bee42.com
COPY --from=build /output/whoamI /whoamI
ENTRYPOINT ["/whoamI"]
EXPOSE 80
