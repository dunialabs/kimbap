#!/usr/bin/env node

/**
 * Log sync scheduled task
 * Fetch logs from proxy and save to local database
 * Runs independently, does not depend on Next.js application
 */

const { PrismaClient } = require('@prisma/client');
const http = require('http');
const https = require('https');

const prisma = new PrismaClient();

// Sync state tracking
let isSyncInProgress = false;
let syncStartTime = null;

// Configuration
const CONFIG = {
  SYNC_INTERVAL_MINUTES: parseInt(process.env.LOG_SYNC_INTERVAL_MINUTES) || 2,  // 2 minute interval
  MAX_LOGS_PER_REQUEST: parseInt(process.env.MAX_LOGS_PER_REQUEST) || 5000,     // max fetch size: 5000 logs
  BATCH_SIZE: parseInt(process.env.LOG_BATCH_SIZE) || 500,                      // save 500 logs per batch
  ENABLED: process.env.LOG_SYNC_ENABLED !== 'false',
  REQUEST_TIMEOUT: parseInt(process.env.LOG_SYNC_TIMEOUT) || 180000,            // 3 minute timeout
  RETRY_ATTEMPTS: parseInt(process.env.LOG_SYNC_RETRY_ATTEMPTS) || 2           // retry twice
};

/**
 * Get Kimbap Core URL from config table
 */
async function getKimbapCoreUrl() {
  try {
    // Get configuration from database
    const config = await prisma.config.findFirst();

    if (!config || !config.kimbap_core_host) {
      console.log('[LogSync] No Kimbap Core configuration found in database, using default localhost');
      return 'http://localhost:3002';
    }

    let url = config.kimbap_core_host;
    const port = typeof config.kimbap_core_port === 'number' ? config.kimbap_core_port : undefined;

    // Build URL with smart protocol detection (same logic as proxy-api.ts)
    if (url.startsWith('http://') || url.startsWith('https://')) {
      // Host already contains protocol
      const urlObj = new URL(url);
      const isHttps = urlObj.protocol === 'https:';
      const defaultPort = isHttps ? 443 : 80;

      // Add port if it's not the default for the protocol
      if (port && port !== defaultPort) {
        url = `${url}:${port}`;
      }
    } else {
      // Host doesn't contain protocol, add it based on type
      // Treat localhost and host.docker.internal as HTTP addresses
      const isIP = /^(\d{1,3}\.){3}\d{1,3}$/.test(url);
      const isLocalhost = url === 'localhost';
      const isHostDockerInternal = url === 'host.docker.internal';
      const protocol = (isIP || isLocalhost || isHostDockerInternal) ? 'http' : 'https';
      const defaultPort = protocol === 'https' ? 443 : 80;

      if (port && port !== defaultPort) {
        url = `${protocol}://${url}:${port}`;
      } else {
        url = `${protocol}://${url}`;
      }
    }

    return url;
  } catch (error) {
    console.error('[LogSync] Failed to get Kimbap Core config:', error.message);
    console.log('[LogSync] Falling back to default localhost');
    return 'http://localhost:3002';
  }
}

/**
 * Get proxyKey from proxy
 */
async function getProxyKey() {
  return new Promise((resolve) => {
    const stepStartTime = Date.now();

    (async () => {
      try {
      const PROXY_URL = await getKimbapCoreUrl();
      const url = new URL(PROXY_URL);

      const postData = JSON.stringify({
        action: 5001, // GET_PROXY
        data: {}
      });

      const isHttps = url.protocol === 'https:';
      const defaultPort = isHttps ? 443 : 80;
      const fullUrl = `${PROXY_URL}/admin`;


      const options = {
        hostname: url.hostname,
        port: url.port || defaultPort,
        path: '/admin',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Content-Length': Buffer.byteLength(postData)
        },
        timeout: CONFIG.REQUEST_TIMEOUT
      };

      const requestModule = isHttps ? https : http;
      const req = requestModule.request(options, (res) => {
        let data = '';

        res.on('data', (chunk) => {
          data += chunk;
        });

        res.on('end', () => {
          try {
            const elapsed = Date.now() - stepStartTime;
            const result = JSON.parse(data);

            if (!result.success) {
              console.error(`\x1b[33m[LogSync] ❌ Failed to get proxyKey (${elapsed}ms)\x1b[0m`);
              resolve(null);
              return;
            }

            resolve(result.data?.proxy?.proxyKey || null);
          } catch (parseError) {
            const elapsed = Date.now() - stepStartTime;
            console.error(`[LogSync] Failed to parse proxyKey response (${elapsed}ms):`, parseError.message);
            resolve(null);
          }
        });
      });

      req.on('error', (error) => {
        const elapsed = Date.now() - stepStartTime;
        console.error(`[LogSync] ProxyKey request failed (${elapsed}ms):`, error.message);
        resolve(null);
      });

      req.write(postData);
      req.end();

      } catch (error) {
        console.error('[LogSync] Failed to get proxyKey:', error.message);
        resolve(null);
      }
    })();
  });
}

