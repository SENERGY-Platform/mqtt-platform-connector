FROM golang:1.12


COPY . /go/src/mqtt-platform-connector
WORKDIR /go/src/mqtt-platform-connector

ENV GO111MODULE=on

RUN go build

EXPOSE 8080

CMD ./mqtt-platform-connector