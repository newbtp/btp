# Build Gbtp in a stock Go builder container
FROM golang:1.12-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /go-btpereum
RUN cd /go-btpereum && make gbtp

# Pull Gbtp into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-btpereum/build/bin/gbtp /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["gbtp"]
