FROM golang:1.27-alpine AS builder

WORKDIR /transcriber-app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o transcriber-bin ./cmd/main.go

FROM alpine:3.22
RUN apk add --no-cache postgresql-client curl

WORKDIR /app
COPY --from=builder /transcriber-app/transcriber-bin /app/transcriber-bin

ENTRYPOINT ["/app/transcriber-bin"]