'use client'

import { useState } from 'react'
import { Eye, EyeOff, AlertCircle, Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { renderErrorMessageWithLinks } from '@/lib/error-utils'

interface MasterPasswordDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (password: string) => void | Promise<void>
  title?: string
  description?: string
  isLoading?: boolean
}

export function MasterPasswordDialog({
  open,
  onOpenChange,
  onConfirm,
  title = 'Master Password',
  description = 'Enter the master password to continue.',
  isLoading = false,
}: MasterPasswordDialogProps) {
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [isValidating, setIsValidating] = useState(false)
  const [validationError, setValidationError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!password.trim()) return

    try {
      setIsValidating(true)
      setValidationError('')
      await onConfirm(password.trim())
    } catch (error) {
      setValidationError('Could not validate master password. Try again.')
    } finally {
      setIsValidating(false)
    }
  }

  const handleClose = () => {
    setPassword('')
    setShowPassword(false)
    setValidationError('')
    setIsValidating(false)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[425px] py-[32px] px-[24px]" hideCloseButton={true}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-[24px] font-bold mb-[4px]">
            {title}
          </DialogTitle>
          <DialogDescription className="mt-[0px]">
            {description}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="masterPassword">Master Password</Label>
            <div className="relative">
              <Input
                id="masterPassword"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                value={password}
                onChange={(e) => {
                  setPassword(e.target.value)
                  if (validationError) {
                    setValidationError('')
                  }
                }}
                placeholder="........"
                disabled={isLoading || isValidating}
                className={`pr-10 ${
                  validationError
                    ? 'border-red-500 focus-visible:ring-red-500'
                    : ''
                }`}
                autoFocus
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                onClick={() => setShowPassword(!showPassword)}
                disabled={isLoading || isValidating}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? (
                  <EyeOff className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                ) : (
                  <Eye className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                )}
              </Button>
            </div>
            {validationError && (
              <div className="flex items-center gap-2 text-sm text-red-600 dark:text-red-400 mt-1">
                <AlertCircle className="h-4 w-4" />
                <span>{renderErrorMessageWithLinks(validationError)}</span>
              </div>
            )}
          </div>
          <div className="flex gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={isLoading || isValidating}
              className="flex-1 h-12 text-base bg-white border-border text-foreground hover:bg-gray-50 rounded-[8px] dark:bg-slate-800 dark:hover:bg-slate-700"
              size="lg"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!password.trim() || isLoading || isValidating}
              className="flex-1 h-12 text-base bg-slate-900 hover:bg-slate-800 disabled:bg-slate-900 disabled:opacity-30 text-white rounded-[8px] dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900 dark:disabled:bg-slate-100"
              size="lg"
            >
              {isValidating ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Validating...
                </>
              ) : isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Processing...
                </>
              ) : (
                'Continue'
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