/**
 * Get owner's access token
 */
async function getOwnerToken() {
  const envToken = process.env.LOG_SYNC_TOKEN?.trim();
  if (envToken) {
    return envToken;
  }
  console.error('\x1b[31m[LogSync] ❌ LOG_SYNC_TOKEN env var is not set. Log sync will not work until a valid owner access token is configured.\x1b[0m');
  console.error('\x1b[31m[LogSync] Set LOG_SYNC_TOKEN in your environment or .env file. See: https://docs.kimbap.sh/configuration#log-sync\x1b[0m');
  return null;
}

/**
 * Get the maximum idInCore value for this proxyKey, used for incremental sync
 */
async function getMaxIdInCore(proxyKey) {
  const stepStartTime = Date.now();

  if (!proxyKey) {
    console.log('[LogSync] ProxyKey is empty, starting full sync from beginning');
    return null;
  }

  try {
    const result = await prisma.log.findFirst({
      where: {
        proxyKey: proxyKey,
        idInCore: {
          not: null
        }
      },
      orderBy: {
        idInCore: 'desc'
      },
      select: {
        idInCore: true
      }
    });

    const elapsed = Date.now() - stepStartTime;

    if (result && result.idInCore) {
      const maxId = Number(result.idInCore);
      return maxId;
    } else {
      return null;
    }
  } catch (error) {
    const elapsed = Date.now() - stepStartTime;
    console.error(`[LogSync] Failed to query maximum idInCore (${elapsed}ms):`, error.message);
    return null;
  }
}

/**
 * Fetch logs from proxy with retry mechanism
 * Using Node.js native http module
 */
async function fetchLogsFromProxy(token, startId = null) {
  for (let attempt = 1; attempt <= CONFIG.RETRY_ATTEMPTS; attempt++) {
    const result = await fetchLogsFromProxyOnce(token, startId, attempt);
    if (result.length > 0 || attempt === CONFIG.RETRY_ATTEMPTS) {
      return result;
    }

    if (attempt < CONFIG.RETRY_ATTEMPTS) {
      const delayMs = attempt * 2000; // 2s, 4s delay between retries
      console.log(`\x1b[33m[LogSync] Waiting ${delayMs/1000}s before retry...\x1b[0m`);
      await new Promise(resolve => setTimeout(resolve, delayMs));
    }
  }

  return [];
}

/**
 * Single attempt to fetch logs from proxy
 */
