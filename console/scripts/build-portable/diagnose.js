#!/usr/bin/env node

/**
 * 
 * 
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class BuildDiagnostic {
  constructor() {
    this.rootDir = path.resolve(__dirname, '../..');
    this.outputDir = path.join(this.rootDir, 'dist', `kimbap-console-${process.platform}`);
  }

  diagnose() {
    console.log('🔍 Diagnosing portable build...\n');
    
    this.checkEnvironment();
    this.checkBuildOutput();
    this.checkNodeJs();
    this.checkNextJs();
    this.checkStartupScript();
    this.generateReport();
  }

  checkEnvironment() {
    console.log('📋 Environment Check:');
    console.log(`- Node.js: ${process.version}`);
    console.log(`- Platform: ${process.platform}-${process.arch}`);
    console.log(`- Working Directory: ${this.rootDir}`);
    
    // 
    const commands = ['npm', 'tar', 'gzip'];
    commands.forEach(cmd => {
      try {
        execSync(`which ${cmd}`, { stdio: 'ignore' });
        console.log(`- ${cmd}: ✅`);
      } catch (error) {
        console.log(`- ${cmd}: ❌`);
      }
    });
    console.log('');
  }

  checkBuildOutput() {
    console.log('📁 Build Output Check:');
    console.log(`- Output Directory: ${fs.existsSync(this.outputDir) ? '✅' : '❌'} ${this.outputDir}`);
    
    if (fs.existsSync(this.outputDir)) {
      const dirs = ['app', 'node', 'scripts', 'config', 'postgresql'];
      dirs.forEach(dir => {
        const dirPath = path.join(this.outputDir, dir);
        const exists = fs.existsSync(dirPath);
        let size = '';
        if (exists) {
          try {
            const stats = this.getDirSize(dirPath);
            size = ` (${this.formatSize(stats)})`;
          } catch (e) {
            size = ' (size unknown)';
          }
        }
        console.log(`- ${dir}/:${size} ${exists ? '✅' : '❌'}`);
      });

      // 
      const rootFiles = ['README.txt'];
      rootFiles.forEach(file => {
        const filePath = path.join(this.outputDir, file);
        console.log(`- ${file}: ${fs.existsSync(filePath) ? '✅' : '❌'}`);
      });
    }
    console.log('');
  }

  checkNodeJs() {
    console.log('🔧 Node.js Check:');
    const nodeExec = path.join(this.outputDir, 'node/bin/node');
    
    if (fs.existsSync(nodeExec)) {
      console.log('- Node.js executable: ✅');
      try {
        const version = execSync(`"${nodeExec}" --version`, { encoding: 'utf8' }).trim();
        console.log(`- Version: ${version}`);
        
        // 
        const stats = fs.statSync(nodeExec);
        const isExecutable = !!(stats.mode & parseInt('111', 8));
        console.log(`- Executable permissions: ${isExecutable ? '✅' : '❌'}`);
        
        // 
        const sizeMB = Math.round(stats.size / 1024 / 1024);
        console.log(`- File size: ${sizeMB}MB`);
        
      } catch (error) {
        console.log('- Version check: ❌', error.message);
      }
    } else {
      console.log('- Node.js executable: ❌');
      
      // 
      const altPaths = [
        'node/node',
        'node/bin/node.exe'
      ];
      altPaths.forEach(altPath => {
        const fullPath = path.join(this.outputDir, altPath);
        if (fs.existsSync(fullPath)) {
          console.log(`- Found at alternative path: ${altPath} ⚠️`);
        }
      });
    }
    console.log('');
  }

  checkNextJs() {
    console.log('⚡ Next.js Build Check:');
    const nextDir = path.join(this.outputDir, 'app/.next');
    
    if (fs.existsSync(nextDir)) {
      console.log('- .next directory: ✅');
      
      const requiredFiles = [
        'BUILD_ID',
        'routes-manifest.json',
        'prerender-manifest.json',
        'build-manifest.json',
        'react-loadable-manifest.json',
        'app-build-manifest.json'
      ];
      
      requiredFiles.forEach(file => {
        const filePath = path.join(nextDir, file);
        const exists = fs.existsSync(filePath);
        let info = '';
        if (exists) {
          try {
            const stats = fs.statSync(filePath);
            info = ` (${this.formatSize(stats.size)})`;
          } catch (e) {
            info = '';
          }
        }
        console.log(`- ${file}:${info} ${exists ? '✅' : '❌'}`);
      });

      // 
      const requiredDirs = ['server', 'static'];
      requiredDirs.forEach(dir => {
        const dirPath = path.join(nextDir, dir);
        console.log(`- ${dir}/ directory: ${fs.existsSync(dirPath) ? '✅' : '❌'}`);
      });

    } else {
      console.log('- .next directory: ❌');
    }
    
    // 
    const appFiles = [
      'app/package.json',
      'app/next.config.mjs',
      'app/.env.local'
    ];
    
    appFiles.forEach(file => {
      const filePath = path.join(this.outputDir, file);
      console.log(`- ${file}: ${fs.existsSync(filePath) ? '✅' : '❌'}`);
    });
    
    console.log('');
  }

  checkStartupScript() {
    console.log('📜 Startup Script Check:');
    
    const scriptFile = process.platform === 'win32' ? 'start.bat' : 'start.sh';
    const scriptPath = path.join(this.outputDir, 'scripts', scriptFile);
    
    if (fs.existsSync(scriptPath)) {
      console.log(`- ${scriptFile}: ✅`);
      
      const stats = fs.statSync(scriptPath);
      if (process.platform !== 'win32') {
        const isExecutable = !!(stats.mode & parseInt('111', 8));
        console.log(`- Executable permissions: ${isExecutable ? '✅' : '❌'}`);
      }
      
      // 
      try {
        const content = fs.readFileSync(scriptPath, 'utf8');
        const checks = [
          { pattern: /node.*next.*start/, name: 'Next.js start command' },
          { pattern: /DATABASE_URL/, name: 'Database URL configuration' },
          { pattern: /localhost:3000/, name: 'Port configuration' }
        ];
        
        checks.forEach(check => {
          const found = check.pattern.test(content);
          console.log(`- ${check.name}: ${found ? '✅' : '❌'}`);
        });
        
      } catch (error) {
        console.log('- Script content check: ❌', error.message);
      }
    } else {
      console.log(`- ${scriptFile}: ❌`);
    }
    console.log('');
  }

  getDirSize(dirPath) {
    let totalSize = 0;
    const files = fs.readdirSync(dirPath);
    
    for (const file of files) {
      const filePath = path.join(dirPath, file);
      const stats = fs.statSync(filePath);
      
      if (stats.isDirectory()) {
        totalSize += this.getDirSize(filePath);
      } else {
        totalSize += stats.size;
      }
    }
    
    return totalSize;
  }

  formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  generateReport() {
    console.log('📊 Summary Report:');
    
    const issues = [];
    const warnings = [];
    
    // 
    const criticalPaths = [
      'node/bin/node',
      'app/.next/BUILD_ID',
      'scripts/' + (process.platform === 'win32' ? 'start.bat' : 'start.sh'),
      'app/package.json'
    ];
    
    criticalPaths.forEach(pathStr => {
      const fullPath = path.join(this.outputDir, pathStr);
      if (!fs.existsSync(fullPath)) {
        issues.push(`Missing critical file: ${pathStr}`);
      }
    });
    
    // 
    if (fs.existsSync(this.outputDir)) {
      try {
        const totalSize = this.getDirSize(this.outputDir);
        const sizeMB = Math.round(totalSize / 1024 / 1024);
        console.log(`- Total package size: ${this.formatSize(totalSize)}`);
        
        if (sizeMB < 100) {
          warnings.push(`Package size is unusually small (${sizeMB}MB)`);
        }
      } catch (error) {
        warnings.push('Could not calculate package size');
      }
    }
    
    // 
    if (issues.length === 0 && warnings.length === 0) {
      console.log('- Status: ✅ All checks passed!');
      console.log('- Build appears to be complete and ready for use');
    } else {
      if (issues.length > 0) {
        console.log('- Status: ❌ Issues found');
        issues.forEach(issue => console.log(`  • ${issue}`));
      }
      
      if (warnings.length > 0) {
        console.log('- Warnings: ⚠️');
        warnings.forEach(warning => console.log(`  • ${warning}`));
      }
    }
    
    console.log('\n💡 Next steps:');
    if (issues.length > 0) {
      console.log('- Fix the issues listed above');
      console.log('- Run: npm run build:complete');
      console.log('- Or: npm run build:portable');
    } else {
      console.log('- Test the build: cd dist/kimbap-console-' + process.platform);
      console.log('- Start the app: ./scripts/' + (process.platform === 'win32' ? 'start.bat' : 'start.sh'));
    }
  }
}

// CLI 
if (require.main === module) {
  const args = process.argv.slice(2);
  
  if (args.includes('--help') || args.includes('-h')) {
    console.log(`


:
  npm run build:diagnose              # 
  node scripts/build-portable/diagnose.js  # 

:
  --help, -h                         # 

:
  npm run build:diagnose
  npm run build:check                # 
  npm run build:complete             # 
`);
    process.exit(0);
  }
  
  console.log('🔧 Kimbap Console \n');
  const diagnostic = new BuildDiagnostic();
  diagnostic.diagnose();
}

module.exports = BuildDiagnostic;