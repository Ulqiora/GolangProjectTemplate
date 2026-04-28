FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY vendor ./vendor
COPY pkg ./pkg

RUN go build -mod=vendor -o /out/smart-span-demo ./pkg/smart-span/example/cmd/smart-span-demo

FROM alpine:3.22

RUN adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/smart-span-demo /app/smart-span-demo

USER appuser

EXPOSE 18080

ENTRYPOINT ["/app/smart-span-demo"]
