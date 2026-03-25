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

interface AccessTokenDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (accessToken: string) => void
  title?: string
  description?: string
  isLoading?: boolean
  error?: string
}

export function AccessTokenDialog({
  open,
  onOpenChange,
  onConfirm,
  title = 'Access Token',
  description = 'Enter an access token to decrypt this configuration.',
  isLoading = false,
  error
}: AccessTokenDialogProps) {
  const [accessToken, setAccessToken] = useState('')
  const [showToken, setShowToken] = useState(false)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!accessToken.trim()) return

    onConfirm(accessToken.trim())
  }

  const handleClose = () => {
    setAccessToken('')
    setShowToken(false)
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
            <Label htmlFor="accessToken">Access Token</Label>
            <div className="relative">
              <Input
                id="accessToken"
                type={showToken ? 'text' : 'password'}
                value={accessToken}
                onChange={(e) => {
                  setAccessToken(e.target.value)
                }}
                placeholder="kimbap_..."
                disabled={isLoading}
                className={`pr-10 ${
                  error
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
                onClick={() => setShowToken(!showToken)}
                disabled={isLoading}
                aria-label={showToken ? 'Hide token' : 'Show token'}
              >

                {showToken ? (
                  <EyeOff className="h-4 w-4 text-gray-500 dark:text-gray-400" aria-hidden="true" />
                ) : (
                  <Eye className="h-4 w-4 text-gray-500 dark:text-gray-400" aria-hidden="true" />
                )}
              </Button>
            </div>
            {error && (
              <div className="flex items-center gap-2 text-sm text-red-600 dark:text-red-400 mt-1">
                <AlertCircle className="h-4 w-4" aria-hidden="true" />
                <span>{error}</span>
              </div>
            )}
            <p className="text-xs text-muted-foreground">
              Find this in Profile &gt; Access Tokens.
            </p>
          </div>

          <div className="flex gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={isLoading}
              className="flex-1 h-12 text-base bg-white border-border text-foreground hover:bg-gray-50 rounded-[8px] dark:bg-slate-800 dark:hover:bg-slate-700"
              size="lg"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!accessToken.trim() || isLoading}
              className="flex-1 h-12 text-base bg-slate-900 hover:bg-slate-800 disabled:bg-slate-900 disabled:opacity-30 text-white rounded-[8px] dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900 dark:disabled:bg-slate-100"
              size="lg"
            >
              {isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Decrypting...
                </>
              ) : (
                  'Decrypt'
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
