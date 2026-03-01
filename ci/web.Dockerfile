# REGISTRY_PREFIX: Optional registry mirror prefix (e.g., "myregistry.example.com/library/")
#                  Leave empty for Docker Hub (default)
ARG REGISTRY_PREFIX=

# Dependencies stage
FROM ${REGISTRY_PREFIX}node:20-alpine AS deps

WORKDIR /app

# Copy package files
COPY package.json package-lock.json* yarn.lock* pnpm-lock.yaml* ./

# Install dependencies
RUN \
    if [ -f yarn.lock ]; then yarn --frozen-lockfile; \
    elif [ -f package-lock.json ]; then npm ci; \
    elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm i --frozen-lockfile; \
    else npm i; \
    fi

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
# IMPORTANT: Must use NEXT_PUBLIC_ prefix for Next.js to inline them in client code
ENV NEXT_PUBLIC_PRIMARY_DOMAIN=__PRIMARY_DOMAIN__
ENV NEXT_PUBLIC_USE_HTTPS=__USE_HTTPS__

# Set environment variables for build
ENV NEXT_TELEMETRY_DISABLED 1
ENV NODE_ENV production

# Build the application
RUN \
    if [ -f yarn.lock ]; then yarn build; \
    elif [ -f package-lock.json ]; then npm run build; \
    elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm build; \
    else npm run build; \
    fi

# Production stage
ARG REGISTRY_PREFIX=
FROM ${REGISTRY_PREFIX}node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV production
ENV NEXT_TELEMETRY_DISABLED 1

# Create non-root user
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy public files
COPY --from=builder /app/public ./public

# Copy built application
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

# Copy entrypoint script
COPY --chown=nextjs:nodejs docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# Switch to non-root user
USER nextjs

# Expose port
EXPOSE 3000

ENV PORT 3000
ENV HOSTNAME "0.0.0.0"

# Install curl for health check
USER root
RUN apk add --no-cache curl
USER nextjs

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:3000/ || exit 1

# Run the application with entrypoint for runtime env injection
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["node", "server.js"]
