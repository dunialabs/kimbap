#!/usr/bin/env node

/**
 * 依赖下载脚本
 * 下载 Node.js 和 PostgreSQL 便携版
 */

const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

class DependencyDownloader {
  constructor() {
    this.platform = process.platform;
    this.arch = process.arch === 'x64' ? 'x64' : 'arm64';
    this.tempDir = path.resolve(__dirname, '../../temp-downloads');
    
    // 版本配置
    this.nodeVersion = '20.11.0';
    this.postgresVersion = '16.1';
    
    // 确保临时目录存在
    if (!fs.existsSync(this.tempDir)) {
      fs.mkdirSync(this.tempDir, { recursive: true });
    }
  }

  async downloadAll() {
    console.log('📥 Starting dependency downloads...');
    console.log(`Platform: ${this.platform}-${this.arch}`);
    
    try {
      await this.downloadNode();
      await this.downloadPostgreSQL();
      
      console.log('✅ All dependencies downloaded successfully!');
      console.log(`📁 Files saved to: ${this.tempDir}`);
      
    } catch (error) {
      console.error('❌ Download failed:', error);
      process.exit(1);
    }
  }

  async downloadNode() {
    console.log(`📥 Downloading Node.js v${this.nodeVersion}...`);
    
    const nodeUrl = this.getNodeDownloadUrl();
    const fileName = path.basename(nodeUrl);
    const filePath = path.join(this.tempDir, fileName);
    
    console.log(`URL: ${nodeUrl}`);
    
    if (fs.existsSync(filePath)) {
      console.log('✅ Node.js already downloaded, skipping...');
      return;
    }
    
    await this.downloadFile(nodeUrl, filePath);
    console.log('✅ Node.js downloaded successfully');
  }

  async downloadPostgreSQL() {
    console.log(`📥 Downloading PostgreSQL v${this.postgresVersion}...`);
    
    // 使用预编译的便携版 PostgreSQL
    const pgInfo = this.getPostgreSQLInfo();
    const filePath = path.join(this.tempDir, pgInfo.fileName);
    
    console.log(`URL: ${pgInfo.url}`);
    
    if (fs.existsSync(filePath)) {
      console.log('✅ PostgreSQL already downloaded, skipping...');
      return;
    }
    
    await this.downloadFile(pgInfo.url, filePath);
    console.log('✅ PostgreSQL downloaded successfully');
  }

  getNodeDownloadUrl() {
    const baseUrl = 'https://nodejs.org/dist';
    let fileName;
    
    switch (this.platform) {
      case 'win32':
        fileName = `node-v${this.nodeVersion}-win-${this.arch}.zip`;
        break;
      case 'darwin':
        fileName = `node-v${this.nodeVersion}-darwin-${this.arch}.tar.gz`;
        break;
      case 'linux':
        fileName = `node-v${this.nodeVersion}-linux-${this.arch}.tar.xz`;
        break;
      default:
        throw new Error(`Unsupported platform: ${this.platform}`);
    }
    
    return `${baseUrl}/v${this.nodeVersion}/${fileName}`;
  }

  getPostgreSQLInfo() {
    // 使用 PostgreSQL 官方二进制包
    switch (this.platform) {
      case 'win32':
        return {
          url: `https://get.enterprisedb.com/postgresql/postgresql-${this.postgresVersion}-1-windows-x64-binaries.zip`,
          fileName: `postgresql-${this.postgresVersion}-1-windows-x64-binaries.zip`
        };
        
      case 'darwin':
        // macOS 使用预编译的 PostgreSQL 二进制包
        if (this.arch === 'arm64') {
          return {
            url: `https://sbp.enterprisedb.com/getfile.jsp?fileid=1258649&_ga=2.99696307.1234567890.1234567890-1234567890.1234567890`,
            fileName: `postgresql-${this.postgresVersion}-osx-binaries.zip`,
            // 备用URL - EDB官方二进制包
            alternativeUrl: `https://get.enterprisedb.com/postgresql/postgresql-${this.postgresVersion}-1-osx-binaries.zip`
          };
        } else {
          return {
            url: `https://get.enterprisedb.com/postgresql/postgresql-${this.postgresVersion}-1-osx-binaries.zip`,
            fileName: `postgresql-${this.postgresVersion}-1-osx-binaries.zip`
          };
        }
        
      case 'linux':
        return {
          url: `https://get.enterprisedb.com/postgresql/postgresql-${this.postgresVersion}-1-linux-x64-binaries.tar.gz`,
          fileName: `postgresql-${this.postgresVersion}-1-linux-x64-binaries.tar.gz`
        };
        
      default:
        throw new Error(`Unsupported platform: ${this.platform}`);
    }
  }

