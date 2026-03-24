/**
 * Cloudflared configuration template generator
 * This template is used by both protocol-10011 and start-cloudflared-auto.js
 */

const os = require('os');

/**
 * Determine if cloudflared is running in Docker
 * @returns {boolean} True if running in Docker
 */
function isRunningInDocker() {
  // Check if CLOUDFLARED_IN_DOCKER env var is set
  if (process.env.CLOUDFLARED_IN_DOCKER === 'true') {
    return true;
  }
  
  // Default assumption: we use Docker for cloudflared in this project
  return true;
}

/**
 * Get the appropriate host address based on environment
 * @param {boolean} isDocker - Whether running in Docker
 * @returns {string} The host address
 */
function getHostAddress(isDocker) {
  if (isDocker) {
    // Docker on different platforms
    const platform = os.platform();
    if (platform === 'linux') {
      // On Linux, Docker can use host network or bridge network
      // host.docker.internal is not available on Linux by default
      // Use 172.17.0.1 (default Docker bridge gateway) or host mode
      return process.env.DOCKER_HOST || '172.17.0.1';
    } else {
      // macOS and Windows support host.docker.internal
      return 'host.docker.internal';
    }
  } else {
    // Running natively, use localhost
    return 'localhost';
  }
}

/**
 * Get the credentials file path based on environment
 * @param {string} tunnelId - The tunnel ID
 * @param {boolean} isDocker - Whether running in Docker
 * @returns {string} The credentials file path
 */
function getCredentialsPath(tunnelId, isDocker) {
  if (isDocker) {
    // Docker container path
    return `/etc/cloudflared/${tunnelId}.json`;
  } else {
    // Local file system path
    const path = require('path');
    return path.join(process.cwd(), 'cloudflared', `${tunnelId}.json`);
  }
}

/**
 * Generate cloudflared config YAML
 * @param {Object} params - Configuration parameters
 * @param {string} params.tunnelId - The tunnel ID
 * @param {string} params.subdomain - The subdomain/hostname
 * @param {number} [params.frontendPort] - Frontend port (default from env or 3000)
 * @param {number} [params.backendPort] - Backend port (default from env or 3002)
 * @param {boolean} [params.isDocker] - Whether cloudflared runs in Docker (auto-detected if not specified)
 * @returns {string} The config YAML content
 */
function generateCloudflaredConfig(params) {
  const {
    tunnelId,
    subdomain,
    frontendPort = process.env.FRONTEND_PORT || 3000,
    backendPort = process.env.BACKEND_PORT || 3002,
    isDocker = isRunningInDocker()
  } = params;

  const hostAddress = getHostAddress(isDocker);
  const credentialsPath = getCredentialsPath(tunnelId, isDocker);

  return `tunnel: ${tunnelId}
credentials-file: ${credentialsPath}

# Cloudflared configuration with WebSocket support
ingress:
  # Backend API service with WebSocket support
  # Matches /mcp, /admin, /health endpoints and their sub-paths
  - hostname: ${subdomain}
    path: ^/(mcp|admin|health)(/.*)?$
    service: http://${hostAddress}:${backendPort}
    originRequest:
      # Disable TLS verification for local development
      noTLSVerify: true
      # Preserve the original host header
      httpHostHeader: ${subdomain}
      # Connection timeout (default: 30s, increase for slow connections)
      connectTimeout: 30s
      # Keep-alive timeout for persistent connections (default: 90s)
      keepAliveTimeout: 90s
      # Maximum idle connections (default: 100)
      keepAliveConnections: 100
      # Disable chunked encoding if needed
      disableChunkedEncoding: false
      # Origin server name for SNI
      originServerName: ${subdomain}
      # Proxy type (empty string means HTTP/WebSocket auto-detect)
      proxyType: ""
      # TCP keep-alive interval (default: 30s)
      tcpKeepAlive: 30s
      # No happy eyeballs (use IPv4 only if you have issues)
      noHappyEyeballs: false
      # HTTP2 origin support (default: false, set true for better performance)
      http2Origin: false

  # Frontend service - catches all other paths
  - hostname: ${subdomain}
    service: http://${hostAddress}:${frontendPort}
    originRequest:
      noTLSVerify: true
      connectTimeout: 10s
      keepAliveTimeout: 90s
  
  # Catch-all rule
  - service: http_status:404`;
}

module.exports = {
  generateCloudflaredConfig
};