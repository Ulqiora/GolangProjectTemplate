FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -o /out/producer-demo ./kafka/example/cmd/producer-demo

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
WORKDIR /home/app
COPY --from=builder /out/producer-demo /usr/local/bin/producer-demo

EXPOSE 2112
ENTRYPOINT ["/usr/local/bin/producer-demo"]
