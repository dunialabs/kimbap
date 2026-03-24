# 📦 便携包构建故障排除指南

本文档总结了在构建 KIMBAP Console 便携包过程中遇到的问题和解决方案，帮助后续开发者避免相同问题。

## 🛠️ 构建过程问题

### 1. autoprefixer 模块找不到

**问题描述**:
```
Error: Cannot find module 'autoprefixer'
```

**原因分析**:
- 构建脚本使用 `npm ci --production` 删除了 devDependencies
- Next.js 构建需要 autoprefixer (在 devDependencies 中)

**解决方案**:
```javascript
// 修改 build-portable.js 中的 buildNextApp 方法
async buildNextApp() {
  // 安装所有依赖（包括devDependencies用于构建）
  execSync('npm ci', { stdio: 'inherit' });
  
  // 构建应用
  execSync('npm run build', { stdio: 'inherit' });
}

// 在 copyAppFiles 方法中再安装生产依赖
async copyAppFiles() {
  // ... 复制文件
  
  // 安装仅生产依赖到最终包中
  process.chdir(appDir);
  execSync('npm ci --production', { stdio: 'inherit' });
}
```

### 2. buttonVariants 导入错误

**问题描述**:
```
Attempted import error: 'buttonVariants' is not exported from '@/components/ui/button'
```

**原因分析**:
- button 组件缺少 buttonVariants 导出
- alert-dialog 组件依赖此导出

**解决方案**:
```typescript
// components/ui/button.tsx 添加以下内容
import { cva, type VariantProps } from 'class-variance-authority'

const buttonVariants = cva(
  'inline-flex items-center justify-center...',
  {
    variants: {
      variant: { /* variants */ },
      size: { /* sizes */ }
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    }
  }
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, ...props }, ref) => {
    return (
      <button
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  }
)

export { Button, buttonVariants }
```

### 3. 静态导出与 API 路由冲突

**问题描述**:
```
Error: export const dynamic = "force-static"/export const revalidate not configured on route "/api/dashboard/overview" with "output: export"
```

**原因分析**:
- `next.config.mjs` 配置了 `output: 'export'` 静态导出
- 静态导出不支持 API 路由
- 便携包需要 API 路由功能

**解决方案**:
创建专门的便携包配置，禁用静态导出：
```javascript
// 在构建脚本中临时替换 next.config.mjs
const portableConfig = `/** @type {import('next').NextConfig} */
const nextConfig = {
  // 便携包配置 - 支持API routes和SSR
  output: undefined, // 禁用静态导出
  
  images: {
    unoptimized: true,
    // ...其他配置
  },
  
  // ...其他配置保持不变
}

export default nextConfig`;

// 备份原配置，使用便携包配置，构建完成后恢复
```

## 🗂️ 文件处理问题

### 4. Node.js 下载文件损坏

**问题描述**:
```
node-v20.11.0-darwin-arm64/bin/node: truncated gzip input: Unknown error: -1
tar: Error exit delayed from previous errors.
```

**原因分析**:
- 网络中断导致下载的 tar.gz 文件不完整
- `gzip -t` 测试显示文件损坏

**解决方案**:
```bash
# 检查文件完整性
gzip -t temp-downloads/node-v20.11.0-darwin-arm64.tar.gz

# 如果文件损坏，删除并重新下载
rm temp-downloads/node-v20.11.0-darwin-arm64.tar.gz
npm run build:portable  # 会自动重新下载
```

### 5. Node.js 文件解压后路径问题

**问题描述**:
```
./scripts/start.sh: line 34: ../node/bin/node: No such file or directory
```

**原因分析**:
- Node.js 解压后在 `node-v20.11.0-darwin-arm64/` 子目录中
- 文件重组织失败，导致路径不正确

**解决方案**:
```bash
# 正确的解压方式，使用 --strip-components=1 去除顶层目录
tar -xzf temp-downloads/node-v20.11.0-darwin-arm64.tar.gz \
    -C dist/kimbap-console-darwin/node \
    --strip-components=1

# 验证可执行文件存在
ls -la dist/kimbap-console-darwin/node/bin/node
```

### 6. Next.js 构建文件缺失

**问题描述**:
```
Error: Could not find a production build in the '.next' directory
Error: ENOENT: no such file or directory, open '.next/routes-manifest.json'
Error: ENOENT: no such file or directory, open '.next/prerender-manifest.json'
```

**原因分析**:
- `.next` 目录复制不完整
- 缺少关键的 manifest 文件
- 缺少 BUILD_ID 文件

**解决方案**:
```bash
# 完整复制 .next 目录
cp -r .next dist/kimbap-console-darwin/app/

# 手动创建缺失的文件
echo "portable-build-$(date +%s)" > dist/kimbap-console-darwin/app/.next/BUILD_ID

# 创建 prerender-manifest.json（如果不存在）
echo '{"version":4,"routes":{},"dynamicRoutes":{},"notFoundRoutes":[],"preview":{"previewModeId":"","previewModeSigningKey":"","previewModeEncryptionKey":""}}' > dist/kimbap-console-darwin/app/.next/prerender-manifest.json
```

## 🚀 最佳实践

### 1. 构建脚本改进

