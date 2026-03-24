# styles.ts 使用指南

## 1. 导入方式

```tsx
import { 
  pageStyles, 
  cardStyles, 
  dialogSizes, 
  formStyles,
  textStyles,
  statusStyles,
  cx 
} from '@/lib/styles'
```

## 2. 实际使用示例

### 2.1 页面布局使用

**原代码：**
```tsx
// 之前的写法（members/page.tsx）
<div className="space-y-4">
  <div>
    <h1 className="text-xl font-bold">Access Token Management</h1>
    <p className="text-sm text-muted-foreground">Manage API access tokens</p>
  </div>
</div>
```

**使用 styles.ts 后：**
```tsx
import { pageStyles } from '@/lib/styles'

<div className={pageStyles.container}>
  <div>
    <h1 className={pageStyles.header.title}>Access Token Management</h1>
    <p className={pageStyles.header.description}>Manage API access tokens</p>
  </div>
</div>
```

### 2.2 卡片头部使用

**原代码：**
```tsx
<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
  <CardTitle className="text-lg flex items-center gap-2">
    <Key className="h-5 w-5" />
    Access Tokens
  </CardTitle>
</CardHeader>
```

**使用 styles.ts 后：**
```tsx
import { cardStyles } from '@/lib/styles'

<CardHeader className={cardStyles.header.withAction}>
  <CardTitle className={cardStyles.title.withIcon}>
    <Key className="h-5 w-5" />
    Access Tokens
  </CardTitle>
</CardHeader>
```

### 2.3 对话框尺寸使用

**原代码：**
```tsx
<DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
```

**使用 styles.ts 后：**
```tsx
import { dialogSizes } from '@/lib/styles'

<DialogContent className={dialogSizes.lg}>
```

### 2.4 表单布局使用

**原代码：**
```tsx
<div className="grid grid-cols-2 gap-4">
  <div className="space-y-2">
    <Label>Name</Label>
    <Input />
  </div>
  <div className="space-y-2">
    <Label>Email</Label>
    <Input />
  </div>
</div>
```

**使用 styles.ts 后：**
```tsx
import { formStyles } from '@/lib/styles'

<div className={formStyles.grid.two}>
  <div className={formStyles.field}>
    <Label>Name</Label>
    <Input />
  </div>
  <div className={formStyles.field}>
    <Label>Email</Label>
    <Input />
  </div>
</div>
```

### 2.5 状态样式使用

**原代码：**
```tsx
<Card className="border-amber-200 bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-950/20 dark:to-orange-950/20">
  <CardTitle className="text-amber-900 dark:text-amber-100">
    Warning
  </CardTitle>
</Card>
```

**使用 styles.ts 后：**
```tsx
import { statusStyles } from '@/lib/styles'

<Card className={`${statusStyles.warning.border} ${statusStyles.warning.bg}`}>
  <CardTitle className={statusStyles.warning.text}>
    Warning
  </CardTitle>
</Card>
```

### 2.6 组合多个样式（使用 cx 工具函数）

```tsx
import { cx, formStyles, buttonStyles } from '@/lib/styles'

// 条件性组合样式
<div className={cx(
  formStyles.field,
  hasError && 'border-red-500',
  isDisabled && 'opacity-50'
)}>
  {/* content */}
</div>

// 组合按钮样式
<Button 
  className={cx(
    buttonStyles.transparent,
    buttonStyles.small,
    isActive && 'bg-blue-50'
  )}
>
  Click me
</Button>
```

## 3. 完整页面改造示例

### 改造前的页面代码：
```tsx
export default function SamplePage() {
  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-bold">Page Title</h1>
        <p className="text-sm text-muted-foreground">Page description</p>
      </div>
      
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <CardTitle className="text-lg">Card Title</CardTitle>
          <Button>Action</Button>
        </CardHeader>
        <CardContent>
          <div className="space-y-6">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Field 1</Label>
                <Input />
              </div>
              <div className="space-y-2">
                <Label>Field 2</Label>
                <Input />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
```

### 改造后使用 styles.ts：
```tsx
import { 
  pageStyles, 
  cardStyles, 
  formStyles,
  spacing 
} from '@/lib/styles'

export default function SamplePage() {
  return (
    <div className={pageStyles.container}>
      <div>
        <h1 className={pageStyles.header.title}>Page Title</h1>
        <p className={pageStyles.header.description}>Page description</p>
      </div>
      
      <Card>
        <CardHeader className={cardStyles.header.withAction}>
          <CardTitle className={cardStyles.title.default}>Card Title</CardTitle>
          <Button>Action</Button>
        </CardHeader>
        <CardContent>
          <div className={spacing.relaxed}>
            <div className={formStyles.grid.two}>
              <div className={formStyles.field}>
                <Label>Field 1</Label>
                <Input />
              </div>
              <div className={formStyles.field}>
                <Label>Field 2</Label>
                <Input />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
```

## 4. 优势

1. **一致性**: 所有页面使用相同的样式常量
2. **可维护性**: 修改样式只需要在 styles.ts 中更改一处
3. **语义化**: `pageStyles.header.title` 比 `text-xl font-bold` 更有意义
4. **类型安全**: TypeScript 会提供自动补全和类型检查
5. **减少错误**: 避免拼写错误和样式不一致

## 5. 最佳实践

1. **优先使用预定义样式**：先查看 styles.ts 是否有合适的样式
2. **扩展而不是覆盖**：如需特殊样式，使用 cx() 函数组合
3. **保持语义化**：选择有意义的样式名称
4. **定期更新**：发现新的重复模式时，添加到 styles.ts

## 6. VS Code 代码片段

可以创建代码片段来快速使用：

```json
{
  "Import Styles": {
    "prefix": "styles",
    "body": [
      "import { ${1:pageStyles}, ${2:cardStyles} } from '@/lib/styles'"
    ]
  },
  "Page Layout": {
    "prefix": "page-layout",
    "body": [
      "<div className={pageStyles.container}>",
      "  <div>",
      "    <h1 className={pageStyles.header.title}>${1:Title}</h1>",
      "    <p className={pageStyles.header.description}>${2:Description}</p>",
      "  </div>",
      "  $0",
      "</div>"
    ]
  }
}
```