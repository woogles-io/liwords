FROM golang as build-env

RUN mkdir /opt/program
WORKDIR /opt/program

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

WORKDIR /opt/program/cmd/liwords-api

ARG BUILD_HASH=unknown
ARG BUILD_DATE=unknown

RUN go build -ldflags  "-X=main.BuildDate=${BUILD_DATE} -X=main.BuildHash=${BUILD_HASH}"

# Build minimal image:
FROM debian:latest
COPY --from=build-env /opt/program/cmd/liwords-api/liwords-api /opt/liwords-api
COPY --from=build-env /opt/program/db /opt/db
EXPOSE 8001

WORKDIR /opt
CMD ["./liwords-api"]

LABEL org.opencontainers.image.source https://github.com/woogles-io/liwords