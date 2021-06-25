FROM golang:1.16 as builder

COPY . /data
WORKDIR /data

ARG DRONE_COMMIT=0
ARG DRONE_SYSTEM_HOST=unknown

RUN CGO_ENABLED=0 go build -ldflags="-X 'main.GitHash=$DRONE_COMMIT' -X 'main.BuildHost=$DRONE_SYSTEM_HOST' -X 'main.BuildTime=`date`'"

FROM alpine:3.13.1

RUN sed -i 's/dl-cdn/nl/' /etc/apk/repositories
RUN apk add -U --no-cache ca-certificates

COPY --from=builder /data/fruit-socks5-server /bin/fruit-socks5-server

ENTRYPOINT [ "/bin/fruit-socks5-server" ]
