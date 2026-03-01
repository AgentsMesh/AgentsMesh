# Development Dockerfile for Admin Console with hot reload
FROM node:20-alpine

WORKDIR /app

# Install dependencies for development
RUN apk add --no-cache libc6-compat

# Enable corepack for pnpm
RUN corepack enable

# Expose port (different from main web to avoid conflict)
EXPOSE 3001

# Install dependencies and start dev server
# Use -H 0.0.0.0 to bind to all interfaces for Docker access
CMD ["sh", "-c", "pnpm install && pnpm exec next dev -H 0.0.0.0 -p 3001"]
