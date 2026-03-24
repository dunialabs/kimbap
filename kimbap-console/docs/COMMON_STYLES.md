# 公共样式规范文档

## 概述
本文档总结了KIMBAP Console项目中可以抽取的公共样式和组件模式，用于提高代码复用性和维护性。

## 1. 页面布局模式

### 1.1 页面标题模式
```tsx
// 标准页面头部布局
<div className="space-y-4">
  <div>
    <h1 className="text-xl font-bold">{title}</h1>
    <p className="text-sm text-muted-foreground">{description}</p>
  </div>
  {/* 页面内容 */}
</div>
```

**使用场景:**
- `/dashboard/members/page.tsx`
- `/dashboard/billing/page.tsx`
- `/dashboard/network-access/page.tsx`
- `/dashboard/server-control/page.tsx`

### 1.2 卡片头部模式
```tsx
// 标准卡片头部（带操作按钮）
<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
  <div>
    <CardTitle className="text-lg flex items-center gap-2">
      <Icon className="h-5 w-5" />
      {title}
    </CardTitle>
    <CardDescription className="text-sm">{description}</CardDescription>
  </div>
  <Button>{action}</Button>
</CardHeader>
```

## 2. 通用间距规范

### 2.1 垂直间距
- `space-y-1`: 紧凑列表项（如列表内文本）
- `space-y-2`: 表单控件组
- `space-y-3`: 卡片列表
- `space-y-4`: 主要内容区块
- `space-y-6`: 对话框内主要区块

### 2.2 网格布局
- `grid gap-2`: 紧凑网格
- `grid gap-4`: 标准网格
- `grid gap-6`: 宽松网格
- `grid grid-cols-2 gap-4`: 两列表单布局
- `grid md:grid-cols-3 gap-4`: 响应式三列卡片

## 3. 对话框样式规范

### 3.1 标准对话框尺寸
```tsx
// 小型对话框（确认、简单表单）
<DialogContent className="sm:max-w-md">

// 中型对话框（复杂表单）
<DialogContent className="sm:max-w-lg">

// 大型对话框（多步骤、详细配置）
<DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-y-auto">

// 超大型对话框（全功能编辑器）
<DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
```

## 4. 按钮样式规范

### 4.1 按钮变体使用
```tsx
// 主要操作
<Button>Primary Action</Button>

// 次要操作
<Button variant="outline">Secondary Action</Button>

// 危险操作
<Button variant="destructive">Delete</Button>

// 透明背景（工具栏）
<Button variant="outline" className="bg-transparent">

// 小尺寸操作按钮
<Button variant="outline" size="sm" className="text-xs">
```

## 5. 卡片内容布局

### 5.1 列表项布局
```tsx
// 标准列表项（带边框）
<div className="p-4 border rounded-lg">
  <div className="flex items-center justify-between mb-3">
    {/* 主要内容 */}
  </div>
  <div className="flex items-center gap-6 text-xs text-muted-foreground">
    {/* 次要信息 */}
  </div>
</div>
```

### 5.2 统计卡片
```tsx
<Card>
  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
    <CardTitle className="text-sm font-medium">{metric}</CardTitle>
    <Icon className="h-4 w-4 text-muted-foreground" />
  </CardHeader>
  <CardContent>
    <div className="text-2xl font-bold">{value}</div>
    <p className="text-xs text-muted-foreground">{description}</p>
  </CardContent>
</Card>
```

## 6. 表单布局模式

### 6.1 标准表单字段
```tsx
<div className="space-y-2">
  <Label htmlFor="field-id">
    Field Label <span className="text-red-500">*</span>
  </Label>
  <Input id="field-id" placeholder="placeholder" />
  <p className="text-sm text-muted-foreground">Help text</p>
</div>
```

### 6.2 两列表单布局
```tsx
<div className="grid grid-cols-2 gap-4">
  <div className="space-y-2">{/* Field 1 */}</div>
  <div className="space-y-2">{/* Field 2 */}</div>
</div>
```

## 7. 提示和警告样式

