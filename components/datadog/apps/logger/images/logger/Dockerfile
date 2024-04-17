FROM golang:1.21 AS builder

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go* ./
RUN set -eu && \
    go mod download; \
    go mod verify;

COPY . .
ENV CGO_ENABLED=0
RUN set -eu && \
    env CGO_ENABLED=1 go test -v -race ./...; \
    go build -v -o /usr/local/bin/app ./...;

# we want curl to also be available in the final image
FROM curlimages/curl

COPY --from=builder /usr/local/bin/app /app
EXPOSE 3333

ENTRYPOINT ["/app"]
