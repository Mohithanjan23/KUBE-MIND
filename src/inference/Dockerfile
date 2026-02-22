# Use Go 1.21 as build image
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /inference_server ./src/inference/main.go

# Minimal output image
FROM alpine:3.18
COPY --from=builder /inference_server /inference_server
EXPOSE 9000 9090
ENTRYPOINT ["/inference_server"]
