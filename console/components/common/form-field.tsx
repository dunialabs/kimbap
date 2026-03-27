import React from 'react'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'

interface FormFieldProps {
  label: string
  htmlFor?: string
  required?: boolean
  helpText?: string
  error?: string
  className?: string
  children: React.ReactNode
}

export function FormField({
  label,
  htmlFor,
  required,
  helpText,
  error,
  className,
  children
}: FormFieldProps) {
  return (
    <div className={cn('space-y-2', className)}>
      <Label htmlFor={htmlFor}>
        {label} {required && <span className="text-red-500 dark:text-red-400">*</span>}
      </Label>
      {children}
      {helpText && !error && (
        <p className="text-sm text-muted-foreground">{helpText}</p>
      )}
      {error && <p className="text-sm text-red-500 dark:text-red-400">{error}</p>}
    </div>
  )
}