FROM golang:1.24.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

COPY --from=builder /app/.env* ./
COPY --from=builder /app/configs ./configs

# Скопировать статичный фронтенд

EXPOSE 8080

CMD ["./main"] 