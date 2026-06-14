# Stage 1: Build binary
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./

# Install git dan swag tool untuk generate dokumentasi
RUN apk add --no-cache git && \
    go install github.com/swaggo/swag/cmd/swag@latest && \
    go mod download

COPY . .
RUN /go/bin/swag init -g cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/main.go

# Stage 2: Runtime image
FROM alpine:3.21.3
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/main .

EXPOSE 8081
CMD ["./main"]