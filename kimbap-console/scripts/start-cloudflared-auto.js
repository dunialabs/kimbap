#!/usr/bin/env node

/**
 * Start cloudflared automatically based on database configuration
 * This script checks for existing tunnel configuration and starts cloudflared if available
 */

const { PrismaClient } = require('@prisma/client');
const { exec } = require('child_process');
const { promisify } = require('util');
const path = require('path');
const fs = require('fs').promises;
const axios = require('axios');
const crypto = require('crypto');
const os = require('os');
const { generateCloudflaredConfig } = require('../settings/cloudflared-config-template');

const execAsync = promisify(exec);
const prisma = new PrismaClient();

// Configuration from lib/config.ts
const CLOUD_API_BASE_URL = process.env.KIMBAP_CLOUD_API_URL || 'https://kimbap-cloud.kimbap.io';
const CLOUD_API_ENDPOINTS = {
  tunnelCreate: '/tunnel/create',
  tunnelDelete: '/tunnel/delete'
};

// ANSI color codes
const colors = {
  orange: '\x1b[38;5;208m',  // Orange color
  reset: '\x1b[0m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m'
};

// Helper function to print in orange
function logOrange(message) {
  console.log(`${colors.orange}${message}${colors.reset}`);
}

/**
 * Generate unique appId
 */
function generateAppId() {
  const hostname = os.hostname().replace(/[^a-zA-Z0-9]/g, '').substring(0, 4);
  const timestamp = Date.now().toString(36).substring(0, 4);
  const random = crypto.randomBytes(2).toString('hex');
  return `${hostname}${timestamp}${random}`.substring(0, 10).toLowerCase();
}

/**
 * Create new tunnel via API
 */
async function createTunnel(appId) {
  try {
    const response = await axios.post(
      `${CLOUD_API_BASE_URL}${CLOUD_API_ENDPOINTS.tunnelCreate}`,
      { appId },
      {
        headers: { 'Content-Type': 'application/json' },
        timeout: 15000
      }
    );
    return response.data;
  } catch (error) {
    if (error.response) {
      console.error('API Error Response:', error.response.status, error.response.data);
    } else if (error.request) {
      console.error('No response from API:', error.message);
    } else {
      console.error('Request setup error:', error.message);
    }
    throw new Error(error.response?.data?.error || error.message || 'Failed to create tunnel');
  }
}

/**
 * Delete tunnel via API
 */
async function deleteTunnel(tunnelId) {
  try {
    const response = await axios.post(
      `${CLOUD_API_BASE_URL}${CLOUD_API_ENDPOINTS.tunnelDelete}`,
      { tunnelId },
      {
        headers: { 'Content-Type': 'application/json' },
        timeout: 15000
      }
    );
    return response.data;
  } catch (error) {
    // Tunnel might already be deleted, continue anyway
    console.error('Failed to delete tunnel:', error.message);
  }
}

/**
 * Ensure cloudflared directory exists
 */
async function ensureConfigDir() {
  const configDir = path.join(process.cwd(), 'cloudflared');
  try {
    await fs.mkdir(configDir, { recursive: true });
    return configDir;
  } catch (error) {
    console.error('Failed to create cloudflared config directory:', error);
    throw error;
  }
}

/**
 * Write credentials file
 */
async function writeCredentials(tunnelId, credentials) {
  const configDir = await ensureConfigDir();
  const credentialsFile = path.join(configDir, `${tunnelId}.json`);
  
  try {
    await fs.writeFile(credentialsFile, JSON.stringify(credentials, null, 2), 'utf8');
    // Credentials written
    return credentialsFile;
  } catch (error) {
    console.error('Failed to write credentials:', error);
    throw error;
  }
}

/**
 * Write config file
 */
