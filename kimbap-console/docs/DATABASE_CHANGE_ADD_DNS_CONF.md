# 数据库变更 - 新增 dns_conf 表

## 变更内容

新增了 `dns_conf` 表用于管理 DNS 配置。

### 表结构
```sql
CREATE TABLE dns_conf (
  id SERIAL PRIMARY KEY,
  subdomain VARCHAR(128) DEFAULT '' NOT NULL,
  type INT DEFAULT 0 NOT NULL,
  public_ip VARCHAR(128) DEFAULT '' NOT NULL,
  addtime INT DEFAULT 0 NOT NULL,
  update_time INT DEFAULT 0 NOT NULL
);
```

### Prisma Schema
```prisma
model DnsConf {
  id         Int    @id @default(autoincrement())
  subdomain  String @default("") @map("subdomain") @db.VarChar(128)
  type       Int    @default(0) @map("type") // 0: A record, 1: CNAME, etc.
  publicIp   String @default("") @map("public_ip") @db.VarChar(128)
  addtime    Int    @default(0) @map("addtime") // Creation timestamp
  updateTime Int    @default(0) @map("update_time") // Last update timestamp

  @@map("dns_conf")
}
```

## 字段说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `id` | INT | auto | 主键，自增 |
| `subdomain` | VARCHAR(128) | "" | 子域名 |
| `type` | INT | 0 | DNS记录类型（0: A记录, 1: CNAME等） |
| `public_ip` | VARCHAR(128) | "" | 公网IP地址 |
| `addtime` | INT | 0 | 创建时间（Unix时间戳） |
| `update_time` | INT | 0 | 最后更新时间（Unix时间戳） |

## 迁移信息

- **迁移文件**: 
  - `20241201120000_fix_dns_conf_add_id_remove_proxy_id` - 创建基础表
  - `20241201130000_add_time_fields_to_dns_conf` - 添加时间字段
- **变更类型**: 新增表
- **数据影响**: 无，全新表

## 团队升级步骤

### 其他团队成员操作

当拉取代码后，只需运行：
```bash
# 拉取最新代码
git pull

# 启动开发环境（自动应用迁移）
npm run dev
```

**自动执行的操作**：
1. 检测到新迁移文件
2. 创建 `dns_conf` 表
3. 更新 Prisma Client
4. 启动应用

### 手动执行（可选）
```bash
# 查看迁移状态
npx prisma migrate status

# 手动应用迁移
npx prisma migrate deploy

# 重新生成客户端
npx prisma generate
```

## 使用示例

### 创建 DNS 配置
```typescript
const now = Math.floor(Date.now() / 1000); // Unix timestamp
const dnsRecord = await prisma.dnsConf.create({
  data: {
    subdomain: "api",
    type: 0, // A记录
    publicIp: "192.168.1.100",
    addtime: now,
    updateTime: now
  }
});
```

### 查询 DNS 配置
```typescript
// 获取所有 DNS 配置
const allDnsConfigs = await prisma.dnsConf.findMany();

// 根据子域名查找
const apiDnsConfig = await prisma.dnsConf.findFirst({
  where: {
    subdomain: "api"
  }
});

// 根据类型查找
const aRecords = await prisma.dnsConf.findMany({
  where: {
    type: 0 // A记录
  }
});

// 根据创建时间范围查找
const recentRecords = await prisma.dnsConf.findMany({
  where: {
    addtime: {
      gte: Math.floor(Date.now() / 1000) - 86400 // 最近24小时
    }
  },
  orderBy: {
    addtime: 'desc'
  }
});

// 查找最近更新的记录
const recentlyUpdated = await prisma.dnsConf.findMany({
  where: {
    updateTime: {
      gt: 0 // 已更新过的记录
    }
  },
  orderBy: {
    updateTime: 'desc'
  },
  take: 10 // 最近10条
});
```

### 更新 DNS 配置
```typescript
const updatedDnsConfig = await prisma.dnsConf.update({
  where: { id: 1 },
  data: {
    publicIp: "192.168.1.200",
    updateTime: Math.floor(Date.now() / 1000) // 更新时间戳
  }
});
```

### 删除 DNS 配置
```typescript
await prisma.dnsConf.delete({
  where: { id: 1 }
});
```

## DNS 记录类型定义

| 值 | 类型 | 说明 |
|----|------|------|
| 0 | A | IPv4 地址记录 |
| 1 | AAAA | IPv6 地址记录 |  
| 2 | CNAME | 规范名称记录 |
| 3 | MX | 邮件交换记录 |
| 4 | TXT | 文本记录 |

## 验证方式

```bash
# 检查表是否创建成功
npx prisma studio
```

或者用 SQL 查询：
```sql
SELECT * FROM dns_conf;
```

期望结果：
- ✅ `dns_conf` 表存在
- ✅ 表结构符合预期
- ✅ 可以正常增删改查

## 总结

这是一个**新功能添加**：
- ✅ 新增 DNS 配置管理功能
- ✅ 自动化迁移，无需手动操作
- ✅ 符合数据库设计规范
- ✅ 团队协作友好

团队成员只需要拉取代码并运行 `npm run dev` 即可完成升级。