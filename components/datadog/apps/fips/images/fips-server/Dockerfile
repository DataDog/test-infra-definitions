FROM golang:1.22.5

WORKDIR /build

ADD . /build

RUN go mod tidy && go build -o fips-server ./src
RUN chmod +x ./run.sh

ENTRYPOINT ["./run.sh"]
EXPOSE 443/tcp
