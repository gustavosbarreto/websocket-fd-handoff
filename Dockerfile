# base stage
FROM golang:1.24.1-alpine3.20 AS builder

RUN apk add --no-cache git ca-certificates libgcc curl

WORKDIR /src
COPY ./go.mod .
COPY ./go.sum .
COPY main.go .
RUN go build

FROM alpine:3.21.3 AS production

RUN apk add libgcc

COPY --from=openresty/openresty:1.25.3.1-5-alpine-apk /usr/local/openresty /usr/local/openresty

COPY entrypoint.sh /
COPY nginx.conf /etc/nginx/nginx.conf
COPY lua /etc/nginx/lua

COPY --from=builder /src/websocket-receiver /

ENTRYPOINT ["/entrypoint.sh"]
