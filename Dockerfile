# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bot ./cmd/bot

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /bot /app/bot
COPY configs/config.yaml /app/configs/config.yaml

ENTRYPOINT ["/app/bot"]
CMD ["-config", "/app/configs/config.yaml"]
