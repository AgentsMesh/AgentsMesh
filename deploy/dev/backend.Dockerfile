# Development Dockerfile with hot reload using Air
FROM docker.1ms.run/library/golang:1.25-alpine

# Install air for hot reload
RUN go install github.com/air-verse/air@latest

# Install golang-migrate for database migrations
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Install dependencies for debugging
RUN apk add --no-cache git ca-certificates tzdata curl

# Download GeoIP database (DB-IP City Lite, free, CC BY 4.0, no account needed)
# This enables geo-aware relay selection. ~20MB, rebuilt monthly.
# Stored in /opt/geoip/ (not /app/) since /app is mounted as volume in dev.
# Try current month first, fall back to previous month if not yet available.
RUN mkdir -p /opt/geoip && \
    CURRENT=$(date +%Y-%m) && \
    MONTH=$(date +%m) && YEAR=$(date +%Y) && \
    if [ "$MONTH" = "01" ]; then PREV="$((YEAR-1))-12"; else PREV=$(printf "%d-%02d" "$YEAR" "$((10#$MONTH-1))"); fi && \
    (curl -sSfL "https://download.db-ip.com/free/dbip-city-lite-${CURRENT}.mmdb.gz" | gunzip > /opt/geoip/geoip.mmdb) || \
    (curl -sSfL "https://download.db-ip.com/free/dbip-city-lite-${PREV}.mmdb.gz" | gunzip > /opt/geoip/geoip.mmdb) || \
    echo "GeoIP download failed (non-fatal), geo-aware relay disabled"

# Copy proto module first (required by backend go.mod replace directive)
WORKDIR /proto
COPY proto/go.mod proto/go.sum ./
RUN go mod download

# Copy agentfile module (required by backend go.mod replace directive)
WORKDIR /agentfile
COPY agentfile/go.mod agentfile/go.sum ./
RUN go mod download

# Copy backend module
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Source code will be mounted as volume

# Expose port
EXPOSE 8080

# Use air for hot reload
CMD ["air", "-c", ".air.toml"]
