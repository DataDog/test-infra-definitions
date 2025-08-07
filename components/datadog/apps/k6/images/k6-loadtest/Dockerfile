# Build k6 with the xk6 statsd output extension
FROM golang:1.24 AS builder

ENV GOOS=linux

RUN go install go.k6.io/xk6/cmd/xk6@latest \
    && /go/bin/xk6 build \
    --with github.com/LeonAdato/xk6-output-statsd@latest \
    --output /k6

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /k6 /usr/bin/k6

# Copy the k6 test script into the container
COPY script.js /home/k6/script.js

ENTRYPOINT ["k6"] 