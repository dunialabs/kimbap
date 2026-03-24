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

const execAsync = promisify(exec);
const prisma = new PrismaClient();

// We need to handle TypeScript modules in a Node.js context
async function loadServices() {
  // Since these are TypeScript modules, we need to handle them carefully
  // In production, these would be compiled to JavaScript
  try {
    // Try to load compiled versions first
    const { KimbapCloudApiService } = require('../dist/lib/KimbapCloudApiService');
    const { CloudflaredDockerService } = require('../dist/lib/CloudflaredDockerService');
    return { KimbapCloudApiService, CloudflaredDockerService };
  } catch (error) {
    // If compiled versions don't exist, use ts-node or tsx
    try {
      // Register TypeScript transpiler
      require('ts-node/register');
      const { KimbapCloudApiService } = require('../lib/KimbapCloudApiService');
      const { CloudflaredDockerService } = require('../lib/CloudflaredDockerService');
      return { KimbapCloudApiService, CloudflaredDockerService };
    } catch (tsError) {
      throw new Error('Failed to load TypeScript modules. Make sure ts-node is installed or modules are compiled.');
    }
  }
}

/**
 * Main function to start cloudflared
 */
async function startCloudflared() {
  try {
    console.log('🔍 Checking for cloudflared configuration...');
    
    // Load TypeScript services
    const { KimbapCloudApiService, CloudflaredDockerService } = await loadServices();
    const kimbapCloudApi = new KimbapCloudApiService();
    const cloudflaredService = new CloudflaredDockerService();
    
    // Query the first dns_conf record with tunnelId
    const dnsRecord = await prisma.dnsConf.findFirst({
      where: {
        tunnelId: {
          not: ''
        }
      },
      orderBy: {
        id: 'asc'
      }
    });
    
    if (!dnsRecord || !dnsRecord.tunnelId) {
      console.log('ℹ️  No tunnel configuration found in database. Skipping cloudflared startup.');
      return false;
    }
    
    console.log(`📡 Found tunnel configuration: ${dnsRecord.tunnelId}`);
    console.log(`🌐 Subdomain: ${dnsRecord.subdomain}`);
    
    // Get tunnel credentials from API
    console.log('🔑 Fetching tunnel credentials from API...');
    let tunnelInfo;
    try {
      tunnelInfo = await kimbapCloudApi.getTunnelCredentials(dnsRecord.tunnelId);
      console.log('✅ Successfully retrieved tunnel credentials');
    } catch (apiError) {
      console.error('❌ Failed to fetch tunnel credentials from API:', apiError.message);
      console.log('⚠️  Skipping cloudflared startup due to API error');
      return false;
    }
    
    // Ensure cloudflared directory exists
    await cloudflaredService.ensureConfigDir();
    
    // Write credentials file
    console.log('📝 Writing tunnel credentials...');
    const credentialsFile = await cloudflaredService.writeCredentials(
      dnsRecord.tunnelId,
      tunnelInfo.credentials
    );
    console.log(`✅ Credentials written to: ${credentialsFile}`);
    
    // Also write a generic credentials.json for compatibility
    await cloudflaredService.writeCredentials('credentials', tunnelInfo.credentials);
    
    // Determine the local port (default to 3000 for frontend)
    const localPort = process.env.FRONTEND_PORT || 3000;
    
    // Generate config YAML
    const configYml = `tunnel: ${dnsRecord.tunnelId}
credentials-file: /etc/cloudflared/${dnsRecord.tunnelId}.json

ingress:
  - hostname: ${tunnelInfo.subdomain || dnsRecord.subdomain}
    service: http://host.docker.internal:${localPort}
  - service: http_status:404`;
    
    // Write config file
    console.log('📝 Writing cloudflared configuration...');
    await cloudflaredService.writeConfig(configYml);
    console.log('✅ Configuration written successfully');
    
    // Check if cloudflared is already running
    if (await cloudflaredService.isRunning()) {
      console.log('🔄 Cloudflared is already running. Restarting with new configuration...');
      await cloudflaredService.restart();
      console.log('✅ Cloudflared restarted successfully');
    } else {
      console.log('🚀 Starting cloudflared container...');
      await cloudflaredService.start();
      console.log('✅ Cloudflared started successfully');
    }
    
    // Show the public URL
    console.log('');
    console.log('🌍 Your application is now accessible at:');
    console.log(`   https://${tunnelInfo.subdomain || dnsRecord.subdomain}`);
    console.log('');
    
    // Get initial logs to verify startup
    console.log('📋 Cloudflared container logs:');
    const logs = await cloudflaredService.getLogs(10);
    console.log(logs);
    
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