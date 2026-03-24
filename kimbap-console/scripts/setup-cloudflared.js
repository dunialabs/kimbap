#!/usr/bin/env node

/**
 * Setup cloudflared configuration (without Docker)
 * Creates configuration directory and files for manual cloudflared setup
 */

const fs = require('fs');
const path = require('path');

// Colors for console output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  red: '\x1b[31m',
  cyan: '\x1b[36m'
};

function log(message, color = '') {
  console.log(`${color}${message}${colors.reset}`);
}

function createCloudflaredConfig() {
  const configDir = path.join(__dirname, '..', 'cloudflared');
  
  // Create directory if it doesn't exist
  if (!fs.existsSync(configDir)) {
    fs.mkdirSync(configDir, { recursive: true });
  }
  
  // Create a default config file if it doesn't exist
  const configPath = path.join(configDir, 'config.yml');
  if (!fs.existsSync(configPath)) {
    const defaultConfig = `# Cloudflare Tunnel configuration
# This file is auto-generated. Please update with your tunnel configuration.

# Example configuration:
# tunnel: YOUR_TUNNEL_ID
# credentials-file: /etc/cloudflared/credentials.json

# ingress:
#   - hostname: your-domain.com
#     service: http://localhost:3000
#   - hostname: api.your-domain.com
#     service: http://localhost:3002
#   - service: http_status:404

# Note: You'll need to set up your tunnel with:
# 1. Create a tunnel: cloudflared tunnel create kimbap-console
# 2. Route traffic: cloudflared tunnel route dns kimbap-console your-domain.com
# 3. Copy the credentials file to ./cloudflared/credentials.json
# 4. Set CLOUDFLARE_TUNNEL_TOKEN environment variable
`;
    
    fs.writeFileSync(configPath, defaultConfig);
    log('📝 Created default cloudflared config file at ./cloudflared/config.yml', colors.cyan);
    log('⚠️  Please update the config file with your tunnel settings', colors.yellow);
  }
}

async function main() {
  log('\n🚀 Setting up Cloudflared configuration (Manual setup)...', colors.bright + colors.blue);
  
  // Create cloudflared configuration directory and files
  createCloudflaredConfig();
  
  log('\n✅ Cloudflared configuration setup complete!', colors.bright + colors.green);
  log('\n📌 To use Cloudflared manually:', colors.cyan);
  log('   1. Install cloudflared on your system', colors.cyan);
  log('   2. Configure your Cloudflare tunnel in ./cloudflared/config.yml', colors.cyan);
  log('   3. Set CLOUDFLARE_TUNNEL_TOKEN environment variable', colors.cyan);
  log('   4. Run cloudflared with your configuration', colors.cyan);
  log('\n💡 For more info: https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/', colors.blue);
}

// Run if called directly
if (require.main === module) {
  main().catch(error => {
    log(`❌ Error: ${error.message}`, colors.red);
    process.exit(1);
  });
}

module.exports = { createCloudflaredConfig };