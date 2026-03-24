#!/bin/sh

# Docker Entrypoint Script for KIMBAP Console
# This script ensures database migrations are applied before starting the application

echo "🚀 Starting KIMBAP Console Production..."

# Always refresh tool templates from the image defaults
if [ -f "/app/defaults/tool-templates.json" ]; then
    mkdir -p /app/data
    cp /app/defaults/tool-templates.json /app/data/tool-templates.json
    echo "✅ Tool templates refreshed from image defaults"
else
    echo "⚠️  Default tool templates not found, skipping refresh"
fi

# Configure DATABASE_URL environment variable
if [ -n "$DB_HOST" ] && [ -n "$DB_USER" ] && [ -n "$DB_PASSWORD" ] && [ -n "$DB_NAME" ]; then
    export DATABASE_URL="postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME"
    echo "✅ DATABASE_URL configured from environment variables"
elif [ -z "$DATABASE_URL" ]; then
    echo "⚠️  DATABASE_URL not set, skipping database check"
fi

# Check if the database is reachable
if [ -n "$DATABASE_URL" ]; then
    echo "🔍 Testing database connection..."
    if command -v pg_isready >/dev/null 2>&1; then
        if pg_isready -d "$DATABASE_URL" >/dev/null 2>&1; then
            echo "✅ Database connection successful"
        else
            echo "⚠️  Database not reachable or not configured"
            echo "   The application will start but may have limited functionality"
            echo "   Please ensure DATABASE_URL is set correctly"
        fi
    else
    echo "⚠️  pg_isready not available, skipping connection test"
    fi
else
    echo "⚠️  Database not reachable or not configured"
    echo "   The application will start but may have limited functionality"
    echo "   Please ensure DATABASE_URL is set correctly"
fi

# Step 1: Run database initialization (silent mode by default)
echo "📦 Initializing database..."

# In Docker, skip Prisma client generation since it's already done during build
if [ "$SKIP_PRISMA_GENERATE" = "true" ]; then
    echo "✅ Prisma Client already generated during Docker build"
    echo "🔄 Running database migrations only..."
    
    # Only run migrations, skip Prisma generation
    node -e "
    const { execSync } = require('child_process');
    
    console.log('🔄 Applying database migrations...');
    try {
        execSync('npx prisma migrate deploy --schema=./prisma/schema.prisma', { 
            stdio: 'inherit',
            cwd: '/app'
        });
        console.log('✅ Database migrations applied successfully');
    } catch (error) {
        console.log('ℹ️  No migrations to apply or database already up to date');
    }
    "
else
    # Normal initialization with Prisma generation
    if [ -f "/app/scripts/unified-db-init.js" ]; then
        node /app/scripts/unified-db-init.js
    else
        echo "⚠️  Database initialization script not found, skipping"
    fi
    
    # Check if initialization was successful
    if [ $? -ne 0 ]; then
        echo "❌ Database initialization failed"
        exit 1
    fi
fi

# Step 2: Start log sync job in background
if [ -f "/app/jobs/start.js" ]; then
    echo "🔄 Starting log sync job..."
    DATABASE_URL="$DATABASE_URL" node /app/jobs/start.js &
    echo "✅ Log sync job started in background"
else
    echo "⚠️  Log sync job script not found, skipping"
fi

# Step 3: Start the application based on environment
if [ "$NODE_ENV" = "production" ]; then
    # Note: proxy-server has been migrated to kimbap-core
    # Only start the Next.js frontend server

    echo "🌐 Starting frontend server on port 3000..."
    # Start frontend with environment variables
    DATABASE_URL="$DATABASE_URL" node /app/server.js

    echo "✅ Frontend service started:"
    echo "   Frontend: http://localhost:3000"
    echo "   Backend API (kimbap-core): configured via MCP_GATEWAY_URL"
else
    echo "🔧 Starting development server..."
    # In development, use the start script
    npm run start:docker
fi
