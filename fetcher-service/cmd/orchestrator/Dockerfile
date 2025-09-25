# syntax=docker/dockerfile:1.4
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

RUN apk --no-cache add git ca-certificates

COPY fetcher-service/go.mod fetcher-service/go.sum ./
COPY pkg/go.mod /pkg/go.mod

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY fetcher-service/ ./
COPY pkg/ /pkg/
COPY fetcher-service/pkg/ ./pkg/

RUN go mod tidy

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o orchestrator ./cmd/orchestrator

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/orchestrator .

EXPOSE 50051

CMD ["./orchestrator"]