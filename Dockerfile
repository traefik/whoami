# Create a minimal container to run a Golang static binary
FROM scratch
COPY whoami /
ENTRYPOINT ["/whoami"]
EXPOSE 80
