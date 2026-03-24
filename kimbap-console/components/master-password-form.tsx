'use client'

import type React from 'react'
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Eye, EyeOff, Lock } from 'lucide-react'
import { renderErrorMessageWithLinks } from '@/lib/error-utils'

interface MasterPasswordFormProps {
  onSuccess: () => void
}

export function MasterPasswordForm({ onSuccess }: MasterPasswordFormProps) {
  const [masterPassword, setMasterPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [cryptoAvailable, setCryptoAvailable] = useState(true)

  // Check Web Crypto API availability on component mount
  useEffect(() => {
    if (typeof window !== 'undefined') {
      if (!window.crypto || !window.crypto.subtle) {
        setCryptoAvailable(false)
        setError(
          'Web Crypto API is not available. This application requires HTTPS or localhost. ' +
          'Please access via HTTPS (https://...) or localhost (http://localhost:...). ' +
          'For more information, visit: https://docs.kimbap.sh/#caddy-https-config'
        )
      } else {
        setCryptoAvailable(true)
      }
    }
  }, []) // Only run on mount

  const handleCreatePassword = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!masterPassword.trim()) {
      setError('Please enter a master password')
      return
    }

    if (masterPassword !== confirmPassword) {
      setError('Passwords do not match')
      return
    }

    if (masterPassword.length < 10) {
      setError('Password must be at least 10 characters long')
      return
    }

    // First, check Web Crypto API availability before doing anything
    if (typeof window !== 'undefined' && (!window.crypto || !window.crypto.subtle)) {
      setError(
        'Web Crypto API is not available. This application requires HTTPS or localhost. ' +
        'Please access via HTTPS (https://...) or localhost (http://localhost:...). ' +
        'For more information, visit: https://docs.kimbap.sh/#caddy-https-config'
      )
      setCryptoAvailable(false)
      return
    }

    setIsLoading(true)
    setError('')

    try {
      const { MasterPasswordManager } = await import('@/lib/crypto')
      await MasterPasswordManager.setMasterPassword(masterPassword)
      onSuccess()
    } catch (error: any) {
      setIsLoading(false)
      setError(error.message || 'Could not create master password')
    }
  }

  return (
    <div className="space-y-[12px] max-w-[460px] py-[32px] px-[24px]">
      <div>
        <h2 className="text-[24px] font-bold mb-[4px]">
          Set Your Master Password
        </h2>
        <p className="text-muted-foreground text-[14px]">
          This password protects your server configuration.
        </p>
      </div>

      <form onSubmit={handleCreatePassword} className="space-y-[12px]">
        {/* Master Password */}
        <div className="space-y-[4px]">
          <Label htmlFor="master-password" className="text-[14px] font-[700]">
            Master Password
          </Label>
          <div className="relative">
            <input
              id="master-password"
              type={showPassword ? 'text' : 'password'}
              placeholder="At least 10 characters"
              value={masterPassword}
              onChange={(e) => {
                setMasterPassword(e.target.value)
                setError('')
              }}
              disabled={isLoading || !cryptoAvailable}
              className="h-12 w-full pl-3 pr-10 rounded-lg border border-input bg-background focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              required
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 rounded"
              aria-label={showPassword ? 'Hide password' : 'Show password'}
            >

              {showPassword ? (
                <EyeOff className="h-5 w-5" />
              ) : (
                <Eye className="h-5 w-5" />
              )}
            </button>
          </div>
        </div>

        {/* Confirm Master Password */}
        <div className="space-y-[4px]">
          <Label htmlFor="confirm-password" className="text-[14px] font-[700]">
            Confirm Master Password
          </Label>
          <div className="relative">
            <input
              id="confirm-password"
              type={showConfirmPassword ? 'text' : 'password'}
              placeholder="Re-enter master password"
              value={confirmPassword}
              onChange={(e) => {
                setConfirmPassword(e.target.value)
                setError('')
              }}
              disabled={isLoading || !cryptoAvailable}
              className="h-12 w-full pl-3 pr-10 rounded-lg border border-input bg-background focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              required
            />
            <button
              type="button"
              onClick={() => setShowConfirmPassword(!showConfirmPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 rounded"
              aria-label={showConfirmPassword ? 'Hide password' : 'Show password'}
            >

              {showConfirmPassword ? (
                <EyeOff className="h-5 w-5" />
              ) : (
                <Eye className="h-5 w-5" />
              )}
            </button>
          </div>
        </div>

        {/* Warning Message */}
        <Alert className="border-blue-200 bg-blue-50 dark:bg-blue-950/20 dark:border-blue-900">
          <AlertDescription className="text-sm text-blue-700 dark:text-blue-200">
            <span className="font-[700]">Important:</span> This password cannot
            be recovered. Make sure to remember it or store it in a secure
            location.
          </AlertDescription>
        </Alert>

        {/* Error Message */}
        {error && (
           <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20" role="alert">
            <AlertDescription className="text-sm text-red-800 dark:text-red-200">
              {renderErrorMessageWithLinks(error)}
            </AlertDescription>
          </Alert>
        )}

        {/* Submit Button */}
        <Button
          type="submit"
          disabled={
            isLoading || !cryptoAvailable || !masterPassword.trim() || !confirmPassword.trim()
          }
          className="w-full h-12 text-base bg-slate-900 hover:bg-slate-800 disabled:bg-slate-900 disabled:opacity-30 text-white dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900 dark:disabled:bg-slate-100"
          size="lg"
        >
          {isLoading ? (
            <>
              <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
              Creating...
            </>
          ) : (
            <>
              <Lock className="w-4 h-4 mr-2" />
              Create Password
            </>
          )}
        </Button>
      </form>
    </div>
  )
}
