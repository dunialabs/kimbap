#!/usr/bin/env node

/**
 * 解压缩工具
 * 处理 zip, tar.gz, tar.xz 等格式的解压
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class ExtractUtils {
  constructor() {
    this.platform = process.platform;
  }

  async extractArchive(archivePath, extractDir) {
    const ext = path.extname(archivePath).toLowerCase();
    const fullName = path.basename(archivePath).toLowerCase();
    
    console.log(`📦 Extracting ${path.basename(archivePath)}...`);
    
    // 确保目标目录存在
    if (!fs.existsSync(extractDir)) {
      fs.mkdirSync(extractDir, { recursive: true });
    }
    
    try {
      if (fullName.includes('.tar.gz') || fullName.includes('.tgz')) {
        await this.extractTarGz(archivePath, extractDir);
      } else if (fullName.includes('.tar.xz')) {
        await this.extractTarXz(archivePath, extractDir);
      } else if (ext === '.zip') {
        await this.extractZip(archivePath, extractDir);
      } else if (ext === '.dmg') {
        await this.extractDmg(archivePath, extractDir);
      } else {
        throw new Error(`Unsupported archive format: ${ext}`);
      }
      
      console.log(`✅ Extracted to ${extractDir}`);
    } catch (error) {
      throw new Error(`Extraction failed: ${error.message}`);
    }
  }

  async extractZip(archivePath, extractDir) {
    if (this.platform === 'win32') {
      // Windows: 使用 PowerShell
      const command = `powershell -Command "Expand-Archive -Path '${archivePath}' -DestinationPath '${extractDir}' -Force"`;
      execSync(command, { stdio: 'inherit' });
    } else {
      // macOS/Linux: 使用 unzip
      try {
        execSync(`unzip -q "${archivePath}" -d "${extractDir}"`, { stdio: 'inherit' });
      } catch (error) {
        // 如果系统没有 unzip，尝试使用 Node.js 实现
        await this.extractZipNode(archivePath, extractDir);
      }
    }
  }

  async extractTarGz(archivePath, extractDir) {
    const command = `tar -xzf "${archivePath}" -C "${extractDir}"`;
    execSync(command, { stdio: 'inherit' });
  }

  async extractTarXz(archivePath, extractDir) {
    const command = `tar -xJf "${archivePath}" -C "${extractDir}"`;
    execSync(command, { stdio: 'inherit' });
  }

  async extractDmg(archivePath, extractDir) {
    if (this.platform !== 'darwin') {
      throw new Error('DMG files can only be extracted on macOS');
    }
    
    // macOS DMG 处理
    const mountPoint = '/tmp/kimbap-dmg-mount';
    
    try {
      // 挂载 DMG
      execSync(`hdiutil attach "${archivePath}" -mountpoint "${mountPoint}" -quiet`, { stdio: 'inherit' });
      
      // 复制内容
      execSync(`cp -R "${mountPoint}"/* "${extractDir}/"`, { stdio: 'inherit' });
      
      // 卸载 DMG
      execSync(`hdiutil detach "${mountPoint}" -quiet`, { stdio: 'inherit' });
      
    } catch (error) {
      // 确保卸载 DMG
      try {
        execSync(`hdiutil detach "${mountPoint}" -force -quiet`, { stdio: 'ignore' });
      } catch (e) {
        // 忽略卸载错误
      }
      throw error;
    }
  }

  // 纯 Node.js 实现的 ZIP 解压（备用方案）
  async extractZipNode(archivePath, extractDir) {
    console.log('Using Node.js ZIP extraction...');
    
    // 这里可以使用 yauzl 或其他纯 JS 的 zip 库
    // 为了简化，我们先使用系统命令
    throw new Error('Node.js ZIP extraction not implemented. Please install unzip command.');
  }

  // 检查解压后的内容并进行整理
  async organizeExtractedFiles(extractDir, expectedStructure) {
    console.log('📂 Organizing extracted files...');
    
    const files = fs.readdirSync(extractDir);
    
    // 如果解压后只有一个目录，将其内容移到上级
    if (files.length === 1 && fs.statSync(path.join(extractDir, files[0])).isDirectory()) {
      const tempDir = extractDir + '_temp';
      
      // 重命名原目录
      fs.renameSync(extractDir, tempDir);

      const singleDir = path.join(tempDir, files[0]);
      
      // 将单个子目录重命名为目标目录
      fs.renameSync(singleDir, extractDir);
      
      // 删除临时目录
      fs.rmSync(tempDir, { recursive: true, force: true });
      
      console.log('✅ Files organized');
    }
  }

  // 验证解压后的文件
  validateExtraction(extractDir, expectedFiles) {
    console.log('🔍 Validating extraction...');
    
    for (const expectedFile of expectedFiles) {
      const filePath = path.join(extractDir, expectedFile);
      if (!fs.existsSync(filePath)) {
        throw new Error(`Expected file not found: ${expectedFile}`);
      }
    }
    
    console.log('✅ Extraction validated');
  }
}

// CLI 使用示例
if (require.main === module) {
  const args = process.argv.slice(2);
  
  if (args.length < 2) {
    console.log('Usage: node extract-utils.js <archive> <destination>');
    process.exit(1);
  }
  
  const [archivePath, extractDir] = args;
  const extractor = new ExtractUtils();
  
  extractor.extractArchive(archivePath, extractDir)
    .then(() => {
      console.log('Extraction completed successfully');
    })
    .catch((error) => {
      console.error('Extraction failed:', error);
      process.exit(1);
    });
}

module.exports = ExtractUtils;
