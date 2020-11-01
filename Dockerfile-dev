FROM golang:alpine

RUN apk add --no-cache git make build-base

COPY go.sum /opt/go.sum
COPY go.mod /opt/go.mod
RUN cd /opt && go mod download

WORKDIR /opt/program