# Create a minimal container to run a Golang static binary
FROM scratch
COPY whoamI /
ENTRYPOINT ["/whoamI"]
EXPOSE 80