async function fetchLogsFromProxyOnce(token, startId = null, attempt = 1) {
  return new Promise((resolve) => {
    const stepStartTime = Date.now();

    (async () => {
      try {
      const PROXY_URL = await getKimbapCoreUrl();
      const url = new URL(PROXY_URL);

      const requestData = {
        limit: CONFIG.MAX_LOGS_PER_REQUEST
      };

      // If startId exists, include it for incremental sync
      if (startId !== null) {
        requestData.id = startId;
      }

      const postData = JSON.stringify({
        action: 7002,
        data: requestData
      });

      const isHttps = url.protocol === 'https:';
      const defaultPort = isHttps ? 443 : 80;
      const fullUrl = `${PROXY_URL}/admin`;


      const options = {
        hostname: url.hostname,
        port: url.port || defaultPort,
        path: '/admin',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Content-Length': Buffer.byteLength(postData),
          ...(token ? { 'Authorization': `Bearer ${token}` } : {})
        },
        timeout: CONFIG.REQUEST_TIMEOUT
      };

      const requestModule = isHttps ? https : http;
      const req = requestModule.request(options, (res) => {
        let data = '';

        res.on('data', (chunk) => {
          data += chunk;
        });

        res.on('end', () => {
          try {
            const elapsed = Date.now() - stepStartTime;
            const result = JSON.parse(data);

            if (!result.success) {
              console.error(`\x1b[33m[LogSync] ❌ Server error (${elapsed}ms): ` + (result.error?.message || 'Unknown error') + '\x1b[0m');
              resolve([]);
              return;
            }

            const logs = result.data?.logs || [];
            console.log(`\x1b[36m[LogSync] API call: ${elapsed}ms | Returned: ${logs.length} logs\x1b[0m`);

            if (logs.length === 0 && attempt < CONFIG.RETRY_ATTEMPTS) {
              console.log(`\x1b[33m[LogSync] Retry ${attempt}/${CONFIG.RETRY_ATTEMPTS}\x1b[0m`);
            }

            resolve(logs);
          } catch (parseError) {
            const elapsed = Date.now() - stepStartTime;
            console.error(`[LogSync] Failed to parse response (${elapsed}ms):`, parseError.message);
            resolve([]);
          }
        });
      });

      req.on('error', (error) => {
        const elapsed = Date.now() - stepStartTime;
        console.error(`[LogSync] Request failed (${elapsed}ms):`, error.message);
        resolve([]);
      });

      req.write(postData);
      req.end();

      } catch (error) {
        console.error('[LogSync] Failed to fetch logs:', error.message);
        resolve([]);
      }
    })();
  });
}

/**
 * Save logs to local database with optimized batch processing
 */
async function saveLogsToDatabase(logs, currentProxyKey) {
  if (!logs || logs.length === 0) {
    return 0;
  }

  let savedCount = 0;

  // First, query existing records in bulk
  const logIds = logs.map(log => BigInt(log.id));
  const existingLogs = await prisma.log.findMany({
    where: {
      ...(currentProxyKey ? { proxyKey: currentProxyKey } : {}),
      idInCore: {
        in: logIds
      }
    },
    select: {
      idInCore: true
    }
  });

  // Build a Set of existing IDs for quick lookup
  const existingIdSet = new Set(existingLogs.map(log => log.idInCore.toString()));

  // Filter down to new records that still need insertion
  const logsToInsert = logs.filter(log => !existingIdSet.has(log.id.toString()));

  if (logsToInsert.length === 0) {
    return 0;
  }

  // Insert in batches using the configured batch size
  for (let i = 0; i < logsToInsert.length; i += CONFIG.BATCH_SIZE) {
    const batch = logsToInsert.slice(i, Math.min(i + CONFIG.BATCH_SIZE, logsToInsert.length));

    try {
      // Prepare batch insert data with simplified field mapping
      const dataToInsert = batch.map(log => ({
        idInCore: BigInt(log.id),
        addtime: BigInt(log.createdAt || log.addtime || Math.floor(Date.now() / 1000)),
        action: log.action || 0,
        userid: log.userid || log.userId || '',
        serverId: log.serverId || log.server_id || null,
        sessionId: log.sessionId || log.session_id || '',
        upstreamRequestId: log.upstreamRequestId || log.upstream_request_id || '',
        uniformRequestId: log.uniformRequestId || log.uniform_request_id || null,
        parentUniformRequestId: log.parentUniformRequestId || log.parent_uniform_request_id || null,
        proxyRequestId: log.proxyRequestId || log.proxy_request_id || null,
        ip: log.ip || '',
        ua: log.ua || '',
        tokenMask: log.tokenMask || log.token_mask || '',
        requestParams: log.requestParams || log.request_params || '',
        responseResult: log.responseResult || log.response_result || '',
        error: log.error || '',
        duration: log.duration ? parseInt(log.duration) : null,
        statusCode: log.statusCode || log.status_code ? parseInt(log.statusCode || log.status_code) : null,
        proxyKey: currentProxyKey
      }));

      // Insert the batch with createMany
      const result = await prisma.log.createMany({
        data: dataToInsert,
        skipDuplicates: true
      });

      savedCount += result.count;

    } catch (error) {
      const batchStart = i;
      const batchEnd = Math.min(i + CONFIG.BATCH_SIZE, logsToInsert.length);
      const firstId = batch[0]?.id ?? 'unknown';
      const lastId = batch[batch.length - 1]?.id ?? 'unknown';
      console.error(`\x1b[31m[LogSync] Batch insert failed (records ${batchStart}-${batchEnd}, idInCore ${firstId}-${lastId}): ${error.message}\x1b[0m`);
      console.error(`\x1b[33m[LogSync] ⚠️ Stopping sync after this batch failure so the next cycle re-fetches from the last contiguous saved log.\x1b[0m`);
      break;
    }
  }

  console.log(`\x1b[36m[LogSync] Inserted: ${savedCount} logs\x1b[0m`);
  return savedCount;
}

