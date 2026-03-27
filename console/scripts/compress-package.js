#!/usr/bin/env node

/**
 * 
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
      
      // 
      if (fs.existsSync(outputFile)) {
        fs.unlinkSync(outputFile);
      }

      // 
      process.chdir(this.distDir);
      
      if (this.platform === 'win32') {
        // Windows: PowerShellZIP
        const cmd = `powershell -command "Compress-Archive -Path '${this.packageName}' -DestinationPath '${this.packageName}.zip' -CompressionLevel Optimal"`;
        execSync(cmd, { stdio: 'inherit' });
      } else {
        // Mac/Linux: tar.gz
        const cmd = `tar -czf "${this.packageName}.tar.gz" "${this.packageName}"`;
        execSync(cmd, { stdio: 'inherit' });
      }

      const compressedSize = this.getFileSize(outputFile);
      const originalSize = this.getDirectorySize(this.packageDir);
      const compressionRatio = ((originalSize - compressedSize) / originalSize * 100).toFixed(1);

      console.log(`✅ Package compressed successfully!`);
      console.log(`📁 File: ${outputFile}`);
      console.log(`📊 Original: ${originalSize}MB → Compressed: ${compressedSize}MB (${compressionRatio}% reduction)`);
      
      // 
      await this.createChecksum(outputFile);
      
      // 
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
        // 
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
    const releaseNotes = `# Kimbap Console 

## 
- ****: ${path.basename(filePath)}
- ****: ${this.platform}-${this.arch}
- ****: ${this.getFileSize(filePath)}MB
- ****: ${new Date().toISOString()}

## 

### 1. 
\`\`\`bash
# Windows
#  ZIP 

# Mac/Linux
tar -xzf ${path.basename(filePath)}
\`\`\`

### 2. 
\`\`\`bash
# Windows
scripts\\start.bat

# Mac/Linux
./scripts/start.sh
\`\`\`

### 3. 
: http://localhost:3000

## 

（）：

1. **Docker PostgreSQL** ()
2. ** PostgreSQL **  
3. ** PostgreSQL **

 README.md 。

## 

：
- \`logs/\` 
-  README.md 
-  GitHub 

---
Kimbap Console Team`;

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