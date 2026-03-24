# 📦 KIMBAP Console 便携包构建指南

本指南将帮助您构建 KIMBAP Console 的便携版本，让用户可以在任何电脑上运行应用而无需安装 Node.js、Docker 等依赖。

## 🎯 快速开始

### 1. 检查系统要求
```bash
npm run build:check
```

### 2. 测试构建功能
```bash
npm run build:test
```

### 3. 构建便携包
```bash
npm run build:portable
```

## 📋 详细步骤

### 步骤 1: 准备环境

确保您的开发环境满足以下要求：
- Node.js 18+ 
- 网络连接（下载依赖）
- 1GB+ 磁盘空间

```bash
# 检查环境
node --version
npm --version
```

### 步骤 2: 项目构建

```bash
# 安装依赖
npm install

# 构建 Next.js 应用
npm run build

# 生成 Prisma 客户端
npm run db:generate
```

### 步骤 3: 下载便携版依赖

```bash
# 下载 Node.js 和 PostgreSQL
npm run download:deps

# 查看下载状态
node scripts/build-portable/download-dependencies.js --list
```

### 步骤 4: 构建便携包

```bash
# 执行完整构建
npm run build:portable
```

构建完成后，文件将保存在 `dist/kimbap-console-{platform}/` 目录中。

## 📁 输出文件说明

```
dist/kimbap-console-{platform}/
├── app/                    # Next.js 应用（~50MB）
│   ├── .next/             # 构建输出
│   ├── public/            # 静态资源  
│   ├── prisma/            # 数据库模式
│   ├── node_modules/      # 依赖包
│   ├── package.json
│   └── .env.local         # 生产环境配置
├── node/                   # Node.js 运行时（~60MB）
│   ├── bin/node           # (Unix) 或 node.exe (Windows)
│   └── ...
├── postgresql/             # PostgreSQL 数据库（~80MB）
│   ├── bin/               # 数据库工具
│   ├── lib/               # 运行库
│   └── share/             # 共享文件
├── scripts/                # 启动脚本
│   ├── start.bat          # Windows 启动
│   └── start.sh           # Mac/Linux 启动
├── config/                 # 配置文件
│   └── config.json        # 应用配置
└── README.txt              # 使用说明
```

## 🚀 测试便携包

### Windows 测试
```cmd
# 进入构建目录
cd dist\kimbap-console-win32

# 运行启动脚本
scripts\start.bat
```

### Mac/Linux 测试
```bash
# 进入构建目录
cd dist/kimbap-console-darwin  # 或 kimbap-console-linux

# 添加执行权限
chmod +x scripts/start.sh

# 运行启动脚本
./scripts/start.sh
```

应用将在 http://localhost:3000 启动。

## ⚙️ 自定义配置

### 修改应用端口
编辑 `dist/kimbap-console-{platform}/config/config.json`:

```json
{
  "app": {
    "port": 8080,
    "host": "localhost"
  }
}
```

### 数据库配置
```json
{
  "database": {
    "host": "localhost", 
    "port": 5432,
    "database": "kimbap_db",
    "username": "kimbap",
    "password": "your-secure-password"
  }
}
```

### 启动脚本自定义
可以直接编辑 `scripts/start.bat` 或 `scripts/start.sh` 来：
- 修改数据库参数
- 添加环境变量
- 自定义启动流程

## 📦 分发准备

### 1. 压缩打包
```bash
# Windows
cd dist
7z a kimbap-console-windows.zip kimbap-console-win32/

# Mac/Linux  
cd dist
tar -czf kimbap-console-mac.tar.gz kimbap-console-darwin/
tar -czf kimbap-console-linux.tar.gz kimbap-console-linux/
```

### 2. 创建安装说明

为最终用户创建简单的安装文档：

```markdown
# KIMBAP Console 安装说明

1. 解压文件到任意目录
2. Windows: 双击 scripts/start.bat
3. Mac/Linux: 运行 ./scripts/start.sh  
4. 浏览器打开 http://localhost:3000
5. 首次运行需要 3-5 分钟初始化数据库
```

