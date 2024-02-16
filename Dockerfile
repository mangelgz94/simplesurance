FROM golang:1.21-alpine as build
RUN apk add --no-cache tzdata ca-certificates

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

RUN mkdir -p /files_repository

COPY . .

RUN go build -o ./api -ldflags "-X main.BuildVersion=1" ./cmd

CMD ["./api"]