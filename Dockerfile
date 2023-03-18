FROM golang:1.20 AS builder
WORKDIR /go/src/scout
COPY . /go/src/scout/
RUN go build


FROM ubuntu:latest
LABEL MAINTAINER="https://github.com/airdb"

COPY --from=builder /go/src/scout/ /srv/scout
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /srv/scout
CMD /srv/scout/scout
