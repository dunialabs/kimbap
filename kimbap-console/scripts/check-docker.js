#!/usr/bin/env node

const { execSync } = require('child_process');

function checkDocker() {
  try {
    // Check if Docker is installed
    execSync('docker --version', { stdio: 'pipe' });
    
    // Check if Docker daemon is running
    execSync('docker ps', { stdio: 'pipe' });
    
    console.log('✅ Docker is installed and running');
    return true;
  } catch (error) {
    if (error.message.includes('command not found') || error.message.includes('not found')) {
      console.error('❌ Docker is not installed');
      console.log('\n📥 Please install Docker Desktop:');
      console.log('   - macOS: https://docs.docker.com/desktop/install/mac-install/');
      console.log('   - Windows: https://docs.docker.com/desktop/install/windows-install/');
      console.log('   - Linux: https://docs.docker.com/desktop/install/linux-install/');
    } else if (error.message.includes('docker daemon') || error.message.includes('daemon running')) {
      console.error('❌ Docker is installed but not running');
      console.log('\n🚀 Please start Docker Desktop:');
      console.log('   - macOS: Open Docker Desktop from Applications');
      console.log('   - Windows: Open Docker Desktop from Start menu');
      console.log('   - Linux: Start Docker service: sudo systemctl start docker');
      console.log('\n⏱️  Wait for Docker Desktop to fully start (look for the whale icon in your system tray)');
    } else {
      console.error('❌ Unknown Docker error:', error.message);
    }
    return false;
  }
}

if (require.main === module) {
  const isDockerReady = checkDocker();
  process.exit(isDockerReady ? 0 : 1);
}

module.exports = { checkDocker };