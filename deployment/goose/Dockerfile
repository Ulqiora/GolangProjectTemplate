# Билдер
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Финальный образ
FROM alpine:3.18
RUN apk add --no-cache postgresql-client bash
WORKDIR /app
COPY --from=builder /go/bin/goose /usr/local/bin/
COPY scripts/entrypoint.sh entrypoint.sh
COPY migrations migrations
RUN ls -la && sleep 15
RUN chmod +x entrypoint.sh
ENTRYPOINT ["./entrypoint.sh"]