FROM golang:1.24.1-alpine3.20

RUN apk add --no-cache git ca-certificates curl

WORKDIR /src
COPY ./go.mod .
COPY ./go.sum .
COPY main.go .
RUN go build

ENTRYPOINT ["/src/websocket-receiver"]
