# Development Dockerfile with hot reload
FROM docker.1ms.run/library/node:20-alpine

WORKDIR /app

# Install dependencies for development
RUN apk add --no-cache libc6-compat

# Enable corepack for pnpm
RUN corepack enable

# Expose port
EXPOSE 3000

# Install dependencies and start dev server
# This ensures native modules are compiled for Linux
# Use -H 0.0.0.0 to bind to all interfaces for Docker access
# Use --webpack instead of Turbopack (Turbopack has issues in Docker)
# See: https://github.com/vercel/next.js/discussions/77102
CMD ["sh", "-c", "pnpm install && pnpm exec next dev -H 0.0.0.0 --webpack"]
