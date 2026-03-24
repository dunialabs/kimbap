# 📦 KIMBAP Console 独立部署方案文档

## 一、部署架构概述

将 Next.js + PostgreSQL 应用打包成可在任意电脑上独立运行的程序包，用户无需预装 Node.js、Docker 等环境。

## 二、打包方案选择

### **方案 A：All-in-One 便携包（推荐）**

**包含内容：**
- Next.js 生产构建文件
- Node.js 运行时（便携版）
- PostgreSQL 便携版
- 启动脚本

**目录结构：**
```
kimbap-console-portable/
├── app/                    # Next.js 构建输出
│   ├── .next/
│   ├── public/
│   ├── prisma/
│   └── package.json
├── postgresql/             # PostgreSQL 便携版
│   ├── bin/
│   ├── data/
│   └── lib/
├── node/                   # Node.js 便携版
│   └── node.exe (或 node)
├── scripts/
│   ├── start.bat          # Windows 启动脚本
│   ├── start.sh           # Mac/Linux 启动脚本
│   └── setup.js           # 初始化脚本
└── README.txt             # 使用说明
```

**打包步骤：**

1. **构建 Next.js 应用**
```bash
npm run build
npm ci --production  # 只安装生产依赖
```

2. **下载便携版 PostgreSQL**
- Windows: PostgreSQL Portable (约 150MB)
- Mac: Postgres.app
- Linux: PostgreSQL 二进制包

3. **下载 Node.js 便携版**
- 从 nodejs.org 下载对应平台的二进制文件

4. **创建启动脚本**

Windows `start.bat`:
```batch
@echo off
echo Starting KIMBAP Console...

REM 设置环境变量
set PATH=%~dp0postgresql\bin;%~dp0node;%PATH%
set PGDATA=%~dp0postgresql\data
set DATABASE_URL=postgresql://kimbap:kimbap123@localhost:5432/kimbap_db

REM 初始化数据库（首次运行）
if not exist "%PGDATA%" (
    echo Initializing database...
    initdb -D "%PGDATA%" -U kimbap
    pg_ctl -D "%PGDATA%" start
    createdb -U kimbap kimbap_db
    cd app && ..\node\node.exe node_modules\.bin\prisma migrate deploy
) else (
    REM 启动数据库
    pg_ctl -D "%PGDATA%" start
)

REM 启动应用
cd app
..\node\node.exe node_modules\next\dist\bin\next start -p 3000

pause
```

Mac/Linux `start.sh`:
```bash
#!/bin/bash
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

export PATH="$SCRIPT_DIR/postgresql/bin:$SCRIPT_DIR/node:$PATH"
export PGDATA="$SCRIPT_DIR/postgresql/data"
export DATABASE_URL="postgresql://kimbap:kimbap123@localhost:5432/kimbap_db"

# 初始化数据库（首次运行）
if [ ! -d "$PGDATA" ]; then
    echo "Initializing database..."
    initdb -D "$PGDATA" -U kimbap
    pg_ctl -D "$PGDATA" start
    createdb -U kimbap kimbap_db
    cd "$SCRIPT_DIR/app" && ../node/node node_modules/.bin/prisma migrate deploy
else
    # 启动数据库
    pg_ctl -D "$PGDATA" start
fi

# 启动应用
cd "$SCRIPT_DIR/app"
../node/node node_modules/next/dist/bin/next start -p 3000
```

### **方案 B：Docker Desktop 依赖方案**

**前提：** 用户需要安装 Docker Desktop

**打包内容：**
```
kimbap-console-docker/
├── app.tar             # Next.js Docker 镜像
├── postgres.tar        # PostgreSQL Docker 镜像
├── docker-compose.yml
├── .env.production
├── install.bat/.sh     # 安装脚本
└── start.bat/.sh       # 启动脚本
```

**安装脚本：**
```bash
# 加载 Docker 镜像
docker load < app.tar
docker load < postgres.tar

# 启动服务
docker-compose up -d
```

### **方案 C：系统服务安装包**

使用专业打包工具创建安装程序：

**Windows：** 使用 NSIS 或 WiX
- 自动安装 PostgreSQL 服务
- 注册 Windows 服务
- 创建开始菜单快捷方式

**Mac：** 使用 pkgbuild
- 创建 .pkg 安装包
- 自动配置 launchd 服务

**Linux：** 使用 .deb/.rpm
- 系统包管理器安装
- systemd 服务配置

## 三、自动化打包脚本

