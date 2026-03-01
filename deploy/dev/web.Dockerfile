# Development Dockerfile with hot reload
FROM node:20-alpine

WORKDIR /app

# Install dependencies for development
RUN apk add --no-cache libc6-compat

# Enable corepack for pnpm
RUN corepack enable

# Expose port
EXPOSE 3000

# Use entrypoint script (mounted via docker-compose) for smart dependency management
# This prevents unnecessary pnpm install runs
# Use --webpack instead of Turbopack (Turbopack has issues in Docker)
# See: https://github.com/vercel/next.js/discussions/77102
CMD ["sh", "/usr/local/bin/web-entrypoint.sh"]
