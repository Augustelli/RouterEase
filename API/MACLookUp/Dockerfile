# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go mod tidy
RUN go build -o maclookup

# Final image
FROM alpine:latest
WORKDIR /app
# Download the OUI file during build
RUN apk add --no-cache curl && \
    curl -o ieee-oui.txt https://raw.githubusercontent.com/royhills/arp-scan/master/ieee-oui.txt
COPY --from=builder /app/maclookup .
EXPOSE 8080
ENTRYPOINT ["./maclookup", "server"]