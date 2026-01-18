# Development Dockerfile with hot reload using Air
FROM docker.1ms.run/library/golang:1.25-alpine

# Install air for hot reload
RUN go install github.com/air-verse/air@latest

# Install golang-migrate for database migrations
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Install dependencies for debugging
RUN apk add --no-cache git ca-certificates tzdata

# Copy proto module first (required by backend go.mod replace directive)
WORKDIR /proto
COPY proto/go.mod proto/go.sum ./
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
