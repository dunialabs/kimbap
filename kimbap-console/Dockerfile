# PETA Console Docker Image
FROM node:20-alpine AS base

# Set working directory
WORKDIR /app

# Install required system dependencies
RUN apk add --no-cache \
    postgresql-client \
    curl \
    bash

# Build stage
FROM base AS builder

# Copy package.json and lock files
COPY package*.json ./
COPY prisma ./prisma/

# Configure npm for better network reliability
RUN npm config set fetch-retries 5 && \
    npm config set fetch-retry-mintimeout 20000 && \
    npm config set fetch-retry-maxtimeout 120000 && \
    npm config set fetch-timeout 300000

# Install all dependencies with retry mechanism
RUN npm ci --prefer-offline --no-audit || \
    (echo "First npm ci attempt failed, retrying..." && \
     sleep 5 && \
     npm ci --prefer-offline --no-audit) || \
    (echo "Second npm ci attempt failed, retrying one more time..." && \
     sleep 10 && \
     npm ci --prefer-offline --no-audit) && \
    npm run db:generate

# Copy source code
COPY . .

# Build frontend (standalone mode)
RUN npm run build

# Production stage
FROM node:20-alpine AS production

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache \
    postgresql-client \
    curl \
    bash \
    dumb-init

# Create non-root user (before copying files)
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nextjs -u 1001

# Create data directory for persistent storage
RUN mkdir -p /app/data && \
    chown -R nextjs:nodejs /app/data && \
    chmod -R 755 /app/data

# Copy build output
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nodejs /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/prisma ./prisma
COPY --from=builder --chown=nextjs:nodejs /app/scripts/unified-db-init.js ./scripts/unified-db-init.js
COPY --from=builder --chown=nextjs:nodejs /app/jobs ./jobs
COPY --from=builder --chown=nextjs:nodejs /app/data/tool-templates.json /app/defaults/tool-templates.json

# Install production dependencies (avoid copying full node_modules)
COPY --chown=nextjs:nodejs package*.json ./
RUN npm ci --only=production --ignore-scripts && \
    npm cache clean --force

# Generate Prisma client and ensure permissions are correct
RUN npx prisma generate --schema=./prisma/schema.prisma && \
    chown -R nextjs:nodejs /app/node_modules/.prisma && \
    chmod -R 755 /app/node_modules/.prisma

# Copy Docker entrypoint script and set permissions
COPY --chown=nextjs:nodejs docker/docker-entrypoint.sh /app/docker/docker-entrypoint.sh
RUN chmod +x /app/docker/docker-entrypoint.sh

# Switch to non-root user
USER nextjs

# Environment variables
ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
ENV PORT=3000
ENV BACKEND_PORT=3002

# Expose ports
EXPOSE 3000 3002

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:3000 || exit 1

ENTRYPOINT ["/usr/bin/dumb-init", "--", "/app/docker/docker-entrypoint.sh"]
