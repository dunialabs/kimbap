'use client'

import type React from 'react'
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Key, Eye, EyeOff, CheckCircle2 } from 'lucide-react'
import { renderErrorMessageWithLinks } from '@/lib/error-utils'

interface InitializeServerFormProps {
  onSuccess: (ownerToken: string) => void
  onManualConnect: () => void
  onRestore: () => void

  kimbapCoreStatus?: {
    isAvailable: number // 1-Available, 2-Not started, 3-Started but not KIMBAP Core, 4-Config saved
    host?: string
    port?: number
  } | null
}

export function InitializeServerForm({
  onSuccess,
  onManualConnect,
  onRestore,

  kimbapCoreStatus
}: InitializeServerFormProps) {
  const [isInitializing, setIsInitializing] = useState(false)
  const [error, setError] = useState('')
  const [masterPassword, setMasterPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const [cryptoAvailable, setCryptoAvailable] = useState(true)

  // Check Web Crypto API availability on component mount
  useEffect(() => {
    if (typeof window !== 'undefined') {
      if (!window.crypto || !window.crypto.subtle) {
        setCryptoAvailable(false)
        setError(
          'Web Crypto API is not available. This application requires HTTPS or localhost. ' +
          'Please access via HTTPS (https://...) or localhost (http://localhost:...). ' +
          'For more information, visit: https://docs.kimbap.io/#caddy-https-config'
        )
      } else {
        setCryptoAvailable(true)
      }
    }
  }, []) // Only run on mount

  const handleInitialize = async () => {
    setIsInitializing(true)
    setError('')

    // First, check Web Crypto API availability before doing anything
    if (typeof window !== 'undefined' && (!window.crypto || !window.crypto.subtle)) {
      setError(
        'Web Crypto API is not available. This application requires HTTPS or localhost. ' +
        'Please access via HTTPS (https://...) or localhost (http://localhost:...). ' +
        'For more information, visit: https://docs.kimbap.io/#caddy-https-config'
      )
      setIsInitializing(false)
      setCryptoAvailable(false)
      return
    }

    // Validate master password
    if (!masterPassword.trim()) {
      setError('Enter a master password')
      setIsInitializing(false)
      return
    }

    if (masterPassword.length < 10) {
      setError('Password must be at least 10 characters')
      setIsInitializing(false)
      return
    }

    if (masterPassword !== confirmPassword) {
      setError('Passwords do not match')
      setIsInitializing(false)
      return
    }

    try {
      const { MasterPasswordManager } = await import('@/lib/crypto')
      const { api } = await import('@/lib/api-client')

      // Verify crypto is still available before creating server
      if (typeof window !== 'undefined' && (!window.crypto || !window.crypto.subtle)) {
        throw new Error(
          'Web Crypto API is not available. This application requires HTTPS or localhost. ' +
          'Please access via HTTPS (https://...) or localhost (http://localhost:...). ' +
          'For more information, visit: https://docs.kimbap.io/#caddy-https-config'
        )
      }

      // Call protocol-10001 to create server and get token
      const response = await api.servers.create(masterPassword)

      if (!response.data?.data) {
        throw new Error('Could not initialize server')
      }

      const { accessToken, proxyId, proxyName, proxyKey, role, userid } =
        response.data.data

      // Initialization successful, now save the master password
      await MasterPasswordManager.setMasterPassword(masterPassword)

      const newServerInfo = {
        id: proxyId,
        proxyId: proxyId,
        name: proxyName,
        proxyName: proxyName,
        proxyKey: proxyKey || '',
        address: `https://server-${proxyId}.mcp.local`,
        status: 'running',
        role: role === 1 ? 'Owner' : 'Unknown',
        userRole: 'Admin'
      }

      // Store userid in localStorage for API requests
      if (userid) {
        localStorage.setItem('userid', userid)
      }

      localStorage.setItem('selectedServer', JSON.stringify(newServerInfo))
      localStorage.setItem('auth_token', accessToken)

      onSuccess(accessToken)
    } catch (error: any) {
      const errorMessage =
        error.response?.data?.common?.message ||
        error.response?.data?.message ||
        error.message ||
        'Could not initialize server. Try again.'
      setError(errorMessage)
      setIsInitializing(false)
    }
  }

  // Check if local KIMBAP Core is detected (isAvailable: 1 or 4)
  const hasLocalKimbapCore =
    kimbapCoreStatus?.isAvailable === 1 || kimbapCoreStatus?.isAvailable === 4

  return (
    <form onSubmit={(e) => { e.preventDefault(); handleInitialize(); }} className="space-y-[12px] max-w-[460px] py-[32px] px-[24px]">
      {/* Local KIMBAP Core Detection Banner */}
      {hasLocalKimbapCore && (
        <div
          className="flex items-center gap-3 p-3 mb-[4px] rounded-xl border border-emerald-200 bg-emerald-50 dark:bg-emerald-950/20 dark:border-emerald-800"
        >
          <CheckCircle2
            className="h-5 w-5 flex-shrink-0 text-emerald-500 dark:text-emerald-400"
          />
          <span
            className="text-emerald-600 dark:text-emerald-400 text-[14px]"
          >
            Local Kimbap Server Detected
          </span>
        </div>
      )}

      <div>
        <h2 className="text-[24px] font-bold mb-[4px]">Set Password & Initialize</h2>
        <p className="text-muted-foreground text-[14px]">
          Set a password to protect your server.
        </p>
      </div>

      {/* Master Password Input */}
      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="master-password" className="text-sm font-medium">
            Master Password
          </Label>
          <div className="relative">
            <Input
              id="master-password"
              type={showPassword ? 'text' : 'password'}
              value={masterPassword}
              onChange={(e) => setMasterPassword(e.target.value)}
              placeholder="At least 10 characters"
              className="pr-10"
              disabled={isInitializing || !cryptoAvailable}
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              disabled={isInitializing}
              aria-label={showPassword ? 'Hide password' : 'Show password'}
            >

              {showPassword ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="confirm-password" className="text-sm font-medium">
            Confirm Password
          </Label>
          <div className="relative">
            <Input
              id="confirm-password"
              type={showConfirmPassword ? 'text' : 'password'}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="Confirm your master password"
              className="pr-10"
              disabled={isInitializing || !cryptoAvailable}
            />
            <button
              type="button"
              onClick={() => setShowConfirmPassword(!showConfirmPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              disabled={isInitializing}
              aria-label={showConfirmPassword ? 'Hide password' : 'Show password'}
            >

              {showConfirmPassword ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
          </div>
        </div>
      </div>



      {/* Error Message */}
      {error && (
         <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
          <AlertDescription className="text-sm text-red-800 dark:text-red-200">
            {renderErrorMessageWithLinks(error)}
          </AlertDescription>
        </Alert>
      )}

      {/* Initialize Button */}
      <Button
        type="submit"
        disabled={isInitializing || !cryptoAvailable}
        className="w-full h-12 text-base bg-slate-900 hover:bg-slate-800 disabled:bg-slate-900 disabled:opacity-30 text-white rounded-[8px] dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900 dark:disabled:bg-slate-100"
        size="lg"
      >
        {isInitializing ? (
          <>
            <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
            Initializing...
          </>
        ) : (
          <>
            <Key className="w-4 h-4 mr-2" />
            Initialize Server
          </>
        )}
      </Button>

      {/* Manual Connect and Restore Buttons */}
      <div className="flex gap-3">
        <Button
          type="button"
          onClick={onManualConnect}
          disabled={isInitializing}
          variant="outline"
          className="flex-1 h-12 bg-white border-border text-foreground hover:bg-gray-50 rounded-[8px] dark:bg-slate-800 dark:hover:bg-slate-700"
        >
          Manual Connect
        </Button>
        <Button
          type="button"
          onClick={onRestore}
          disabled={isInitializing}
          variant="outline"
          className="flex-1 h-12 bg-white border-border text-foreground hover:bg-gray-50 rounded-[8px] dark:bg-slate-800 dark:hover:bg-slate-700"
        >
          Restore
        </Button>
      </div>
    </form>
  )
}
