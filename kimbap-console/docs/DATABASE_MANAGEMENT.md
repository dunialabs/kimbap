# 数据库管理指南

## 概述

本项目使用 Prisma ORM 进行数据库管理，支持自动迁移和版本控制。系统可以自动处理数据库初始化和升级，确保数据安全。

## 核心概念

### 1. Prisma Schema (prisma/schema.prisma)
- 定义数据库表结构的单一真实来源
- 修改此文件来更新数据库结构

### 2. 迁移 (Migrations)
- 位于 `prisma/migrations/` 目录
- 每次结构变更都会生成一个新的迁移文件
- 迁移是增量的，保护现有数据

## 快速开始

### 首次启动
```bash
# 自动初始化数据库并启动开发环境
npm run dev

# 或者仅初始化数据库
npm run db:init
```

### 日常开发
```bash
# 启动开发环境（自动检查并应用迁移）
npm run dev

# 查看数据库内容
npm run db:studio
```

## 数据库操作命令

### 基础命令
| 命令 | 说明 |
|------|------|
| `npm run db:init` | 初始化数据库（检查并应用迁移） |
| `npm run db:start` | 启动 PostgreSQL 容器 |
| `npm run db:stop` | 停止数据库容器 |
| `npm run db:studio` | 打开 Prisma Studio 查看数据 |
| `npm run db:logs` | 查看数据库日志 |

### 迁移命令
| 命令 | 说明 | 使用场景 |
|------|------|----------|
| `npm run db:migrate` | 创建新迁移 | 修改 schema 后创建迁移 |
| `npm run db:deploy` | 应用迁移到生产 | 生产环境部署 |
| `npm run db:push` | 直接推送 schema 变更 | 仅用于开发环境快速测试 |
| `npm run db:reset` | 重置数据库（Prisma 方式） | ⚠️ 删除 Prisma 管理的表和数据 |
| `npm run db:reset:complete` | 完全重置数据库 | ⚠️ 删除所有数据，包括非 Prisma 管理的表 |

## 工作流程

### 1. 修改数据库结构

#### 步骤 1: 修改 Schema
编辑 `prisma/schema.prisma` 文件：
```prisma
model User {
  id        String   @id @default(uuid())
  email     String   @unique
  name      String?
  // 添加新字段
  phone     String?  // 新增
  createdAt DateTime @default(now())
}
```

#### 步骤 2: 创建迁移
```bash
# 创建迁移文件
npm run db:migrate -- --name add_phone_to_user

# 这会：
# 1. 生成迁移SQL文件
# 2. 应用到开发数据库
# 3. 更新 Prisma Client
```

#### 步骤 3: 提交代码
```bash
git add prisma/migrations prisma/schema.prisma
git commit -m "feat: add phone field to User model"
```

### 2. 团队协作

当其他开发者拉取代码后：
```bash
# 拉取最新代码
git pull

# 启动开发环境（自动应用新迁移）
npm run dev

# 或手动应用迁移
npm run db:deploy
```

### 3. 生产部署

```bash
# 在生产环境
npm run db:deploy  # 安全地应用所有待处理迁移
npm run build      # 构建应用
npm start          # 启动应用
```

## 迁移策略

### 安全迁移原则
1. **永不删除已应用的迁移文件**
2. **先在开发环境测试**
3. **备份生产数据库**
4. **使用 `db:deploy` 而非 `db:push` 在生产环境**

### 处理冲突
如果多人同时修改 schema：

1. 合并代码冲突
2. 重置本地迁移：
```bash
# 保存本地更改
git stash

# 获取最新代码
git pull

# 重置数据库到最新迁移
npm run db:reset

# 应用本地更改
git stash pop

# 创建新迁移
npm run db:migrate -- --name your_changes
```

## 数据库重置

### 两种重置方式

#### 1. `npm run db:reset` (Prisma 标准重置)
- **功能**: 使用 Prisma 的 `migrate reset` 命令
- **限制**: 只删除 Prisma schema 中定义的表
- **适用场景**: 
  - 开发环境快速重置
  - 只需要重置 Prisma 管理的表
- **注意**: 如果数据库中有不在 Prisma schema 中的表，这些表不会被删除

#### 2. `npm run db:reset:complete` (完全重置) ⭐ 推荐
- **功能**: 完全删除并重建数据库
- **优势**: 
  - 删除所有表（包括非 Prisma 管理的表）
  - 在 Docker 环境中会删除并重建整个数据库
  - 确保完全干净的状态
- **适用场景**: 
  - 需要完全清空数据库
  - 数据库中有遗留表或数据
  - 确保完全重置到初始状态

### 使用示例

```bash
# 标准重置（只删除 Prisma 表）
npm run db:reset

# 完全重置（推荐，删除所有数据）
npm run db:reset:complete
```

## 故障排除

### 问题 1: 迁移失败
```bash
# 查看迁移状态
npx prisma migrate status

# 修复迁移
npx prisma migrate resolve --applied "20240101120000_migration_name"
```

### 问题 2: 数据库连接失败
```bash
# 检查容器状态
docker ps

# 重启数据库
npm run db:restart

# 查看日志
npm run db:logs
```

### 问题 3: Schema 不同步
```bash
# 从数据库拉取当前结构
npx prisma db pull

# 比较差异后决定是否创建迁移
npm run db:migrate
```

## 数据备份

### 备份数据
```bash
# 使用 pg_dump
docker exec kimbap-postgres pg_dump -U kimbap kimbap_db > backup.sql

# 或使用 Docker 卷备份
docker run --rm -v kimbap-console_postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/db-backup.tar.gz /data
```

### 恢复数据
```bash
# 从 SQL 文件恢复
docker exec -i kimbap-postgres psql -U kimbap kimbap_db < backup.sql

# 或从卷备份恢复
docker run --rm -v kimbap-console_postgres_data:/data -v $(pwd):/backup alpine tar xzf /backup/db-backup.tar.gz -C /
```

## 最佳实践

1. **版本控制**: 始终提交 `prisma/migrations` 文件夹
2. **命名规范**: 使用描述性的迁移名称，如 `add_user_phone`
3. **小步迭代**: 频繁创建小的迁移，而不是大的变更
4. **测试迁移**: 在应用到生产前，在测试环境验证
5. **文档记录**: 在迁移中添加注释说明业务逻辑

## 环境变量

```bash
# .env 文件
DATABASE_URL="postgresql://kimbap:kimbap123@localhost:5432/kimbap_db"

# 生产环境可能不同
DATABASE_URL="postgresql://user:password@production-host:5432/production_db"
```

## 注意事项

⚠️ **警告**：
- `db:reset` 会删除所有数据
- `db:push` 只应在开发环境使用
- 生产环境始终使用 `db:deploy`
- 定期备份生产数据库

## 相关资源

- [Prisma 文档](https://www.prisma.io/docs)
- [PostgreSQL 文档](https://www.postgresql.org/docs/)
- [Docker Compose 文档](https://docs.docker.com/compose/)