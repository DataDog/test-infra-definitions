FROM golang:1.21 AS builder

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.* ./
RUN set -eu && \
    go mod download; \
    go mod verify;

COPY *.go .
ENV CGO_ENABLED=0
RUN set -eu && \
    go test -v ./...; \
    go build -v -o app ./...;

# we want curl to also be available in the final image
FROM curlimages/curl

COPY --from=builder /usr/src/app/app /usr/local/bin/app
COPY send-message.sh /usr/local/bin/send-message.sh

EXPOSE 3333

ENTRYPOINT ["/usr/local/bin/app"]
