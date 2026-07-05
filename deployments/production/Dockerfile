# ============================================================
# ShortURL: Production Multi-Stage Dockerfile
# ============================================================

# ── Stage 1: Build ──
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build binary with minimal size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/server ./cmd/url-shortener

# ── Stage 2: Run ──
FROM alpine:3.19

RUN apk add --no-cache ca-certificates wget

# Install migrate for database migrations
RUN wget -q -O /usr/local/bin/migrate.tar.gz \
      https://github.com/golang-migrate/migrate/releases/download/v4.19.0/migrate.linux-amd64.tar.gz && \
    tar -xzf /usr/local/bin/migrate.tar.gz -C /usr/local/bin migrate && \
    rm /usr/local/bin/migrate.tar.gz && \
    chmod +x /usr/local/bin/migrate

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Copy Casbin RBAC config
COPY --from=builder /app/internal/infra/rbac_model.conf ./internal/infra/rbac_model.conf
COPY --from=builder /app/internal/infra/rbac_policy.csv ./internal/infra/rbac_policy.csv

# Copy frontend files
COPY --from=builder /app/web ./web

EXPOSE 7500

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:7500/health || exit 1

CMD ["./server"]
