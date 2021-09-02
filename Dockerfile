FROM golang:1.11-alpine3.8 as builder

RUN \
    cd / && \
    apk update && \
    apk add --no-cache git ca-certificates make tzdata curl gcc libc-dev && \
    curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

RUN \
    mkdir -p src/github.com/krpn && \
    cd src/github.com/krpn && \
    git clone https://github.com/krpn/prometheus-alert-webhooker && \
    cd prometheus-alert-webhooker && \
    dep ensure -v && \
    go test ./... && \
    cd cmd/prometheus-alert-webhooker && \
    CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o prometheus-alert-webhooker


FROM alpine:3.8
COPY --from=builder /go/src/github.com/krpn/prometheus-alert-webhooker/cmd/prometheus-alert-webhooker/prometheus-alert-webhooker /
RUN apk add --no-cache ca-certificates tzdata curl
EXPOSE 8080
ENTRYPOINT ["/prometheus-alert-webhooker"]
