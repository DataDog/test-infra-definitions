FROM golang:1.26.1

WORKDIR /build

ADD . /build

RUN go mod tidy && go build -o fips-server ./src
RUN chmod +x ./run.sh

ENTRYPOINT ["./run.sh"]
EXPOSE 443/tcp
