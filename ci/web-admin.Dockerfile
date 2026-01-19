# Dependencies stage
ARG REGISTRY=registry.corp.agentsmesh.ai
FROM ${REGISTRY}/library/node:20-alpine AS deps

WORKDIR /app

# Copy package files
COPY package.json pnpm-lock.yaml ./

# Install dependencies
RUN corepack enable pnpm && pnpm i --frozen-lockfile

# Build stage
ARG REGISTRY=registry.corp.agentsmesh.ai
FROM ${REGISTRY}/library/node:20-alpine AS builder

WORKDIR /app

# Copy dependencies
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Build-time environment variables for Next.js
ARG NEXT_PUBLIC_API_URL=https://api.agentsmesh.cn
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}

# Set environment variables for build
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production

# Build the application
RUN corepack enable pnpm && pnpm build

# Production stage
ARG REGISTRY=registry.corp.agentsmesh.ai
FROM ${REGISTRY}/library/node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Create non-root user
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy built application
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

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

# Run the application
CMD ["node", "server.js"]