**推荐的构建顺序**:
```javascript
async build() {
  // 1. 环境准备
  await this.prepare();
  
  // 2. 先构建应用（包含所有依赖）
  await this.buildNextApp();
  
  // 3. 下载外部依赖
  await this.downloadNode();
  await this.downloadPostgreSQL();
  
  // 4. 复制和清理应用文件（生产依赖）
  await this.copyAppFiles();
  
  // 5. 创建配置和脚本
  await this.createStartupScripts();
  await this.createConfigFiles();
  await this.createDocumentation();
  
  // 6. 验证和清理
  await this.validateBuild();
  await this.cleanup();
}
```

### 2. 文件完整性验证

**添加验证步骤**:
```javascript
async validateBuild() {
  const requiredFiles = [
    'node/bin/node',
    'app/.next/BUILD_ID',
    'app/.next/routes-manifest.json',
    'app/package.json',
    'scripts/start.sh'
  ];
  
  for (const file of requiredFiles) {
    const filePath = path.join(this.outputDir, file);
    if (!fs.existsSync(filePath)) {
      throw new Error(`Required file not found: ${file}`);
    }
  }
}
```

### 3. 错误处理和重试机制

**下载重试**:
```javascript
async downloadWithRetry(url, destination, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      await this.downloadFile(url, destination);
      
      // 验证文件完整性
      if (destination.endsWith('.tar.gz')) {
        execSync(`gzip -t "${destination}"`);
      }
      
      return; // 成功
    } catch (error) {
      console.warn(`Download attempt ${i + 1} failed:`, error.message);
      if (fs.existsSync(destination)) {
        fs.unlinkSync(destination);
      }
      
      if (i === maxRetries - 1) {
        throw error;
      }
      
      // 等待后重试
      await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1)));
    }
  }
}
```

## 🔧 调试工具

### 快速诊断脚本

创建 `scripts/build-portable/diagnose.js`:
```javascript
#!/usr/bin/env node

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
  }

  checkEnvironment() {
    console.log('📋 Environment Check:');
    console.log(`- Node.js: ${process.version}`);
    console.log(`- Platform: ${process.platform}-${process.arch}`);
    console.log(`- Working Directory: ${this.rootDir}`);
    console.log('');
  }

  checkBuildOutput() {
    console.log('📁 Build Output Check:');
    console.log(`- Output Directory: ${fs.existsSync(this.outputDir) ? '✅' : '❌'} ${this.outputDir}`);
    
    if (fs.existsSync(this.outputDir)) {
      const dirs = ['app', 'node', 'scripts', 'config'];
      dirs.forEach(dir => {
        const dirPath = path.join(this.outputDir, dir);
        console.log(`- ${dir}/: ${fs.existsSync(dirPath) ? '✅' : '❌'}`);
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
      } catch (error) {
        console.log('- Version check: ❌', error.message);
      }
    } else {
      console.log('- Node.js executable: ❌');
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
        'build-manifest.json'
      ];
      
      requiredFiles.forEach(file => {
        const filePath = path.join(nextDir, file);
        console.log(`- ${file}: ${fs.existsSync(filePath) ? '✅' : '❌'}`);
      });
    } else {
      console.log('- .next directory: ❌');
    }
  }
}

if (require.main === module) {
  const diagnostic = new BuildDiagnostic();
  diagnostic.diagnose();
}

module.exports = BuildDiagnostic;
```

### 使用方法

```bash
# 运行完整诊断
npm run build:diagnose

# 其他相关命令
npm run build:check     # 检查构建环境
npm run build:complete  # 完成便携包构建
npm run build:test      # 测试构建功能
```

### 诊断工具输出示例

```bash
🔧 KIMBAP Console 便携包诊断工具

📋 Environment Check:
- Node.js: v22.17.0
- Platform: darwin-arm64
- npm: ✅
- tar: ✅
- gzip: ✅

📁 Build Output Check:
- Output Directory: ✅
- app/: (1.3 GB) ✅
- node/: (150.7 MB) ✅
- scripts/: (1 KB) ✅

🔧 Node.js Check:
- Node.js executable: ✅
- Version: v20.11.0
- Executable permissions: ✅

⚡ Next.js Build Check:
- .next directory: ✅
- BUILD_ID: ✅
- routes-manifest.json: ✅

📊 Summary Report:
- Total package size: 1.4 GB
- Status: ✅ All checks passed!
```

## 📝 常见错误速查表

| 错误信息 | 可能原因 | 解决方案 |
|---------|---------|---------|
| `Cannot find module 'autoprefixer'` | devDependencies 被删除 | 构建时保留 devDependencies |
| `buttonVariants is not exported` | 组件导出缺失 | 添加 buttonVariants 导出 |
| `output: export` 冲突 | 静态导出配置 | 禁用静态导出 |
| `truncated gzip input` | 下载文件损坏 | 删除文件重新下载 |
| `node: No such file or directory` | Node.js 路径错误 | 检查解压路径 |
| `Could not find a production build` | BUILD_ID 缺失 | 创建 BUILD_ID 文件 |
| `routes-manifest.json` 缺失 | 构建文件不完整 | 复制完整 .next 目录 |

## 🎯 预防措施

1. **构建前检查**: 使用 `npm run build:check` 验证环境
2. **分步骤构建**: 使用 `npm run build:complete` 完成剩余步骤
3. **文件验证**: 每个步骤后验证关键文件存在
4. **日志记录**: 保存详细的构建日志用于调试
5. **自动化测试**: 构建完成后自动测试启动和访问

---

**更新日期**: 2025-08-25  
**适用版本**: KIMBAP Console v1.0.0+  
**维护者**: Claude Code