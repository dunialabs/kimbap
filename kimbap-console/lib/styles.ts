/**
 * 公共样式常量
 * 定义项目中常用的className组合，提高样式复用性
 */

// 页面布局
export const pageStyles = {
  container: 'space-y-4',
  header: {
    wrapper: 'flex items-center justify-between',
    title: 'text-xl font-bold',
    description: 'text-sm text-muted-foreground'
  }
}

// 卡片样式
export const cardStyles = {
  header: {
    default: 'pb-4',
    withAction: 'flex flex-row items-center justify-between space-y-0 pb-4',
    compact: 'flex flex-row items-center justify-between space-y-0 pb-2'
  },
  title: {
    default: 'text-lg font-semibold',
    withIcon: 'text-lg flex items-center gap-2',
    small: 'text-sm font-medium'
  }
}

// 对话框尺寸
export const dialogSizes = {
  sm: 'sm:max-w-md',
  md: 'sm:max-w-lg',
  lg: 'sm:max-w-2xl max-h-[80vh] overflow-y-auto',
  xl: 'max-w-4xl max-h-[90vh] overflow-y-auto'
}

// 按钮样式
export const buttonStyles = {
  transparent: 'bg-transparent',
  iconOnly: 'h-6 w-6 p-0',
  small: 'text-xs'
}

// 表单布局
export const formStyles = {
  field: 'space-y-2',
  grid: {
    two: 'grid grid-cols-2 gap-4',
    three: 'grid md:grid-cols-3 gap-4'
  },
  section: 'space-y-4',
  container: 'space-y-6'
}

// 列表样式
export const listStyles = {
  item: 'p-4 border rounded-lg',
  itemHeader: 'flex items-center justify-between mb-3',
  itemMeta: 'flex items-center gap-6 text-xs text-muted-foreground',
  container: 'space-y-4'
}

// 文本样式
export const textStyles = {
  pageTitle: 'text-xl font-bold',
  sectionTitle: 'text-lg font-semibold',
  cardTitle: 'text-base font-medium',
  label: 'text-sm font-medium',
  description: 'text-sm text-muted-foreground',
  helper: 'text-xs text-muted-foreground',
  code: 'font-mono text-sm',
  required: 'text-red-500 dark:text-red-400'
}

// 间距规范
export const spacing = {
  tight: 'space-y-1',
  compact: 'space-y-2',
  normal: 'space-y-3',
  comfortable: 'space-y-4',
  relaxed: 'space-y-6'
}

// 网格间距
export const gridGap = {
  tight: 'gap-2',
  normal: 'gap-4',
  relaxed: 'gap-6'
}

// 状态样式
export const statusStyles = {
  success: {
    border: 'border-green-200 dark:border-green-800',
    bg: 'bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-950/20 dark:to-emerald-950/20',
    text: 'text-green-900 dark:text-green-100',
    subtext: 'text-green-700 dark:text-green-200'
  },
  warning: {
    border: 'border-amber-200 dark:border-amber-800',
    bg: 'bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-950/20 dark:to-orange-950/20',
    text: 'text-amber-900 dark:text-amber-100',
    subtext: 'text-amber-700 dark:text-amber-200'
  },
  error: {
    border: 'border-red-200 dark:border-red-800',
    bg: 'bg-gradient-to-r from-red-50 to-pink-50 dark:from-red-950/20 dark:to-pink-950/20',
    text: 'text-red-900 dark:text-red-100',
    subtext: 'text-red-700 dark:text-red-200'
  },
  info: {
    border: 'border-blue-200 dark:border-blue-800',
    bg: 'bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-950/20 dark:to-indigo-950/20',
    text: 'text-blue-900 dark:text-blue-100',
    subtext: 'text-blue-700 dark:text-blue-200'
  }
}

// Badge样式
export const badgeStyles = {
  admin: 'bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200 border-blue-200 dark:border-blue-800',
  member: 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200 border-green-200 dark:border-green-800',
  owner: 'bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-200 border-purple-200 dark:border-purple-800'
}

// 工具函数：组合多个className
export function cx(...classes: (string | undefined | null | false)[]) {
  return classes.filter(Boolean).join(' ')
}