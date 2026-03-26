#!/usr/bin/env node

/**
 * 
 *  zip, tar.gz, tar.xz 
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
    
    // 
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
      // Windows:  PowerShell
      const command = `powershell -Command "Expand-Archive -Path '${archivePath}' -DestinationPath '${extractDir}' -Force"`;
      execSync(command, { stdio: 'inherit' });
    } else {
      // macOS/Linux:  unzip
      try {
        execSync(`unzip -q "${archivePath}" -d "${extractDir}"`, { stdio: 'inherit' });
      } catch (error) {
        //  unzip， Node.js 
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
    
    // macOS DMG 
    const mountPoint = '/tmp/kimbap-dmg-mount';
    
    try {
      //  DMG
      execSync(`hdiutil attach "${archivePath}" -mountpoint "${mountPoint}" -quiet`, { stdio: 'inherit' });
      
      // 
      execSync(`cp -R "${mountPoint}"/* "${extractDir}/"`, { stdio: 'inherit' });
      
      //  DMG
      execSync(`hdiutil detach "${mountPoint}" -quiet`, { stdio: 'inherit' });
      
    } catch (error) {
      //  DMG
      try {
        execSync(`hdiutil detach "${mountPoint}" -force -quiet`, { stdio: 'ignore' });
      } catch (e) {
        // 
      }
      throw error;
    }
  }

  //  Node.js  ZIP （）
  async extractZipNode(archivePath, extractDir) {
    console.log('Using Node.js ZIP extraction...');
    
    //  yauzl  JS  zip 
    // ，
    throw new Error('Node.js ZIP extraction not implemented. Please install unzip command.');
  }

  // 
  async organizeExtractedFiles(extractDir, expectedStructure) {
    console.log('📂 Organizing extracted files...');
    
    const files = fs.readdirSync(extractDir);
    
    // ，
    if (files.length === 1 && fs.statSync(path.join(extractDir, files[0])).isDirectory()) {
      const tempDir = extractDir + '_temp';
      
      // 
      fs.renameSync(extractDir, tempDir);

      const singleDir = path.join(tempDir, files[0]);
      
      // 
      fs.renameSync(singleDir, extractDir);
      
      // 
      fs.rmSync(tempDir, { recursive: true, force: true });
      
      console.log('✅ Files organized');
    }
  }

  // 
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

// CLI 
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
