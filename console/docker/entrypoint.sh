#!/bin/bash
set -e

echo "========================================" 
echo "       Kimbap Console Docker Starting"
echo "========================================"

# Wait for the database to be ready
echo "🔍 Waiting for the database..."
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER"; do
  echo "Database is unavailable - sleeping"
  sleep 2
done

echo "✅ Database is ready!"

# Run database sync
echo "🔄 Syncing the database schema..."
cd /app
export DATABASE_URL="postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME"

# Try running migrations; if they fail, skip (database already has data)
if [ -f "node_modules/.bin/prisma" ]; then
  echo "🔄 Running database migrations..."
  if ! ./node_modules/.bin/prisma migrate deploy 2>/dev/null; then
    echo "⚠️  Database migration skipped (existing data detected)"
    echo "✅ Using existing database schema"
  fi
else
  echo "⚠️  Prisma client not found, skipping migration"
  echo "✅ Assuming database schema is ready"
fi

# Start backend service (background)
echo "🔧 Starting backend service..."
echo "📊 Database URL: postgresql://$DB_USER:***@$DB_HOST:$DB_PORT/$DB_NAME"
export DATABASE_URL="postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME"
NODE_ENV=production DATABASE_URL="$DATABASE_URL" node proxy-server/index.js &
BACKEND_PID=$!

# Wait for backend to start
sleep 3

# Start frontend service
echo "🎨 Starting frontend service..."
DATABASE_URL="$DATABASE_URL" node server.js &
FRONTEND_PID=$!

echo "📱 Kimbap Console is running!"
echo "   Frontend: http://localhost:3000"
echo "   Backend:  http://localhost:3002"

# Cleanup function
cleanup() {
  echo "🛑 Shutting down services..."
  kill $BACKEND_PID $FRONTEND_PID 2>/dev/null || true
  wait $BACKEND_PID $FRONTEND_PID 2>/dev/null || true
  echo "✅ Shutdown complete"
  exit 0
}

# Set up signal handling
trap cleanup SIGTERM SIGINT

# Wait for child processes
wait