/**
 * Execute one log sync
 */
async function syncLogs() {
  // Skip if a sync is already running
  if (isSyncInProgress) {
    const elapsed = syncStartTime ? Math.floor((Date.now() - syncStartTime) / 1000) : 0;
    console.log(`\x1b[33m[LogSync] Sync in progress (${elapsed}s), skipping\x1b[0m`);
    return;
  }

  // Mark sync as running
  isSyncInProgress = true;
  syncStartTime = Date.now();

  console.log(`\x1b[33m[LogSync] Starting sync...\x1b[0m`);

  try {
    // 1. Get Kimbap Core URL and proxyKey
    const kimbapCoreUrl = await getKimbapCoreUrl();
    const proxyKey = await getProxyKey();
    if (!proxyKey) {
      console.log('\x1b[33m[LogSync] Kimbap Core service not initialized\x1b[0m');
      return;
    }

    // 2. Find owner's access token (retry every time, don't give up)
    const ownerToken = await getOwnerToken();
    if (!ownerToken) {
      console.log('\x1b[33m[LogSync] ❌ Failed to get owner token\x1b[0m');
      return;  // Return but will retry on next interval
    }

    // 3. Get maximum idInCore for incremental sync
    const maxIdInCore = await getMaxIdInCore(proxyKey);

    // 4. Fetch and save logs
    const logs = await fetchLogsFromProxy(ownerToken, maxIdInCore);
    const savedCount = await saveLogsToDatabase(logs, proxyKey);

    // 5. Display result
    const syncDuration = Math.floor((Date.now() - syncStartTime) / 1000);
    console.log(`\x1b[32m[LogSync] Completed: ${syncDuration}s\x1b[0m`);
  } catch (error) {
    const syncDuration = Math.floor((Date.now() - syncStartTime) / 1000);
    console.log(`\x1b[31m[LogSync] Failed: ${error.message} (${syncDuration}s)\x1b[0m`);
  } finally {
    // Reset sync state
    isSyncInProgress = false;
    syncStartTime = null;
  }
}

/**
 * Start scheduler
 */
function startScheduler() {
  if (!CONFIG.ENABLED) {
    console.log('\x1b[33m[LogSync] 🚫 Log sync disabled\x1b[0m');
    return;
  }

  console.log(`\x1b[33m[LogSync] 🚀 Started - Syncing every ${CONFIG.SYNC_INTERVAL_MINUTES} minutes (per request: ${CONFIG.MAX_LOGS_PER_REQUEST} logs, batch size: ${CONFIG.BATCH_SIZE} logs)\x1b[0m`);

  // Execute immediately once
  syncLogs();

  // Set timer
  const interval = CONFIG.SYNC_INTERVAL_MINUTES * 60 * 1000;
  const intervalId = setInterval(() => {
    syncLogs();
  }, interval);

  // Do NOT unref the main timer - keep it alive!
  // intervalId.unref() would allow the process to exit if this is the only timer

}

// Graceful shutdown
function gracefulShutdown() {
  console.log('[LogSync] Shutting down...');
  prisma.$disconnect();
  process.exit(0);
}

process.on('SIGINT', gracefulShutdown);
process.on('SIGTERM', gracefulShutdown);

// If running this script directly
if (require.main === module) {
  startScheduler();
}

module.exports = { syncLogs, startScheduler };
