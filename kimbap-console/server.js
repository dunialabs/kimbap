#!/usr/bin/env node

/**
 * Custom HTTPS/HTTP server for Next.js
 * Supports HTTPS with fallback to HTTP if HTTPS fails
 * Only one server runs at a time
 */

const { createServer } = require('http');
const { createServer: createHttpsServer } = require('https');
const { parse } = require('url');
const next = require('next');
const fs = require('fs');
const path = require('path');

const dev = process.env.NODE_ENV !== 'production';
const hostname = 'localhost';
const port = parseInt(process.env.PORT || process.env.FRONTEND_PORT || '3000', 10);
const httpsPort = parseInt(process.env.FRONTEND_HTTPS_PORT || String(port), 10);

// Create Next.js app instance
const app = next({ dev, hostname, port });
const handle = app.getRequestHandler();

// Global variable to store the active server
let activeServer = null;

// Flag to prevent multiple shutdown attempts
let isShuttingDown = false;

/**
 * Graceful shutdown handler
 */
function gracefulShutdown(signal) {
  // Prevent multiple simultaneous shutdown attempts
  if (isShuttingDown) {
    return;
  }
  isShuttingDown = true;

  console.log(`\n${signal} received, shutting down gracefully...`);

  // Close the active server if it exists
  if (activeServer) {
    activeServer.close((err) => {
      if (err) {
        console.error('Error closing server:', err);
      } else {
        console.log('Server closed successfully');
      }

      // Next.js app doesn't need explicit closing
      // The server closure will handle cleanup
      process.exit(0);
    });

    // Set a timeout to force exit if graceful shutdown takes too long
    setTimeout(() => {
      console.error('Forced shutdown after 10 seconds timeout');
      process.exit(1);
    }, 10000);
  } else {
    // No server to close, exit immediately
    process.exit(0);
  }
}

/**
 * Start HTTPS server
 * @returns {Object|null} Server instance if successful, null otherwise
 */
async function startHttpsServer() {
  try {
    // Get certificate paths from environment
    const certPath = process.env.SSL_CERT_PATH;
    const keyPath = process.env.SSL_KEY_PATH;

    // Check if certificate files exist
    if (!certPath || !keyPath || !fs.existsSync(certPath) || !fs.existsSync(keyPath)) {
      console.warn('⚠️  SSL certificates not found or not configured (set SSL_CERT_PATH and SSL_KEY_PATH)');
      return null;
    }

    // Read SSL certificates
    const httpsOptions = {
      key: fs.readFileSync(keyPath),
      cert: fs.readFileSync(certPath)
    };

    // Create HTTPS server
    const server = createHttpsServer(httpsOptions, async (req, res) => {
      try {
        const parsedUrl = parse(req.url, true);
        await handle(req, res, parsedUrl);
      } catch (err) {
        console.error('Error occurred handling', req.url, err);
        res.statusCode = 500;
        res.end('Internal server error');
      }
    });

    // Start listening
    await new Promise((resolve, reject) => {
      server.listen(httpsPort, (err) => {
        if (err) {
          reject(err);
        } else {
          console.log(`✅ Next.js HTTPS server ready on https://${hostname}:${httpsPort}`);
          resolve();
        }
      });
    });

    return server;
  } catch (error) {
    console.error('❌ Failed to start HTTPS server:', error.message);
    return null;
  }
}

/**
 * Start HTTP server
 * @returns {Object|null} Server instance if successful, null otherwise
 */
async function startHttpServer() {
  try {
    const server = createServer(async (req, res) => {
      try {
        const parsedUrl = parse(req.url, true);
        await handle(req, res, parsedUrl);
      } catch (err) {
        console.error('Error occurred handling', req.url, err);
        res.statusCode = 500;
        res.end('Internal server error');
      }
    });

    // Start listening
    await new Promise((resolve, reject) => {
      server.listen(port, (err) => {
        if (err) {
          reject(err);
        } else {
          console.log(`✅ Next.js HTTP server ready on http://${hostname}:${port}`);
          resolve();
        }
      });
    });

    return server;
  } catch (error) {
    console.error('❌ Failed to start HTTP server:', error.message);
    return null;
  }
}

/**
 * Start scheduled tasks
 */
function startJobs() {
  try {
    const { startScheduler } = require('./jobs/log-sync.js');
    startScheduler();
  } catch (error) {
    console.error('[Server] Failed to start scheduled tasks:', error);
  }
}

/**
 * Main startup function
 */
async function startServer() {
  try {
    // Initialize database (apply migrations if needed)
    const { initializeDatabase } = require('./lib/database-init');
    await initializeDatabase();
    
    // Prepare Next.js app
    await app.prepare();
    console.log('🚀 Starting Next.js server...');

    const enableHttps = process.env.ENABLE_HTTPS === 'true';

    // Try HTTPS first if enabled
    if (enableHttps) {
      console.log('🔒 HTTPS is enabled, attempting to start HTTPS server...');
      activeServer = await startHttpsServer();

      if (activeServer) {
        console.log('🎉 Frontend server is running in HTTPS mode');
        // Start scheduled tasks
        startJobs();
        return;
      }

      console.warn('⚠️  HTTPS server failed to start, falling back to HTTP...');
    }

    // Start HTTP server (either as fallback or primary)
    activeServer = await startHttpServer();

    if (!activeServer) {
      console.error('❌ Failed to start any server (neither HTTPS nor HTTP)');
      process.exit(1);
    }

    console.log('🎉 Frontend server is running in HTTP mode');

    // Start scheduled tasks
    startJobs();

  } catch (error) {
    console.error('❌ Server startup failed:', error);
    process.exit(1);
  }
}

// Register signal handlers (only once, at the top level)
process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
process.on('SIGINT', () => gracefulShutdown('SIGINT'));

// Handle uncaught exceptions
process.on('uncaughtException', (error) => {
  console.error('❌ Uncaught Exception:', error);
  gracefulShutdown('UNCAUGHT_EXCEPTION');
});

// Handle unhandled promise rejections
process.on('unhandledRejection', (reason, promise) => {
  console.error('❌ Unhandled Rejection at:', promise, 'reason:', reason);
  gracefulShutdown('UNHANDLED_REJECTION');
});

// Start the server
startServer();
