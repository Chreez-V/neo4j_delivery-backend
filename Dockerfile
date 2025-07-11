FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/main.go

FROM alpine:latest

WORKDIR /app


COPY --from=builder /app/main .
COPY --from=builder /app/scripts/ ./scripts/
RUN ls /
RUN ls /app/scripts

EXPOSE 8080

CMD ["./main"]
