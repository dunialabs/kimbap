#!/usr/bin/env node

/**
 * 压缩独立部署包为分发文件
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class PackageCompressor {
  constructor() {
    this.rootDir = process.cwd();
    this.platform = process.platform;
    this.arch = process.arch;
    this.distDir = path.join(this.rootDir, 'dist');
    this.packageName = `kimbap-console-standalone-${this.platform}-${this.arch}`;
    this.packageDir = path.join(this.distDir, this.packageName);
  }

  async compress() {
    if (!fs.existsSync(this.packageDir)) {
      console.error(`❌ Package directory not found: ${this.packageDir}`);
      console.log('Please run "npm run build:standalone" first.');
      process.exit(1);
    }

    console.log(`🗜️  Compressing package for ${this.platform}-${this.arch}...`);

    try {
      const outputFile = path.join(this.distDir, `${this.packageName}.${this.getArchiveExtension()}`);
      
      // 删除已存在的压缩包
      if (fs.existsSync(outputFile)) {
        fs.unlinkSync(outputFile);
      }

      // 创建压缩包
      process.chdir(this.distDir);
      
      if (this.platform === 'win32') {
        // Windows: 使用PowerShell创建ZIP
        const cmd = `powershell -command "Compress-Archive -Path '${this.packageName}' -DestinationPath '${this.packageName}.zip' -CompressionLevel Optimal"`;
        execSync(cmd, { stdio: 'inherit' });
      } else {
        // Mac/Linux: 使用tar.gz
        const cmd = `tar -czf "${this.packageName}.tar.gz" "${this.packageName}"`;
        execSync(cmd, { stdio: 'inherit' });
      }

      const compressedSize = this.getFileSize(outputFile);
      const originalSize = this.getDirectorySize(this.packageDir);
      const compressionRatio = ((originalSize - compressedSize) / originalSize * 100).toFixed(1);

      console.log(`✅ Package compressed successfully!`);
      console.log(`📁 File: ${outputFile}`);
      console.log(`📊 Original: ${originalSize}MB → Compressed: ${compressedSize}MB (${compressionRatio}% reduction)`);
      
      // 创建校验和
      await this.createChecksum(outputFile);
      
      // 创建发布说明
      await this.createReleaseNotes(outputFile);

    } catch (error) {
      console.error(`❌ Compression failed: ${error.message}`);
      process.exit(1);
    }
  }

  getArchiveExtension() {
    return this.platform === 'win32' ? 'zip' : 'tar.gz';
  }

  getFileSize(filePath) {
    const stats = fs.statSync(filePath);
    return Math.round(stats.size / 1024 / 1024);
  }

  getDirectorySize(dir) {
    let size = 0;
    
    const walk = (currentPath) => {
      try {
        const entries = fs.readdirSync(currentPath, { withFileTypes: true });
        
        for (const entry of entries) {
          const fullPath = path.join(currentPath, entry.name);
          
          if (entry.isDirectory()) {
            walk(fullPath);
          } else {
            size += fs.statSync(fullPath).size;
          }
        }
      } catch (error) {
        // 忽略权限错误
      }
    };
    
    walk(dir);
    return Math.round(size / 1024 / 1024);
  }

  async createChecksum(filePath) {
    try {
      console.log('🔐 Creating checksum...');
      
      const crypto = require('crypto');
      const fileBuffer = fs.readFileSync(filePath);
      const hashSum = crypto.createHash('sha256');
      hashSum.update(fileBuffer);
      const checksum = hashSum.digest('hex');
      
      const checksumFile = `${filePath}.sha256`;
      const checksumContent = `${checksum}  ${path.basename(filePath)}\n`;
      fs.writeFileSync(checksumFile, checksumContent);
      
      console.log(`✅ Checksum: ${checksum.substring(0, 16)}...`);
      return checksum;
    } catch (error) {
      console.warn(`⚠️  Failed to create checksum: ${error.message}`);
      return null;
    }
  }

  async createReleaseNotes(filePath) {
    const releaseNotes = `# KIMBAP Console 独立部署包

## 包信息
- **文件名**: ${path.basename(filePath)}
- **平台**: ${this.platform}-${this.arch}
- **大小**: ${this.getFileSize(filePath)}MB
- **构建时间**: ${new Date().toISOString()}

## 部署步骤

### 1. 解压包文件
\`\`\`bash
# Windows
# 右键解压 ZIP 文件

# Mac/Linux
tar -xzf ${path.basename(filePath)}
\`\`\`

### 2. 启动应用
\`\`\`bash
# Windows
scripts\\start.bat

# Mac/Linux
./scripts/start.sh
\`\`\`

### 3. 访问应用
在浏览器中打开: http://localhost:3000

## 数据库要求

应用支持以下数据库配置（启动时自动检测）：

1. **Docker PostgreSQL** (推荐)
2. **本地 PostgreSQL 安装**  
3. **云端 PostgreSQL 数据库**

详细配置请参考包内的 README.md 文件。

## 技术支持

如遇问题请查看：
- \`logs/\` 目录下的日志文件
- 包内的 README.md 文件
- 项目 GitHub 页面

---
KIMBAP Console Team`;

    const notesFile = path.join(this.distDir, `${this.packageName}-RELEASE-NOTES.md`);
    fs.writeFileSync(notesFile, releaseNotes);
    
    console.log(`📄 Release notes: ${notesFile}`);
  }
}

if (require.main === module) {
  const compressor = new PackageCompressor();
  compressor.compress();
}

module.exports = PackageCompressor;