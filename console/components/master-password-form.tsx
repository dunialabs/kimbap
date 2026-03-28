'use client'

import type React from 'react'
import { useState, useEffect, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Eye, EyeOff, Loader2, Lock } from 'lucide-react'
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
  const masterPasswordInputRef = useRef<HTMLInputElement>(null)
  const passwordTooShort = masterPassword.length > 0 && masterPassword.length < 10
  const passwordsMismatch = confirmPassword.length > 0 && masterPassword !== confirmPassword

  // Check Web Crypto API availability on component mount
  useEffect(() => {
    if (typeof window !== 'undefined') {
      if (!window.crypto || !window.crypto.subtle) {
        setCryptoAvailable(false)
        setError(
          'This console requires a secure connection (HTTPS or localhost). ' +
          'Open it via https:// or http://localhost to continue. ' +
          'Details: https://docs.kimbap.sh/#caddy-https-config'
        )
      } else {
        setCryptoAvailable(true)
      }
    }
  }, []) // Only run on mount

  useEffect(() => {
    if (!cryptoAvailable) {
      return
    }

    const frame = window.requestAnimationFrame(() => {
      masterPasswordInputRef.current?.focus()
    })

    return () => window.cancelAnimationFrame(frame)
  }, [cryptoAvailable])

  const handleCreatePassword = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!masterPassword.trim()) {
      setError('Enter a master password.')
      return
    }

    if (masterPassword !== confirmPassword) {
      setError('Passwords do not match. Enter the same password in both fields.')
      return
    }

    if (masterPassword.length < 10) {
      setError('Choose a master password with at least 10 characters.')
      return
    }

    // First, check Web Crypto API availability before doing anything
    if (typeof window !== 'undefined' && (!window.crypto || !window.crypto.subtle)) {
      setError(
        'This console requires a secure connection (HTTPS or localhost). ' +
        'Open it via https:// or http://localhost to continue. ' +
        'Details: https://docs.kimbap.sh/#caddy-https-config'
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
      setError(error.message || 'Could not create the master password. Try again.')
    }
  }

  return (
    <div className="max-w-[460px] space-y-3 px-6 py-8">
      <div>
        <h2 className="mb-1 text-2xl font-bold">
          Create a Master Password
        </h2>
        <p className="text-sm text-muted-foreground">
          This password protects access to this console.
        </p>
      </div>

      <form onSubmit={handleCreatePassword} className="space-y-3">
        {/* Master Password */}
        <div className="space-y-1">
          <Label htmlFor="master-password" className="text-sm font-medium">
            Master Password
            <span className="ml-1 text-xs font-normal text-muted-foreground">(required)</span>
          </Label>
          <div className="relative">
            <Input
              id="master-password"
              type={showPassword ? 'text' : 'password'}
              placeholder="e.g., correct-horse-battery-staple"
              ref={masterPasswordInputRef}
              value={masterPassword}
              onChange={(e) => {
                setMasterPassword(e.target.value)
                setError('')
              }}
              disabled={isLoading || !cryptoAvailable}
              autoComplete="new-password"
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              className="h-12 rounded-lg px-3 pr-10 transition-colors duration-200 focus-visible:border-blue-500 focus-visible:ring-blue-500"
              aria-invalid={passwordTooShort || Boolean(error)}
              aria-describedby={[passwordTooShort ? 'master-password-hint' : '', error ? 'master-password-error' : ''].filter(Boolean).join(' ') || undefined}
              minLength={10}
              required
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-1 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded text-muted-foreground transition-colors duration-200 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              aria-label={showPassword ? 'Hide password' : 'Show password'}
               disabled={isLoading || !cryptoAvailable}
            >

              {showPassword ? (
                <EyeOff className="h-5 w-5" aria-hidden="true" />
              ) : (
                <Eye className="h-5 w-5" aria-hidden="true" />
              )}
            </button>
          </div>
          {masterPassword && masterPassword.length < 10 && (
            <p id="master-password-hint" className="text-xs text-muted-foreground" aria-live="polite">
              {masterPassword.length}/10 characters minimum
            </p>
          )}
        </div>

        {/* Confirm Master Password */}
        <div className="space-y-1">
          <Label htmlFor="confirm-password" className="text-sm font-medium">
            Confirm Master Password
            <span className="ml-1 text-xs font-normal text-muted-foreground">(required)</span>
          </Label>
          <div className="relative">
            <Input
              id="confirm-password"
              type={showConfirmPassword ? 'text' : 'password'}
              placeholder="e.g., correct-horse-battery-staple"
              value={confirmPassword}
              onChange={(e) => {
                setConfirmPassword(e.target.value)
                setError('')
              }}
              disabled={isLoading || !cryptoAvailable}
              autoComplete="new-password"
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              className="h-12 rounded-lg px-3 pr-10 transition-colors duration-200 focus-visible:border-blue-500 focus-visible:ring-blue-500"
              aria-invalid={passwordsMismatch || Boolean(error)}
              aria-describedby={[confirmPassword && masterPassword ? 'confirm-password-status' : '', error ? 'master-password-error' : ''].filter(Boolean).join(' ') || undefined}
              required
            />
            <button
              type="button"
              onClick={() => setShowConfirmPassword(!showConfirmPassword)}
              className="absolute right-1 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded text-muted-foreground transition-colors duration-200 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              aria-label={showConfirmPassword ? 'Hide confirm password' : 'Show confirm password'}
               disabled={isLoading || !cryptoAvailable}
            >

              {showConfirmPassword ? (
                <EyeOff className="h-5 w-5" aria-hidden="true" />
              ) : (
                <Eye className="h-5 w-5" aria-hidden="true" />
              )}
            </button>
          </div>
          {confirmPassword && masterPassword && (
            confirmPassword === masterPassword
              ? <p id="confirm-password-status" className="text-xs text-green-600 dark:text-green-400" aria-live="polite">Passwords match</p>
              : <p id="confirm-password-status" className="text-xs text-red-600 dark:text-red-400" aria-live="polite">Passwords do not match</p>
          )}
        </div>

        {/* Warning Message */}
        <Alert className="border-blue-200 bg-blue-50 dark:bg-blue-950/20 dark:border-blue-900">
          <AlertDescription className="text-sm text-blue-700 dark:text-blue-200">
            <span className="font-semibold">Important:</span> This password cannot
            be recovered. Store it somewhere safe.
          </AlertDescription>
        </Alert>

        {/* Error Message */}
        {error && (
           <Alert id="master-password-error" className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20" role="alert">
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
          className="h-12 w-full text-base"
          size="lg"
        >
          {isLoading ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
              Create Master Password
            </>
          ) : (
            <>
              <Lock className="w-4 h-4 mr-2" />
              Create Master Password
            </>
          )}
        </Button>
      </form>
    </div>
  )
}
