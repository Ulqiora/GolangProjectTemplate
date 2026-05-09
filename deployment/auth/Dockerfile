FROM golang:1.25 AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .

RUN CGO_ENABLED=0 GOFLAGS=-mod=vendor go build -o /out/auth ./cmd/auth
RUN CGO_ENABLED=0 GOFLAGS=-mod=vendor go build -o /out/outbox ./cmd/outbox
RUN CGO_ENABLED=0 GOFLAGS=-mod=vendor go build -o /out/vk-consumer ./cmd/vk-consumer

FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /out/auth /app/auth
COPY --from=builder /out/outbox /app/outbox
COPY --from=builder /out/vk-consumer /app/vk-consumer
COPY configs /app/configs

ENTRYPOINT ["/app/auth"]
