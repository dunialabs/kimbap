#!/usr/bin/env node

/**
 * Start all scheduled tasks
 */

const { startScheduler } = require('./log-sync.js');

console.log('[Jobs] Starting scheduled task service...');

// Start log sync task
startScheduler();

console.log('[Jobs] All scheduled tasks started');