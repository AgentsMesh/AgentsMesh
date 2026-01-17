#!/bin/sh
# Web entrypoint script for development

set -e

cd /app

# Check if node_modules exists and is valid
if [ ! -d "node_modules" ] || [ ! -f "node_modules/.pnpm-integrity" ]; then
    echo "Installing dependencies..."
    pnpm install
else
    echo "Dependencies already installed, skipping pnpm install"
fi

# Use exec to replace this shell with the Next.js process
echo "Starting Next.js in development mode..."
exec pnpm dev