async function writeConfig(configYaml) {
  const configDir = await ensureConfigDir();
  const configFile = path.join(configDir, 'config.yml');
  
  try {
    await fs.writeFile(configFile, configYaml, 'utf8');
    // Config written
    return configFile;
  } catch (error) {
    console.error('Failed to write config:', error);
    throw error;
  }
}



/**
 * Main function to start cloudflared
 */
async function startCloudflared() {
  try {
    logOrange('🔍 Checking for cloudflared configuration...');
    
    // Query the first dns_conf record with type=1
    const dnsRecord = await prisma.dnsConf.findFirst({
      where: { type: 1 },
      orderBy: { id: 'asc' }
    });
    
    if (!dnsRecord) {
      logOrange('ℹ️  No type=1 tunnel configuration found. Skipping cloudflared startup.');
      return false;
    }
    
    logOrange(`📡 Found tunnel configuration`);
    
    const configDir = path.join(process.cwd(), 'cloudflared');
    const credentialsFile = path.join(configDir, `${dnsRecord.tunnelId}.json`);
    
    // Check if we have local credentials file
    let hasValidCredentials = false;
    let credentials = null;
    
    try {
      await fs.access(credentialsFile);
      const credentialsContent = await fs.readFile(credentialsFile, 'utf8');
      credentials = JSON.parse(credentialsContent);
      
      // Check if credentials have TunnelSecret (required for authentication)
      if (credentials.TunnelSecret) {
        hasValidCredentials = true;
        logOrange('📂 Found valid local credentials');
      }
    } catch (error) {
      // No local credentials file
    }
    
    let tunnelId = dnsRecord.tunnelId;
    let subdomain = dnsRecord.subdomain;
    
    if (hasValidCredentials) {
      // Use existing credentials and start cloudflared
      logOrange('✅ Using existing tunnel credentials');
    } else {
      // No valid local credentials, need to recreate tunnel
      logOrange('⚠️  No valid local credentials found');
      logOrange('🔄 Recreating tunnel...');
      
      // Delete old tunnel if it exists
      if (tunnelId) {
        try {
          await deleteTunnel(tunnelId);
          logOrange('🗑️  Deleted old tunnel');
        } catch (error) {
          // Continue even if delete fails
        }
      }
      
      // Create new tunnel
      const appId = generateAppId();
      logOrange(`🔑 Creating new tunnel with appId: ${appId}`);
      
      try {
        const tunnelResponse = await createTunnel(appId);
        
        tunnelId = tunnelResponse.tunnelId;
        subdomain = tunnelResponse.subdomain;
        credentials = tunnelResponse.credentials;
        
        logOrange(`✅ New tunnel created: ${subdomain}`);
        
        // Update database with new tunnel info
        const now = Math.floor(Date.now() / 1000);
        await prisma.dnsConf.update({
          where: { id: dnsRecord.id },
          data: {
            tunnelId: tunnelId,
            subdomain: subdomain,
            updateTime: now
          }
        });
        
        // Write credentials
        await writeCredentials(tunnelId, credentials);
        await writeCredentials('credentials', credentials);
        
      } catch (error) {
        console.error('❌ Failed to create new tunnel:', error.message);
        return false;
      }
    }
    
    // Generate and write config using common template
    const configYml = generateCloudflaredConfig({
      tunnelId,
      subdomain
    });
    
    await writeConfig(configYml);
    
    // Show the public URL
    logOrange(`🌍 Tunnel configured for: https://${subdomain}`);
    logOrange(`ℹ️  Note: cloudflared container needs to be started manually`);
    
    return true;
    
  } catch (error) {
    console.error('❌ Error starting cloudflared:', error);
    return false;
  } finally {
    await prisma.$disconnect();
  }
}

// Export for use in other scripts
module.exports = { startCloudflared };

// Run if called directly
if (require.main === module) {
  startCloudflared()
    .then(success => {
      process.exit(success ? 0 : 1);
    })
    .catch(error => {
      console.error('Fatal error:', error);
      process.exit(1);
    });
}