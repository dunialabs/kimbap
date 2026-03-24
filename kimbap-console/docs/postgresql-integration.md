# PostgreSQL 集成文档

## 概述

本项目使用 PostgreSQL 作为主数据库，通过 Prisma ORM 提供类型安全的数据库访问。支持本地开发、云端部署和 Serverless 环境。

## 技术栈

- **PostgreSQL 16**: 强大的开源关系型数据库
- **Prisma ORM**: 现代化的 Node.js 和 TypeScript ORM
- **Docker Compose**: 本地开发环境管理
- **TypeScript**: 提供完整的类型支持

## 快速开始

### 1. 启动本地数据库

```bash
# 启动 PostgreSQL 和 Adminer
docker-compose up -d

# 检查服务状态
docker-compose ps
```

数据库连接信息：
- Host: localhost
- Port: 5432
- Database: kimbap_db
- Username: kimbap
- Password: kimbap123

Adminer（数据库管理界面）：http://localhost:8080

### 2. 配置环境变量

创建 `.env.local` 文件：

```bash
DATABASE_URL="postgresql://kimbap:kimbap123@localhost:5432/kimbap_db?schema=public"
```

### 3. 初始化数据库

```bash
# 生成 Prisma Client
npx prisma generate

# 创建数据库表
npx prisma db push

# （可选）打开 Prisma Studio 查看数据
npx prisma studio
```

## 项目结构

```
/prisma
└── schema.prisma          # 数据库模型定义

/lib
├── prisma.ts             # Prisma Client 单例
├── db-service.ts         # 数据库服务层
└── middleware/
    └── prisma-middleware.ts  # API 中间件

/app/api
├── users/route.ts        # 用户管理 API
├── logs/route.ts         # 日志管理 API
└── stats/route.ts        # 统计信息 API
```

## 数据库模型

### User 模型
```prisma
model User {
  id          Int           @id @default(autoincrement())
  email       String        @unique
  name        String
  createdAt   DateTime      @default(now())
  updatedAt   DateTime      @updatedAt
  apiRequests ApiRequest[]
}
```

### Log 模型
```prisma
model Log {
  id        Int      @id @default(autoincrement())
  level     String   // debug, info, warn, error
  message   String
  metadata  Json?
  createdAt DateTime @default(now())
}
```

### ApiRequest 模型
```prisma
model ApiRequest {
  id           Int      @id @default(autoincrement())
  method       String
  url          String
  statusCode   Int?
  responseTime Int?
  userId       Int?
  user         User?    @relation(fields: [userId], references: [id])
  createdAt    DateTime @default(now())
}
```

## API 使用示例

### 在 API 路由中使用数据库

```typescript
import { withDb } from '@/lib/middleware/prisma-middleware'
import { dbService } from '@/lib/db-service'

export const GET = withDb(async (req) => {
  const users = await dbService.users.list(10, 0)
  return NextResponse.json({ data: users })
})
```

### 数据库操作示例

```typescript
// 创建用户
const user = await dbService.users.create('user@example.com', 'John Doe')

// 查询用户
const user = await dbService.users.findById(1)
const user = await dbService.users.findByEmail('user@example.com')

// 更新用户
const updated = await dbService.users.update(1, { name: 'Jane Doe' })

// 删除用户
await dbService.users.delete(1)

// 创建日志
await dbService.logs.create('info', 'User logged in', { userId: 1 })

// 获取 API 统计
const stats = await dbService.apiRequests.getStats()
```

## 部署配置

### 本地开发

使用 Docker Compose：
```bash
docker-compose up -d
```

### Vercel 部署

1. 在 Vercel 创建 PostgreSQL 数据库
2. 复制数据库连接字符串到环境变量
3. 运行数据库迁移：
```bash
npx prisma db push
```

### Supabase 部署

```env
DATABASE_URL="postgresql://postgres:[YOUR-PASSWORD]@db.[YOUR-PROJECT].supabase.co:5432/postgres"
```

### 自托管服务器

1. 安装 PostgreSQL
2. 创建数据库和用户
3. 配置环境变量
4. 运行 `npx prisma db push`

## Prisma 常用命令

```bash
# 生成 Prisma Client
npx prisma generate

# 推送 schema 到数据库（开发环境）
npx prisma db push

# 创建迁移（生产环境）
npx prisma migrate dev --name init

# 应用迁移
npx prisma migrate deploy

# 打开 Prisma Studio
npx prisma studio

# 格式化 schema 文件
npx prisma format
```

## 性能优化

1. **连接池管理**
   - Prisma 自动管理连接池
   - 默认连接数根据环境自动调整

2. **查询优化**
   - 使用 `select` 只查询需要的字段
   - 使用 `include` 进行关联查询
   - 批量操作使用 `createMany`、`updateMany`

3. **索引优化**
   - Email 字段已添加唯一索引
   - 可根据查询模式添加其他索引

## 故障排除

### 连接错误

1. 确保 PostgreSQL 服务正在运行：
```bash
docker-compose ps
```

2. 检查环境变量是否正确：
```bash
echo $DATABASE_URL
```

3. 测试数据库连接：
```bash
npx prisma db pull
```

### 迁移问题

如果遇到迁移冲突：
```bash
# 重置数据库（开发环境）
npx prisma migrate reset

# 或手动解决冲突后
npx prisma migrate resolve
```

### 类型错误

生成最新的 Prisma Client：
```bash
npx prisma generate
```

## 监控和维护

### 查看数据库状态

使用 Adminer：http://localhost:8080

或使用 psql：
```bash
docker exec -it kimbap-postgres psql -U kimbap -d kimbap_db
```

### 备份数据库

```bash
# 备份
docker exec kimbap-postgres pg_dump -U kimbap kimbap_db > backup.sql

# 恢复
docker exec -i kimbap-postgres psql -U kimbap kimbap_db < backup.sql
```

### 日志查看

```bash
# 查看 PostgreSQL 日志
docker logs kimbap-postgres

# 查看应用日志
npm run dev
```

## 安全建议

1. **生产环境**
   - 使用强密码
   - 启用 SSL 连接
   - 限制数据库访问 IP

2. **环境变量**
   - 不要提交 `.env` 文件
   - 使用环境变量管理服务

3. **数据验证**
   - Prisma 提供基本类型验证
   - 应用层进行业务逻辑验证

## 迁移指南

### 从 SQLite 迁移

1. 导出 SQLite 数据
2. 调整数据格式（如日期时间）
3. 导入到 PostgreSQL
4. 验证数据完整性

### 添加新模型

1. 编辑 `prisma/schema.prisma`
2. 运行 `npx prisma generate`
3. 运行 `npx prisma db push`（开发）
4. 或 `npx prisma migrate dev`（生产）