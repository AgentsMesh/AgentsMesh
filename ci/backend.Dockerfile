# Build stage
# Build context should be project root (not backend/)
# REGISTRY_PREFIX: Use internal mirror for GitLab CI (e.g., registry.corp.agentsmesh.ai/library/)
#                  Leave empty for Docker Hub (GitHub Actions)
ARG REGISTRY_PREFIX=
ARG GO_VERSION=1.25
FROM ${REGISTRY_PREFIX}golang:${GO_VERSION} AS builder

WORKDIR /app

# Go module proxy (override via --build-arg for faster downloads in specific regions)
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

# Copy proto module first (required by replace directive in go.mod)
COPY proto /proto

# Copy agentfile module (required by replace directive in go.mod)
COPY agentfile/go.mod agentfile/go.sum /agentfile/

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source code
COPY agentfile/ /agentfile/
COPY backend/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

# Final stage
ARG REGISTRY_PREFIX=
FROM ${REGISTRY_PREFIX}alpine:3.19

WORKDIR /app

# Install ca-certificates, tzdata, and golang-migrate
RUN apk --no-cache add ca-certificates tzdata curl git && \
    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz && \
    mv migrate /usr/local/bin/migrate && \
    chmod +x /usr/local/bin/migrate

# Create non-root user
RUN addgroup -g 1000 -S app && \
    adduser -u 1000 -S app -G app

# Copy binary and migrations from builder
COPY --from=builder /app/server /app/server
COPY --from=builder /app/migrations /app/migrations

# Download GeoIP database (DB-IP City Lite, free, CC BY 4.0, no account needed)
# Enables geo-aware relay selection. ~20MB, rebuilt monthly by DB-IP.
# Try current month first, fall back to previous month if not yet available.
RUN mkdir -p /app/data && \
    CURRENT=$(date +%Y-%m) && \
    MONTH=$(date +%m) && YEAR=$(date +%Y) && \
    if [ "$MONTH" = "01" ]; then PREV="$((YEAR-1))-12"; else PREV=$(printf "%d-%02d" "$YEAR" "$((10#$MONTH-1))"); fi && \
    (curl -sSfL "https://download.db-ip.com/free/dbip-city-lite-${CURRENT}.mmdb.gz" | gunzip > /app/data/geoip.mmdb) || \
    (curl -sSfL "https://download.db-ip.com/free/dbip-city-lite-${PREV}.mmdb.gz" | gunzip > /app/data/geoip.mmdb) || \
    echo "GeoIP download failed (non-fatal), geo-aware relay selection disabled"

# Create data directory for ACME storage and set ownership
RUN mkdir -p /data/acme && \
    chown -R app:app /app /data

# Switch to non-root user
USER app

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the server
# To run migrations manually:
#   migrate -path /app/migrations -database "postgres://user:password@host:5432/dbname?sslmode=disable" up
#   migrate -path /app/migrations -database "postgres://user:password@host:5432/dbname?sslmode=disable" down 1
#   migrate -path /app/migrations -database "postgres://user:password@host:5432/dbname?sslmode=disable" version
ENTRYPOINT ["/app/server"]
