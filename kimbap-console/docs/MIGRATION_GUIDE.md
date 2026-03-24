# 样式系统迁移指南

## 📋 迁移清单

### 阶段1: 准备工作
- [x] 创建 `/lib/styles.ts`
- [x] 创建公共组件 `/components/common/`
- [x] 设置 VS Code 代码片段

### 阶段2: 逐步迁移

#### 优先级1: 高频使用的页面
- [ ] `/dashboard/members/page.tsx`
- [ ] `/dashboard/tool-configure/page.tsx`
- [ ] `/dashboard/server-control/page.tsx`
- [ ] `/dashboard/billing/page.tsx`

#### 优先级2: 对话框组件
- [ ] `components/edit-token-dialog.tsx`
- [ ] `components/destroy-token-dialog.tsx`
- [ ] `components/backup-dialog.tsx`
- [ ] `components/personal-settings-dialog.tsx`

#### 优先级3: 其他页面
- [ ] `/dashboard/logs/page.tsx`
- [ ] `/dashboard/network-access/page.tsx`
- [ ] `/dashboard/usage/page.tsx`

## 🔄 迁移步骤

### 1. 页面标题迁移
```tsx
// Before
<div className="space-y-4">
  <div>
    <h1 className="text-xl font-bold">Title</h1>
    <p className="text-sm text-muted-foreground">Description</p>
  </div>
</div>

// After
import { PageHeader } from '@/components/common/page-header'

<PageHeader
  title="Title"
  description="Description"
/>
```

### 2. 卡片头部迁移
```tsx
// Before
<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
  <CardTitle className="text-lg flex items-center gap-2">
    <Icon className="h-5 w-5" />
    Title
  </CardTitle>
  <Button>Action</Button>
</CardHeader>

// After
import { cardStyles } from '@/lib/styles'

<CardHeader className={cardStyles.header.withAction}>
  <CardTitle className={cardStyles.title.withIcon}>
    <Icon className="h-5 w-5" />
    Title
  </CardTitle>
  <Button>Action</Button>
</CardHeader>
```

### 3. 表单迁移
```tsx
// Before
<div className="space-y-2">
  <Label htmlFor="name">Name <span className="text-red-500">*</span></Label>
  <Input id="name" />
  <p className="text-sm text-muted-foreground">Enter your name</p>
</div>

// After
import { FormField } from '@/components/common/form-field'

<FormField
  label="Name"
  htmlFor="name"
  required
  helpText="Enter your name"
>
  <Input id="name" />
</FormField>
```

### 4. 状态卡片迁移
```tsx
// Before
<Card className="border-amber-200 bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-950/20 dark:to-orange-950/20">
  <CardHeader>
    <CardTitle className="flex items-center gap-2 text-amber-900 dark:text-amber-100">
      <AlertTriangle className="h-5 w-5" />
      Warning
    </CardTitle>
    <CardDescription className="text-amber-700 dark:text-amber-200">
      This is a warning message
    </CardDescription>
  </CardHeader>
</Card>

// After
import { AlertCard } from '@/components/common/alert-card'

<AlertCard
  variant="warning"
  title="Warning"
  description="This is a warning message"
/>
```

## 🛠️ 迁移工具

### 自动替换脚本
```bash
# 替换常见的页面标题模式
find . -name "*.tsx" -exec sed -i '' 's/className="text-xl font-bold"/className={pageStyles.header.title}/g' {} \;

# 替换描述文本
find . -name "*.tsx" -exec sed -i '' 's/className="text-sm text-muted-foreground"/className={pageStyles.header.description}/g' {} \;
```

### VS Code 批量替换
使用正则表达式在 VS Code 中批量替换：

1. **页面容器**
   - 查找: `className="space-y-4"`
   - 替换: `className={pageStyles.container}`

2. **卡片标题**
   - 查找: `className="text-lg flex items-center gap-2"`
   - 替换: `className={cardStyles.title.withIcon}`

3. **表单字段**
   - 查找: `className="space-y-2"`
   - 替换: `className={formStyles.field}`

## 📊 迁移进度跟踪

### 完成情况统计
```tsx
// 可以在代码中添加注释来跟踪迁移进度
// TODO: Migrate to styles.ts - pageStyles.header.title
// MIGRATED: Using FormField component
// PARTIAL: Header migrated, form pending
```

### 迁移验证清单
每个页面迁移完成后检查：
- [ ] 导入了必要的样式常量
- [ ] 使用了公共组件（如果适用）
- [ ] 删除了重复的 className
- [ ] 功能测试通过
- [ ] 样式保持一致

## 🚀 迁移后的好处

### 代码质量提升
- ✅ 减少30-40%的样式代码
- ✅ 提高代码可读性
- ✅ 统一样式规范
- ✅ TypeScript 类型安全

### 开发效率
- ✅ VS Code 自动补全
- ✅ 代码片段快速生成
- ✅ 减少样式调试时间
- ✅ 新页面快速搭建

### 维护性
- ✅ 样式统一修改
- ✅ 主题切换更容易
- ✅ 组件复用率高
- ✅ 设计系统标准化

## 🔧 常见问题

### Q: 如何处理特殊样式需求？
A: 使用 `cx()` 函数组合样式：
```tsx
import { cx, cardStyles } from '@/lib/styles'

<Card className={cx(
  cardStyles.header.withAction,
  'border-2', // 特殊样式
  isActive && 'bg-blue-50' // 条件样式
)}>
```

### Q: 是否需要立即迁移所有页面？
A: 不需要。建议分阶段迁移：
1. 先迁移高频使用的页面
2. 然后迁移公共组件
3. 最后迁移其他页面

### Q: 如何确保迁移后样式一致？
A: 迁移前后对比截图，使用浏览器开发工具检查样式是否一致。

### Q: 团队如何统一使用新的样式系统？
A: 
1. 团队培训文档和示例
2. Code Review 检查
3. ESLint 规则约束
4. VS Code 代码片段共享

## 📚 相关文档
- [公共样式规范文档](./COMMON_STYLES.md)
- [使用示例文档](./STYLES_USAGE_EXAMPLE.md)