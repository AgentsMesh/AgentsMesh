# REGISTRY_PREFIX: Optional registry mirror prefix (e.g., "myregistry.example.com/library/")
#                  Leave empty for Docker Hub (default)
ARG REGISTRY_PREFIX=

# Dependencies stage
FROM ${REGISTRY_PREFIX}node:20-alpine AS deps

WORKDIR /app

# Copy package files
COPY package.json pnpm-lock.yaml ./

# NPM registry (override via --build-arg for faster downloads in specific regions)
ARG NPM_REGISTRY=
RUN corepack enable pnpm && \
    if [ -n "${NPM_REGISTRY}" ]; then pnpm config set registry ${NPM_REGISTRY}; fi && \
    pnpm i --frozen-lockfile

# Build stage
ARG REGISTRY_PREFIX=
FROM ${REGISTRY_PREFIX}node:20-alpine AS builder

WORKDIR /app

# Copy dependencies
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Build-time environment variables for Next.js
# Use placeholders that will be replaced at runtime by docker-entrypoint.sh
# This allows runtime configuration of PRIMARY_DOMAIN and USE_HTTPS
ENV PRIMARY_DOMAIN=__PRIMARY_DOMAIN__
ENV USE_HTTPS=__USE_HTTPS__

# Set environment variables for build
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production

# Build the application
ARG NPM_REGISTRY=
RUN corepack enable pnpm && \
    if [ -n "${NPM_REGISTRY}" ]; then pnpm config set registry ${NPM_REGISTRY}; fi && \
    pnpm build

# Production stage
ARG REGISTRY_PREFIX=
FROM ${REGISTRY_PREFIX}node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Create non-root user
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy built application
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

# Copy entrypoint script
COPY --chown=nextjs:nodejs docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# Switch to non-root user
USER nextjs

# Expose port (admin runs on 3001)
EXPOSE 3001

ENV PORT=3001
ENV HOSTNAME="0.0.0.0"

# Install curl for health check
USER root
RUN apk add --no-cache curl
USER nextjs

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:3001/ || exit 1

# Run the application with entrypoint for runtime env injection
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["node", "server.js"]
