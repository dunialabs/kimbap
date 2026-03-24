"use client"

import { ReactNode, FormEvent } from "react"

import { Button } from "@/components/ui/button"

import { BaseDialog } from "./base-dialog"

interface FormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  children: ReactNode
  onSubmit: (e: FormEvent) => void | Promise<void>
  submitText?: string
  cancelText?: string
  isLoading?: boolean
  className?: string
}

export function FormDialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  onSubmit,
  submitText = "Submit",
  cancelText = "Cancel",
  isLoading = false,
  className,
}: FormDialogProps) {
  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    await onSubmit(e)
  }

  return (
    <BaseDialog
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={description}
      className={className}
      isLoading={isLoading}
      footer={
        <div className="flex gap-2">
          <Button 
            type="button" 
            variant="outline" 
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
          >
            {cancelText}
          </Button>
          <Button type="submit" form="dialog-form" disabled={isLoading}>
            {submitText}
          </Button>
        </div>
      }
    >
      <form id="dialog-form" onSubmit={handleSubmit}>
        {children}
      </form>
    </BaseDialog>
  )
}