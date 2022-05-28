FROM golang:alpine as build-env

RUN mkdir /opt/program
WORKDIR /opt/program

RUN apk update
RUN apk add build-base ca-certificates git

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

WORKDIR /opt/program/cmd/liwords-api

ARG BUILD_HASH=unknown
ARG BUILD_DATE=unknown

RUN go build -ldflags  "-X=main.BuildDate=${BUILD_DATE} -X=main.BuildHash=${BUILD_HASH}"

# Build minimal image:
FROM alpine
COPY --from=build-env /opt/program/cmd/liwords-api/liwords-api /opt/liwords-api
COPY --from=build-env /opt/program/db /opt/db
RUN apk --no-cache add curl
EXPOSE 8001

WORKDIR /opt
CMD ["./liwords-api"]

LABEL org.opencontainers.image.source https://github.com/domino14/liwords