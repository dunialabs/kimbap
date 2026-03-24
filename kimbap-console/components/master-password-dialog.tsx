'use client'

import { useState } from 'react'
import { Eye, EyeOff, AlertCircle, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

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
import { MasterPasswordManager } from '@/lib/crypto'
import { renderErrorMessageWithLinks } from '@/lib/error-utils'

interface MasterPasswordDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (password: string) => void
  title?: string
  description?: string
  isLoading?: boolean
  showForgotPassword?: boolean // Show "Forgot Password" option for owner
}

export function MasterPasswordDialog({
  open,
  onOpenChange,
  onConfirm,
  title = 'Master Password',
  description = 'Enter the master password to continue.',
  isLoading = false,
  showForgotPassword = false
}: MasterPasswordDialogProps) {
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [isValidating, setIsValidating] = useState(false)
  const [validationError, setValidationError] = useState('')

  // Reset mode states
  const [isResetMode, setIsResetMode] = useState(false)
  const [resetToken, setResetToken] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmNewPassword, setConfirmNewPassword] = useState('')
  const [showNewPassword, setShowNewPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const [isResetting, setIsResetting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!password.trim()) return

    try {
      setIsValidating(true)
      setValidationError('')

      // Let server validate master password
      onConfirm(password.trim())
    } catch (error) {
      // Error will be shown via validationError state
      setValidationError('Could not validate master password. Try again.')
    } finally {
      setIsValidating(false)
    }
  }

  const handleResetSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    // Validate inputs
    if (!resetToken.trim()) {
      setValidationError('Access token is required')
      return
    }

    if (!newPassword.trim()) {
      setValidationError('New master password is required')
      return
    }

    if (newPassword.length < 10) {
      setValidationError('Master password must be at least 10 characters long')
      return
    }

    if (newPassword !== confirmNewPassword) {
      setValidationError('Passwords do not match')
      return
    }

    setIsResetting(true)
    setValidationError('Master password reset is not available in this build yet.')
    setIsResetting(false)
  }

  const handleClose = () => {
    setPassword('')
    setShowPassword(false)
    setValidationError('')
    setIsValidating(false)
    setIsResetMode(false)
    setResetToken('')
    setNewPassword('')
    setConfirmNewPassword('')
    setShowNewPassword(false)
    setShowConfirmPassword(false)
    setIsResetting(false)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[425px] py-[32px] px-[24px]" hideCloseButton={true}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-[24px] font-bold mb-[4px]">
            {isResetMode ? 'Reset Master Password' : title}
          </DialogTitle>
          <DialogDescription className="mt-[0px]">
            {isResetMode
              ? 'Enter your owner access token and create a new master password.'
              : description}
          </DialogDescription>
        </DialogHeader>

        {!isResetMode ? (
          // Normal validation mode
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="masterPassword">Master Password</Label>
              <div className="relative">
                <Input
                  id="masterPassword"
                  type={showPassword ? 'text' : 'password'}
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
                    <EyeOff className="h-4 w-4 text-muted-foreground" />
                  ) : (
                    <Eye className="h-4 w-4 text-muted-foreground" />
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
        ) : (
          // Reset mode
          <form onSubmit={handleResetSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="resetToken">Owner Access Token</Label>
              <Input
                id="resetToken"
                type="text"
                value={resetToken}
                onChange={(e) => {
                  setResetToken(e.target.value)
                  if (validationError) {
                    setValidationError('')
                  }
                }}
                placeholder="kimbap_..."
                disabled={isResetting}
                className={
                  validationError
                    ? 'border-red-500 focus-visible:ring-red-500'
                    : ''
                }
                autoFocus
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="newPassword">New Master Password</Label>
              <div className="relative">
                <Input
                  id="newPassword"
                  type={showNewPassword ? 'text' : 'password'}
                  value={newPassword}
                  onChange={(e) => {
                    setNewPassword(e.target.value)
                    if (validationError) {
                      setValidationError('')
                    }
                  }}
                  placeholder="At least 10 characters"
                  disabled={isResetting}
                  className="pr-10"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                  onClick={() => setShowNewPassword(!showNewPassword)}
                  disabled={isResetting}
                  aria-label={showNewPassword ? 'Hide password' : 'Show password'}
                >

                  {showNewPassword ? (
                    <EyeOff className="h-4 w-4 text-muted-foreground" />
                  ) : (
                    <Eye className="h-4 w-4 text-muted-foreground" />
                  )}
                </Button>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmNewPassword">Confirm New Password</Label>
              <div className="relative">
                <Input
                  id="confirmNewPassword"
                  type={showConfirmPassword ? 'text' : 'password'}
                  value={confirmNewPassword}
                  onChange={(e) => {
                    setConfirmNewPassword(e.target.value)
                    if (validationError) {
                      setValidationError('')
                    }
                  }}
                  placeholder="Repeat password"
                  disabled={isResetting}
                  className="pr-10"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  disabled={isResetting}
                  aria-label={showConfirmPassword ? 'Hide password' : 'Show password'}
                >

                  {showConfirmPassword ? (
                    <EyeOff className="h-4 w-4 text-muted-foreground" />
                  ) : (
                    <Eye className="h-4 w-4 text-muted-foreground" />
                  )}
                </Button>
              </div>
            </div>

            {validationError && (
              <div className="flex items-center gap-2 text-sm text-red-600 dark:text-red-400">
                <AlertCircle className="h-4 w-4" />
                <span>{validationError}</span>
              </div>
            )}

            <div className="flex gap-3">
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setIsResetMode(false)
                  setResetToken('')
                  setNewPassword('')
                  setConfirmNewPassword('')
                  setValidationError('')
                }}
                disabled={isResetting}
                className="flex-1 h-12 text-base bg-white border-border text-foreground hover:bg-gray-50 rounded-[8px] dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-slate-100"
                size="lg"
              >
                Back
              </Button>
              <Button
                type="submit"
                disabled={
                  !resetToken.trim() ||
                  !newPassword.trim() ||
                  !confirmNewPassword.trim() ||
                  isResetting
                }
                className="flex-1 h-12 text-base bg-slate-900 hover:bg-slate-800 disabled:bg-slate-900 disabled:opacity-30 text-white rounded-[8px] dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900 dark:disabled:bg-slate-100"
                size="lg"
              >
                {isResetting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Resetting...
                  </>
                ) : (
                  'Reset Password'
                )}
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}
