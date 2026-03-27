# 定时任务目录

这个目录包含所有的定时任务脚本，完全独立于Next.js应用。

## 文件说明

- `log-sync.js` - 日志同步定时任务
- `start.js` - 启动所有定时任务的脚本

## 使用方法

### 启动所有定时任务
```bash
node jobs/start.js
```

### 单独运行日志同步
```bash
node jobs/log-sync.js
```

### 测试单次同步
```bash
node -e "require('./jobs/log-sync.js').syncLogs()"
```

## 环境变量配置

```bash
LOG_SYNC_ENABLED=true              # 启用日志同步 (默认: true)
LOG_SYNC_INTERVAL_MINUTES=2        # 同步间隔分钟数 (默认: 2)
MAX_LOGS_PER_REQUEST=1000          # 每次最大获取日志数 (默认: 1000)
PROXY_URL=http://localhost:8000    # Proxy服务器地址 (默认: http://localhost:8000)
```

## 预期输出

```bash
[LogSync] 启动日志同步 - 间隔2分钟
[LogSync] 2025/10/23 15:00:00 - 开始同步日志
[LogSync] 无新日志，同步完成
[LogSync] 定时同步执行
[LogSync] 2025/10/23 15:02:00 - 开始同步日志
[LogSync] 同步完成，新增 5 条日志
```