### 3. 版本管理
- 在文件名中包含版本号：`kimbap-console-v1.0.0-windows.zip`
- 在 config.json 中记录构建信息
- 保持构建日志以便故障排除

## 🔧 故障排除

### 快速诊断

使用内置的诊断工具快速检查构建状态：

```bash
# 运行完整诊断
npm run build:diagnose
```

这将检查：
- 构建环境和依赖
- 输出文件的完整性
- Node.js 可执行文件
- Next.js 构建文件
- 启动脚本配置

### 常见问题

**问题**: `Next.js build failed`
```bash
# 解决方案
npm run build:check  # 检查环境
npm ci                # 重新安装依赖
npm run build        # 重新构建
```

**问题**: `Download timeout`
```bash
# 解决方案
npm run download:deps  # 重试下载
# 或手动下载到 temp-downloads/ 目录
```

**问题**: `Permission denied` (Mac/Linux)
```bash
# 解决方案
chmod +x scripts/build-portable/*.js
sudo chown -R $USER:$GROUP dist/
```

**问题**: 构建失败或文件缺失
```bash
# 使用完成构建脚本
npm run build:complete  # 完成剩余构建步骤
npm run build:diagnose  # 检查构建结果
```

### 详细故障排除

如果遇到复杂问题，请参考详细的故障排除指南：
- [便携包构建故障排除指南](./PORTABLE_BUILD_TROUBLESHOOTING.md)

### 运行时问题

**问题**: 端口 3000 被占用
- 修改 `config/config.json` 中的端口
- 或关闭占用端口的应用

**问题**: 数据库启动失败
- 检查磁盘空间是否充足
- 查看 `logs/postgresql.log` 错误日志
- 确认端口 5432 未被占用

**问题**: 应用无法访问
- 检查防火墙设置
- 确认 Node.js 进程正在运行
- 查看 `logs/` 目录中的日志文件

## 🔄 持续集成

### GitHub Actions 自动构建

创建 `.github/workflows/build-portable.yml`:

```yaml
name: Build Portable

on:
  release:
    types: [created]
  workflow_dispatch:

jobs:
  build:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        include:
          - os: windows-latest
            platform: win32
          - os: macos-latest  
            platform: darwin
          - os: ubuntu-latest
            platform: linux

    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '20'
          cache: 'npm'
          
      - name: Install dependencies
        run: npm ci
        
      - name: Build application
        run: npm run build
        
      - name: Build portable package
        run: npm run build:portable
        
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: kimbap-console-${{ matrix.platform }}
          path: dist/
```

## 📊 性能优化

### 减小包体积
1. **移除开发依赖**：只打包 `dependencies`
2. **压缩可执行文件**：使用 UPX 压缩
3. **选择性打包**：只包含必要的 PostgreSQL 组件
4. **静态资源优化**：压缩图片和 CSS

### 提升启动速度
1. **预初始化数据库**：提供预配置的数据库模板
2. **并行启动**：数据库和应用并行初始化
3. **缓存依赖**：预编译 Prisma 客户端

## 🛡️ 安全注意事项

1. **默认密码**：首次运行时生成随机数据库密码
2. **防火墙**：默认只监听 localhost
3. **文件权限**：确保适当的文件权限设置
4. **数据加密**：敏感数据加密存储

## 📈 版本发布

### 发布检查清单
- [ ] 构建测试通过
- [ ] 多平台兼容性验证
- [ ] 安全扫描完成
- [ ] 文档更新
- [ ] 版本号更新
- [ ] 发布说明准备

### 发布流程
1. 创建 GitHub Release
2. 自动触发构建流程
3. 上传构建产物
4. 发布更新通知

---

需要帮助？请查看：
- [故障排除指南](./DEPLOYMENT_GUIDE.md#故障排除)
- [GitHub Issues](https://github.com/your-repo/issues)
- [技术文档](./README.md)