### 7.1 信息提示卡片
```tsx
// 警告提示
<Card className="border-amber-200 bg-gradient-to-r from-amber-50 to-orange-50">
  <CardHeader>
    <CardTitle className="flex items-center gap-2 text-amber-900">
      <AlertTriangle className="h-5 w-5" />
      {title}
    </CardTitle>
  </CardHeader>
</Card>

// 信息提示
<Card className="border-blue-200 bg-gradient-to-r from-blue-50 to-indigo-50">
  {/* content */}
</Card>
```

## 8. 工具权限卡片模式

### 8.1 可展开工具卡片
```tsx
<Card className="bg-muted/20">
  <CardHeader className="pb-3">
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        <Tool.icon className="h-5 w-5" />
        <CardTitle className="text-base">{tool.name}</CardTitle>
      </div>
      <div className="flex items-center gap-2">
        <Switch />
        <Button variant="ghost" size="sm" className="h-6 w-6 p-0">
          <ChevronDown className="h-4 w-4" />
        </Button>
      </div>
    </div>
  </CardHeader>
  <Collapsible>
    <CollapsibleContent>
      <CardContent>{/* Sub-functions */}</CardContent>
    </CollapsibleContent>
  </Collapsible>
</Card>
```

## 9. 建议的公共组件

基于以上分析，建议创建以下公共组件：

### 9.1 `PageHeader`
```tsx
interface PageHeaderProps {
  title: string
  description?: string
  action?: React.ReactNode
}
```

### 9.2 `StatCard`
```tsx
interface StatCardProps {
  title: string
  value: string | number
  description?: string
  icon?: React.ComponentType
  trend?: 'up' | 'down' | 'neutral'
}
```

### 9.3 `FormField`
```tsx
interface FormFieldProps {
  label: string
  required?: boolean
  helpText?: string
  error?: string
  children: React.ReactNode
}
```

### 9.4 `ListItem`
```tsx
interface ListItemProps {
  primary: React.ReactNode
  secondary?: React.ReactNode
  actions?: React.ReactNode
  metadata?: Array<{ label: string; value: string }>
}
```

### 9.5 `AlertCard`
```tsx
interface AlertCardProps {
  variant: 'warning' | 'info' | 'error' | 'success'
  title: string
  description?: string
  action?: React.ReactNode
  children?: React.ReactNode
}
```

## 10. 主题色彩规范

### 10.1 状态颜色
- **成功**: `green-100/green-800`
- **警告**: `amber-100/amber-900`
- **错误**: `red-100/red-800`
- **信息**: `blue-100/blue-800`
- **中性**: `gray-100/gray-800`

### 10.2 渐变背景
```css
/* 警告渐变 */
bg-gradient-to-r from-amber-50 to-orange-50
dark:from-amber-950/20 dark:to-orange-950/20

/* 信息渐变 */
bg-gradient-to-r from-blue-50 to-indigo-50
dark:from-blue-950/20 dark:to-indigo-950/20

/* 成功渐变 */
bg-gradient-to-r from-green-50 to-emerald-50
dark:from-green-950/20 dark:to-emerald-950/20
```

## 11. 文本样式规范

### 11.1 文本层级
- **页面标题**: `text-xl font-bold`
- **卡片标题**: `text-lg font-semibold`
- **小节标题**: `text-base font-medium`
- **描述文本**: `text-sm text-muted-foreground`
- **辅助文本**: `text-xs text-muted-foreground`

### 11.2 特殊文本
- **代码/单值**: `font-mono text-sm`
- **必填标记**: `<span className="text-red-500">*</span>`
- **链接**: `text-blue-600 hover:underline`

## 实施建议

1. **创建组件库**: 在 `/components/common/` 目录下创建这些公共组件
2. **样式常量**: 创建 `/lib/styles.ts` 文件定义常用的className组合
3. **主题配置**: 扩展 `tailwind.config.js` 添加自定义主题变量
4. **文档维护**: 为每个公共组件创建使用示例和API文档

## 优先级

基于使用频率，建议按以下优先级实施：

1. **高优先级**: PageHeader, FormField, AlertCard
2. **中优先级**: StatCard, ListItem
3. **低优先级**: 其他特定场景组件

## 收益分析

- **代码减少**: 预计可减少30-40%的样式代码重复
- **维护性提升**: 统一修改样式只需更新一处
- **开发效率**: 新页面开发速度提升50%
- **一致性**: 确保UI风格统一