  // 获取更可靠的 PostgreSQL 下载方案
  getPortablePostgreSQLInfo() {
    // 使用预编译的便携版 PostgreSQL
    const version = '16.1';
    const baseUrl = 'https://ftp.postgresql.org/pub/binary';
    
    switch (this.platform) {
      case 'win32':
        return {
          url: `${baseUrl}/v${version}/win32/postgresql-${version}-1-windows-x64-binaries.zip`,
          fileName: `postgresql-${version}-windows-binaries.zip`,
          // 备用下载地址
          mirrors: [
            `https://get.enterprisedb.com/postgresql/postgresql-${version}-1-windows-x64-binaries.zip`,
            `https://ftp.postgresql.org/pub/binary/v${version}/win32/postgresql-${version}-1-windows-x64-binaries.zip`
          ]
        };
        
      case 'darwin':
        return {
          url: `${baseUrl}/v${version}/macos/postgresql-${version}-1-osx-binaries.zip`,
          fileName: `postgresql-${version}-darwin-binaries.zip`,
          mirrors: [
            `https://get.enterprisedb.com/postgresql/postgresql-${version}-1-osx-binaries.zip`,
            // 使用 Homebrew 的便携版本作为备选
            `https://ghcr.io/v2/homebrew/core/postgresql/blobs/sha256:123456`
          ]
        };
        
      case 'linux':
        return {
          url: `${baseUrl}/v${version}/linux/postgresql-${version}-1-linux-x64-binaries.tar.gz`,
          fileName: `postgresql-${version}-linux-binaries.tar.gz`,
          mirrors: [
            `https://get.enterprisedb.com/postgresql/postgresql-${version}-1-linux-x64-binaries.tar.gz`
          ]
        };
        
      default:
        throw new Error(`Unsupported platform: ${this.platform}`);
    }
  }

  async downloadFile(url, destination) {
    return new Promise((resolve, reject) => {
      console.log(`Downloading: ${path.basename(destination)}`);
      
      const file = fs.createWriteStream(destination);
      let downloadedBytes = 0;
      let totalBytes = 0;
      
      const request = https.get(url, (response) => {
        // 处理重定向
        if (response.statusCode === 301 || response.statusCode === 302) {
          file.close();
          fs.unlinkSync(destination);
          return this.downloadFile(response.headers.location, destination)
            .then(resolve)
            .catch(reject);
        }
        
        if (response.statusCode !== 200) {
          file.close();
          fs.unlinkSync(destination);
          reject(new Error(`HTTP ${response.statusCode}: ${response.statusMessage}`));
          return;
        }
        
        totalBytes = parseInt(response.headers['content-length'] || '0', 10);
        
        response.on('data', (chunk) => {
          downloadedBytes += chunk.length;
          
          if (totalBytes > 0) {
            const percent = ((downloadedBytes / totalBytes) * 100).toFixed(1);
            const mb = (downloadedBytes / 1024 / 1024).toFixed(1);
            const totalMb = (totalBytes / 1024 / 1024).toFixed(1);
            
            process.stdout.write(`\\rProgress: ${percent}% (${mb}/${totalMb} MB)`);
          }
        });
        
        response.pipe(file);
        
        file.on('finish', () => {
          file.close();
          console.log('\\n✅ Download completed');
          resolve();
        });
        
      }).on('error', (err) => {
        file.close();
        if (fs.existsSync(destination)) {
          fs.unlinkSync(destination);
        }
        reject(err);
      });
      
      // 设置超时
      request.setTimeout(300000, () => { // 5 minutes
        request.abort();
        reject(new Error('Download timeout'));
      });
    });
  }

  // 获取文件大小（MB）
  getFileSize(filePath) {
    if (!fs.existsSync(filePath)) return 0;
    const stats = fs.statSync(filePath);
    return (stats.size / 1024 / 1024).toFixed(1);
  }

  // 列出已下载的文件
  listDownloads() {
    console.log('\\n📁 Downloaded files:');
    
    if (!fs.existsSync(this.tempDir)) {
      console.log('No downloads directory found.');
      return;
    }
    
    const files = fs.readdirSync(this.tempDir);
    
    if (files.length === 0) {
      console.log('No files downloaded yet.');
      return;
    }
    
    files.forEach(file => {
      const filePath = path.join(this.tempDir, file);
      const size = this.getFileSize(filePath);
      console.log(`  - ${file} (${size} MB)`);
    });
  }
}

// CLI 使用
if (require.main === module) {
  const args = process.argv.slice(2);
  const downloader = new DependencyDownloader();
  
  if (args.includes('--list')) {
    downloader.listDownloads();
  } else {
    downloader.downloadAll();
  }
}

module.exports = DependencyDownloader;