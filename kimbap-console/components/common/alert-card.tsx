import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { cn } from '@/lib/utils'


interface AlertCardProps {
  variant: 'warning' | 'info' | 'error' | 'success'
  title: string
  description?: string
  action?: React.ReactNode
  className?: string
  children?: React.ReactNode
}

const variantStyles = {
  warning: {
    container: 'border-amber-200 dark:border-amber-800 bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-950/20 dark:to-orange-950/20',
    title: 'text-amber-900 dark:text-amber-100',
    description: 'text-amber-700 dark:text-amber-200',

  },
  info: {
    container: 'border-blue-200 dark:border-blue-800 bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-950/20 dark:to-indigo-950/20',
    title: 'text-blue-900 dark:text-blue-100',
    description: 'text-blue-700 dark:text-blue-200',

  },
  error: {
    container: 'border-red-200 dark:border-red-800 bg-gradient-to-r from-red-50 to-pink-50 dark:from-red-950/20 dark:to-pink-950/20',
    title: 'text-red-900 dark:text-red-100',
    description: 'text-red-700 dark:text-red-200',

  },
  success: {
    container: 'border-green-200 dark:border-green-800 bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-950/20 dark:to-emerald-950/20',
    title: 'text-green-900 dark:text-green-100',
    description: 'text-green-700 dark:text-green-200',

  }
}

export function AlertCard({
  variant,
  title,
  description,
  action,
  className,
  children
}: AlertCardProps) {
  const styles = variantStyles[variant]


  return (
    <Card className={cn(styles.container, className)}>
      <CardHeader>
        <CardTitle className={cn(styles.title)}>
          {title}
        </CardTitle>
        {description && (
          <CardDescription className={styles.description}>
            {description}
          </CardDescription>
        )}
      </CardHeader>
      {(children || action) && (
        <CardContent>
          {children}
          {action && (
            <div className="mt-4 flex items-center gap-3">
              {action}
            </div>
          )}
        </CardContent>
      )}
    </Card>
  )
}