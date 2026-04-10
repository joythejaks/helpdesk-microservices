FROM golang:1.24-alpine3.19

WORKDIR /app

COPY go.mod ./
RUN go mod tidy

COPY . .

RUN go build -o main ./cmd/main.go

CMD ["./main"]