创建 `build-portable.js`:
```javascript
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

async function buildPortable() {
  const platform = process.platform;
  const outputDir = `dist/kimbap-console-${platform}`;
  
  // 1. 构建 Next.js
  console.log('Building Next.js app...');
  execSync('npm run build', { stdio: 'inherit' });
  
  // 2. 创建目录结构
  fs.mkdirSync(outputDir, { recursive: true });
  
  // 3. 复制应用文件
  copyDir('.next', `${outputDir}/app/.next`);
  copyDir('public', `${outputDir}/app/public`);
  copyDir('prisma', `${outputDir}/app/prisma`);
  
  // 4. 下载 PostgreSQL 便携版
  await downloadPostgreSQL(platform, outputDir);
  
  // 5. 下载 Node.js
  await downloadNode(platform, outputDir);
  
  // 6. 创建启动脚本
  createStartScripts(platform, outputDir);
  
  // 7. 打包成 zip
  createArchive(outputDir);
}

function copyDir(src, dest) {
  // 目录复制逻辑
}

async function downloadPostgreSQL(platform, outputDir) {
  // PostgreSQL 下载逻辑
}

async function downloadNode(platform, outputDir) {
  // Node.js 下载逻辑
}

function createStartScripts(platform, outputDir) {
  // 启动脚本生成逻辑
}

function createArchive(outputDir) {
  // 压缩打包逻辑
}

// 执行构建
buildPortable().catch(console.error);
```

## 四、部署步骤（用户端）

1. **下载压缩包**
   - `kimbap-console-windows.zip`
   - `kimbap-console-mac.tar.gz`
   - `kimbap-console-linux.tar.gz`

2. **解压到任意目录**

3. **运行启动脚本**
   - Windows: 双击 `start.bat`
   - Mac/Linux: 运行 `./start.sh`

4. **访问应用**
   - 浏览器打开 `http://localhost:3000`

## 五、优化建议

### **1. 减小包体积**
- 使用 UPX 压缩可执行文件
- 只包含必要的 PostgreSQL 组件
- 生产构建优化

### **2. 数据持久化**
```javascript
// 数据存储位置
Windows: %APPDATA%/kimbap-console/
Mac: ~/Library/Application Support/kimbap-console/
Linux: ~/.config/kimbap-console/
```

### **3. 端口冲突处理**
```javascript
// 自动检测可用端口
const getPort = require('get-port');
const port = await getPort({ port: [3000, 3001, 3002] });
```

### **4. 自动更新机制**
```javascript
// 检查更新
const checkUpdate = async () => {
  const response = await fetch('https://your-server.com/version');
  const { version } = await response.json();
  // 比较版本并提示更新
};
```

## 六、配置文件示例

**config.json:**
```json
{
  "app": {
    "port": 3000,
    "host": "localhost"
  },
  "database": {
    "host": "localhost",
    "port": 5432,
    "database": "kimbap_db",
    "username": "kimbap",
    "password": "kimbap123"
  },
  "paths": {
    "data": "./data",
    "logs": "./logs"
  }
}
```

## 七、故障排除

**常见问题：**

1. **端口占用**
   - 自动切换备用端口
   - 提示用户关闭占用程序

2. **权限问题**
   - Windows: 请求管理员权限
   - Mac/Linux: 使用用户目录存储数据

3. **防火墙拦截**
   - 添加防火墙规则说明
   - 仅监听 localhost

## 八、安全考虑

1. **数据库密码**
   - 首次运行时生成随机密码
   - 存储在用户配置文件中

2. **网络访问**
   - 默认只允许本地访问
   - 可选配置远程访问

3. **数据加密**
   - 敏感数据加密存储
   - 使用 HTTPS（可选）

## 九、预估包大小

- **Windows**: ~200MB（压缩后 ~80MB）
- **Mac**: ~180MB（压缩后 ~70MB）  
- **Linux**: ~170MB（压缩后 ~65MB）

## 十、GitHub Actions 自动构建

```yaml
name: Build Portable

on:
  release:
    types: [created]

jobs:
  build:
    strategy:
      matrix:
        os: [windows-latest, macos-latest, ubuntu-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v2
      - run: npm ci
      - run: npm run build:portable
      - uses: actions/upload-artifact@v2
        with:
          name: kimbap-console-${{ matrix.os }}
          path: dist/
```

## 十一、实施建议

### **Phase 1: 基础打包**
1. 实现方案 A 的基本版本
2. 支持单个平台（如 Windows）
3. 手动打包流程

### **Phase 2: 自动化**
1. 开发自动化打包脚本
2. 支持多平台构建
3. 集成 CI/CD

### **Phase 3: 优化**
1. 减小包体积
2. 添加自动更新
3. 完善错误处理

### **Phase 4: 专业化**
1. 创建安装程序
2. 数字签名
3. 应用商店发布

这个方案可以让用户下载一个压缩包，解压后直接运行，无需安装任何依赖。所有必要的组件都包含在包内。