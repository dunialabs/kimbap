# 数据库更新示例 - 添加 start_port 和 dns_conf 表

## 本次更新内容

1. **Proxy 表新增字段**：
   - `start_port` (Int) - 代理启动端口，默认值 3002

2. **新增 dns_conf 表**：
   - `id` - 主键
   - `subdomain` - 子域名
   - `type` - DNS记录类型 (0: A记录, 1: CNAME等)
   - `public_ip` - 公网IP地址
   - `proxy_id` - 关联的代理ID

## 更新步骤

### 1. 我（开发者）的操作

#### Step 1: 更新 Prisma Schema
编辑 `prisma/schema.prisma`，添加新字段和表：

```prisma
model Proxy {
  id        Int     @id @default(autoincrement()) @map("id")
  name      String  @map("name") @db.VarChar(128)
  proxyKey  String  @default("") @map("proxy_key") @db.VarChar(255)
  addtime   Int     @map("addtime")
  startPort Int     @default(3002) @map("start_port") // 新增字段
  
  @@map("proxy")
}

model DnsConf {
  id        Int    @id @default(autoincrement())
  subdomain String @default("") @map("subdomain") @db.VarChar(128)
  type      Int    @default(0) @map("type")
  publicIp  String @default("") @map("public_ip") @db.VarChar(128)
  proxyId   Int    @default(0) @map("proxy_id")
  
  @@map("dns_conf")
}
```

#### Step 2: 创建迁移基线（首次）
由于这是第一次使用迁移系统，需要创建基线：

```bash
# 1. 从现有数据库拉取结构
npx prisma db pull

# 2. 创建基线迁移
mkdir -p prisma/migrations/20241201000000_initial_baseline
npx prisma migrate diff --from-empty --to-schema-datamodel prisma/schema.prisma --script > prisma/migrations/20241201000000_initial_baseline/migration.sql

# 3. 标记基线为已应用
npx prisma migrate resolve --applied 20241201000000_initial_baseline

# 4. 生成 Prisma Client
npx prisma generate
```

#### Step 3: 提交代码
```bash
git add prisma/
git commit -m "feat: add start_port to proxy table and create dns_conf table"
git push
```

### 2. 其他开发者的操作（自动完成）

当其他开发者拉取代码后，只需运行：

```bash
# 拉取最新代码
git pull

# 启动开发环境（自动应用迁移）
npm run dev
```

**自动流程**：
1. `npm run dev` 调用 `scripts/db-init.js`
2. 脚本检测到新的迁移文件
3. 自动应用迁移（不影响现有数据）
4. 更新 Prisma Client
5. 启动应用

### 3. 手动操作（可选）

如果需要手动管理：

```bash
# 查看迁移状态
npx prisma migrate status

# 应用迁移
npx prisma migrate deploy

# 生成客户端
npx prisma generate
```

## 工作原理

### 迁移系统如何保护数据

1. **增量更新**：迁移只记录变更，不是完整的表结构
2. **版本控制**：每个迁移都有时间戳和名称
3. **事务保护**：迁移在事务中执行，失败会回滚
4. **历史记录**：`_prisma_migrations` 表记录所有已应用的迁移

### 数据安全保证

- ✅ **现有数据保留**：新增字段使用默认值，不影响现有记录
- ✅ **向后兼容**：旧代码可以继续运行（忽略新字段）
- ✅ **可回滚**：如果出现问题，可以回滚到之前版本

## 后续更新流程

当需要再次更新数据库时：

### 示例：添加新字段
```bash
# 1. 修改 schema.prisma
# 2. 创建迁移
npm run db:migrate -- --name add_new_field

# 3. 提交代码
git add prisma/
git commit -m "feat: add new field"
git push
```

### 示例：修改字段类型
```bash
# 1. 修改 schema.prisma
# 2. 创建迁移（会提示数据转换）
npm run db:migrate -- --name change_field_type

# 3. 检查生成的 SQL
# 4. 提交代码
```

## 常见问题

### Q: 如果数据库已经有这些字段怎么办？
A: Prisma 会检测到没有差异，不会重复创建。

### Q: 如果迁移失败怎么办？
A: 
1. 查看错误信息
2. 修复 schema 或手动调整数据库
3. 使用 `prisma migrate resolve` 标记解决

### Q: 如何查看将要执行的 SQL？
A: 查看 `prisma/migrations/*/migration.sql` 文件

### Q: 生产环境如何部署？
A: 
```bash
# 生产环境只应用迁移，不创建新迁移
npm run db:deploy
# 或
npx prisma migrate deploy
```

## 总结

通过 Prisma 迁移系统：
1. **开发者**：修改 schema → 创建迁移 → 提交代码
2. **其他人**：拉取代码 → 运行 `npm run dev` → 自动完成
3. **数据安全**：永远不会丢失，只会增量更新

这个流程确保了团队协作的顺畅和数据的安全。