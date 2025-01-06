FROM golang:1.23-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o rclone-manager ./cmd/main.go



FROM alpine:latest

RUN apk add --no-cache \
    fuse3 \
    curl \
    rclone

RUN mkdir -p /data

RUN ln -s /usr/bin/fusermount3 /usr/bin/fusermount

COPY --from=build /app/rclone-manager /usr/local/bin/rclone-manager

ENTRYPOINT ["/usr/local/bin/rclone-manager"]