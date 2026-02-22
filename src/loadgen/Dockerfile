# Use Go 1.21 as build image
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /loadgen ./src/loadgen/main.go

# Minimal output image
FROM alpine:3.18
COPY --from=builder /loadgen /loadgen
ENTRYPOINT ["/loadgen"]
