# 🚀 KIMBAP Console 便携包构建工具

这个工具可以将 KIMBAP Console 打包成一个可以在任何电脑上独立运行的便携包，无需安装 Node.js、Docker 或 PostgreSQL。

## 📋 构建要求

### 系统要求
- **Node.js 18+** (仅构建时需要)
- **网络连接** (下载依赖时需要)
- **磁盘空间**: 至少 1GB 可用空间

### 支持的平台
- ✅ Windows 10/11 (x64)
- ✅ macOS 10.14+ (x64/ARM64) 
- ✅ Ubuntu 18.04+ (x64)

## 🛠️ 使用方法

### 1. 快速构建（推荐）
```bash
# 一键构建便携包
npm run build:portable
```

### 2. 分步构建
```bash
# 步骤 1: 下载依赖（Node.js + PostgreSQL）
npm run download:deps

# 步骤 2: 构建便携包
npm run build:portable
```

### 3. 手动构建
```bash
# 下载依赖
node scripts/build-portable/download-dependencies.js

# 构建便携包
node scripts/build-portable/build-portable.js

# 查看已下载的文件
node scripts/build-portable/download-dependencies.js --list
```

## 📦 输出结构

构建完成后，会在 `dist/` 目录下生成如下结构：

```
dist/kimbap-console-{platform}/
├── app/                    # Next.js 应用
│   ├── .next/             # 构建输出
│   ├── public/            # 静态资源
│   ├── prisma/            # 数据库 schema
│   ├── node_modules/      # 依赖包
│   ├── package.json       # 应用配置
│   └── .env.local         # 环境变量
├── node/                   # Node.js 运行时
│   ├── bin/node           # Node.js 可执行文件
│   └── ...
├── postgresql/             # PostgreSQL 数据库
│   ├── bin/               # 数据库可执行文件
│   ├── lib/               # 运行时库
│   └── data/              # 数据目录（运行时创建）
├── scripts/                # 启动脚本
│   ├── start.bat          # Windows 启动脚本
│   └── start.sh           # Mac/Linux 启动脚本
├── config/                 # 配置文件
│   └── config.json        # 应用配置
├── logs/                   # 日志目录（运行时创建）
└── README.txt              # 用户使用说明
```

## 🚀 部署使用

### Windows
1. 将构建的文件夹复制到目标电脑
2. 双击 `scripts/start.bat`
3. 等待初始化完成
4. 浏览器打开 http://localhost:3000

### Mac/Linux
1. 将构建的文件夹复制到目标电脑
2. 打开终端，进入应用目录
3. 运行 `chmod +x scripts/start.sh` (首次需要)
4. 运行 `./scripts/start.sh`
5. 浏览器打开 http://localhost:3000

## 🔧 高级配置

### 自定义端口
编辑 `config/config.json`：
```json
{
  "app": {
    "port": 8080
  }
}
```

### 数据库配置
编辑 `config/config.json`：
```json
{
  "database": {
    "host": "localhost",
    "port": 5432,
    "database": "kimbap_db",
    "username": "kimbap",
    "password": "自定义密码"
  }
}
```

## 📊 包大小

| 平台 | 压缩前 | 压缩后 |
|------|--------|--------|
| Windows | ~200MB | ~80MB |
| macOS | ~180MB | ~70MB |
| Linux | ~170MB | ~65MB |

## 🐛 故障排除

### 1. 构建失败
```bash
# 清理并重新构建
rm -rf dist/ temp-*
npm run build:portable
```

### 2. 下载超时
```bash
# 手动下载依赖
npm run download:deps

# 检查下载状态
node scripts/build-portable/download-dependencies.js --list
```

### 3. 权限问题 (Mac/Linux)
```bash
# 添加执行权限
chmod +x scripts/build-portable/*.js
```

### 4. 端口被占用
- 修改 `config/config.json` 中的端口号
- 或关闭占用 3000 端口的其他应用

## 🔍 调试模式

```bash
# 启用详细日志
DEBUG=* npm run build:portable

# 保留临时文件（调试用）
KEEP_TEMP=true npm run build:portable
```

## 📝 自定义构建

### 修改版本号
编辑 `scripts/build-portable/download-dependencies.js`：
```javascript
this.nodeVersion = '20.11.0';      // Node.js 版本
this.postgresVersion = '16.1';      // PostgreSQL 版本
```

### 添加自定义文件
编辑 `scripts/build-portable/build-portable.js` 的 `copyAppFiles` 方法。

### 自定义启动脚本
修改 `createWindowsScript()` 或 `createUnixScript()` 方法。

## 🤝 贡献

如需改进构建工具或添加新平台支持，请：

1. Fork 项目
2. 创建功能分支
3. 提交 Pull Request

## 📄 许可证

与主项目保持一致的许可证。

---

## 🔗 相关链接

- [部署指南](../../docs/DEPLOYMENT_GUIDE.md)
- [项目主页](../../README.md)
- [问题反馈](https://github.com/your-repo/issues)

**注意**: 这是实验性功能，建议先在测试环境中验证后再用于生产环境。