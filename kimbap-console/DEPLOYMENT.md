# KIMBAP Console 部署指南

> KIMBAP MCP (Model Context Protocol) 控制台应用的完整部署解决方案

## 🎯 部署方案概述

KIMBAP Console 提供两种主要部署方案：
- **独立部署包** - 内嵌 PostgreSQL，一键启动，无需依赖
- **Docker 部署** - 容器化部署，适合云端和生产环境

## 🚀 方案一：独立部署包（推荐）

### 特性
✅ 内嵌 PostgreSQL 15 数据库  
✅ 前端代码混淆，后端代码开源  
✅ 跨平台支持（Windows/Mac/Linux）  
✅ 一键启动，无需安装依赖  
✅ 包含完整的 Node.js 运行时  

### 构建命令
```bash
# 构建当前平台
npm run build:complete

# 构建特定平台
npm run build:complete:x64    # x64 架构
npm run build:linux:x64       # Linux x64
npm run build:windows:x64     # Windows x64

# 构建所有平台
npm run build:all
```

### 输出结构
```
dist/kimbap-console-{platform}-{arch}/
├── app/                      # 应用程序（前端已混淆）
│   ├── .next/               # Next.js 前端
│   ├── proxy-server/        # Express 后端
│   └── node_modules/        # 运行时依赖
├── node/                    # Node.js 运行时
├── postgresql/              # PostgreSQL 数据库
│   ├── bin/                # 数据库二进制文件
│   ├── data/               # 数据存储目录
│   └── init-tables.sql     # 表初始化脚本
├── scripts/                 # 启动脚本
│   ├── start.sh            # 主启动脚本（Mac/Linux）
│   └── start.bat           # 主启动脚本（Windows）
└── README.md               # 用户说明
```

### 用户部署步骤

1. **下载并解压**
   ```bash
   # Mac/Linux
   tar -xzf kimbap-console-{platform}-{arch}.tar.gz
   
   # Windows
   # 解压 kimbap-console-win32-x64.zip
   ```

2. **启动应用**
   ```bash
   # Mac/Linux
   cd kimbap-console-{platform}-{arch}
   ./scripts/start.sh
   
   # Windows
   cd kimbap-console-win32-x64
   scripts\start.bat
   ```

3. **访问应用**
   - 前端：http://localhost:3000
   - 后端API：http://localhost:3002

### 智能启动流程
启动脚本会自动：
1. 🔍 检测并启动内嵌 PostgreSQL
2. 📋 创建数据库用户和数据库
3. 🗄️ 初始化数据库表结构
4. 🔧 启动后端 API 服务
5. 🎨 启动前端 Web 界面
6. ✅ 显示访问地址

## 🐳 方案二：Docker 部署

### 特性
✅ 容器化部署，环境一致性  
✅ 支持 Docker Compose 一键启动  
✅ 自动处理数据库依赖  
✅ 支持生产环境配置  
✅ 跨平台镜像支持  

### 快速开始

**方法1：使用 Docker Compose**
```bash
# 启动所有服务
docker compose up -d

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f
```

**方法2：一键部署脚本**
```bash
# 使用环境变量配置
./docker-run-with-env.sh
```

### Docker 管理命令
```bash
# 构建镜像
npm run docker:build

# 启动服务
npm run docker:up

# 停止服务
npm run docker:down

# 查看日志
npm run docker:logs

# 推送到 Docker Hub
npm run docker:push
```

### 环境变量配置
创建 `.env` 文件：
```env
# 数据库配置
DATABASE_URL=postgresql://kimbap:kimbap123@postgres:5432/kimbap_db

# 应用配置
NODE_ENV=production

# 端口配置
PORT=3000
BACKEND_PORT=3002

# MCP 代理配置
PROXY_ADMIN_URL=http://localhost:3002
PROXY_ADMIN_TOKEN=your-admin-token
```

### 服务组件
- **kimbap-console**: 主应用容器（Next.js + Express）
- **postgres**: PostgreSQL 16 数据库
- **cloudflared**: Cloudflare 隧道（可选）
- **adminer**: 数据库管理界面（可选）

### 端口映射
- `3000` → 前端应用
- `3002` → 后端 API
- `5432` → PostgreSQL
- `8080` → Adminer 管理界面

## 🔧 开发者构建

### 环境要求
- Node.js 18+
- Docker (Docker 部署)
- PostgreSQL 15+ (独立部署包构建时需要)

### 构建流程
1. **克隆仓库**
   ```bash
   git clone <repository-url>
   cd kimbap-console
   npm install
   ```

2. **构建后端**
   ```bash
   npm run build:backend
   ```

3. **构建前端**
   ```bash
   npm run build:frontend
   ```

4. **打包部署**
   ```bash
   # 独立部署包
   npm run build:complete
   
   # Docker 镜像
   npm run docker:build
   ```

## 📊 部署对比

| 特性 | 独立部署包 | Docker 部署 |
|------|-----------|------------|
| **安装难度** | ⭐⭐⭐⭐⭐ 极简 | ⭐⭐⭐⭐ 简单 |
| **系统要求** | 无特殊要求 | 需要 Docker |
| **包大小** | ~75MB | ~200MB |
| **启动速度** | 快（10-15秒） | 中等（30-60秒） |
| **跨平台** | 需不同版本 | 统一镜像 |
| **生产部署** | ⭐⭐⭐ 适合 | ⭐⭐⭐⭐⭐ 最佳 |
| **维护更新** | 手动替换 | docker pull |

## 🛡️ 安全特性

### 代码保护
- **前端代码**: 使用 `javascript-obfuscator` 进行混淆
- **后端代码**: 保持开源（`backend-src/` 目录）
- **数据库**: PostgreSQL 15 with SSL 支持

### 访问控制
- JWT 令牌认证
- 角色权限管理（Owner/Admin/Member）
- API 访问令牌系统
- IP 白名单支持

## 🔍 故障排除

### 独立部署包常见问题

**PostgreSQL 启动失败**
```bash
# 检查端口占用
lsof -i :5432

# 清理数据目录
rm -rf postgresql/data
```

**权限问题（Mac/Linux）**
```bash
# 添加执行权限
chmod +x scripts/start.sh
chmod +x postgresql/bin/*
```

**端口冲突**
```bash
# 修改端口（编辑 app/.env.local）
PORT=8080
BACKEND_PORT=8002
```

### Docker 部署常见问题

**容器启动失败**
```bash
# 查看详细日志
docker compose logs --tail=50 kimbap-console

# 重新构建
docker compose build --no-cache
```

**数据库连接失败**
```bash
# 检查数据库状态
docker compose exec postgres pg_isready

# 重启服务
docker compose restart
```

## 📚 相关文档

- [CLAUDE.md](./CLAUDE.md) - 开发环境配置
- [README.md](./README.md) - 项目概述
- [API接口需求文档.md](./API接口需求文档.md) - API 接口文档
- [docs/postgresql-integration.md](./docs/postgresql-integration.md) - 数据库集成

## 🆘 技术支持

如遇到部署问题，请提供以下信息：
- 操作系统版本
- 部署方案（独立/Docker）
- 错误日志
- 环境配置

## 📋 更新日志

### v1.0.0 (当前版本)
- ✅ 支持独立部署包（内嵌 PostgreSQL）
- ✅ 支持 Docker 容器化部署
- ✅ 前端代码混淆保护
- ✅ 跨平台构建支持
- ✅ 智能启动脚本
- ✅ 完整的数据库集成

---

**推荐部署顺序**: 独立部署包 → Docker 部署 → 自定义构建