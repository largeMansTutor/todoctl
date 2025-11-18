# syntax=docker/dockerfile:1.4

## Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
# Install git for fetching private modules if needed
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
# Build a statically linked binary for Linux. CGO is disabled.
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o todo ./cmd/todoctl

## Runtime stage
FROM gcr.io/distroless/static-debian11
WORKDIR /app
COPY --from=builder /app/todo /usr/local/bin/todo
COPY --from=builder /app/openapi /app/openapi
COPY --from=builder /app/migrations /app/migrations
ENV APP_PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/todo", "api"]
