# syntax=docker/dockerfile:1.4
FROM golang:1.25.1-alpine3.22 AS builder

WORKDIR /app

RUN apk --no-cache add git ca-certificates

COPY search-service/go.mod search-service/go.sum ./

COPY search-service/ ./
COPY pkg/ ./pkg/

RUN go mod edit -replace github.com/victoragudo/hotel-management-system/pkg=./pkg

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

RUN go mod tidy

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o search-api ./cmd/api

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/search-api .

EXPOSE 8080

CMD ["./search-api"]