FROM golang:alpine AS build-env

RUN mkdir /opt/program
WORKDIR /opt/program

RUN apk update
RUN apk add build-base ca-certificates git

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

WORKDIR /opt/program/cmd/socketsrv

ARG BUILD_HASH=unknown
ARG BUILD_DATE=unknown

RUN go build -ldflags  "-X=main.BuildDate=${BUILD_DATE} -X=main.BuildHash=${BUILD_HASH}"

# Build minimal image:
FROM alpine
COPY --from=build-env /opt/program/cmd/socketsrv/socketsrv /opt/socketsrv
RUN apk --no-cache add curl
EXPOSE 8087

WORKDIR /opt
CMD ["./socketsrv"]
