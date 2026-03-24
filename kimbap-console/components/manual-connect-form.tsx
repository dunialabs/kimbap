'use client'

import type React from 'react'
import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Eye, EyeOff, Key } from 'lucide-react'

interface ManualConnectFormProps {
  onSuccess: () => void
  onBack?: () => void
}

export function ManualConnectForm({
  onSuccess,
  onBack
}: ManualConnectFormProps) {
  const router = useRouter()
  const [host, setHost] = useState('')
  const [port, setPort] = useState('')
  const [accessToken, setAccessToken] = useState('')
  const [showToken, setShowToken] = useState(false)
  const [isValidating, setIsValidating] = useState(false)
  const [error, setError] = useState('')

  const handleConnect = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!host.trim()) {
      setError('Please enter a host address')
      return
    }

    // Port is optional - only validate if provided
    let portNum: number | undefined
    if (port.trim()) {
      portNum = parseInt(port.trim())
      if (!portNum || portNum <= 0 || portNum > 65535) {
        setError('Port must be between 1 and 65535')
        return
      }
    }

    setIsValidating(true)
    setError('')

    try {
      const { api } = await import('@/lib/api-client')

      // Step 1: Call protocol 10022 to validate and save KIMBAP Core config
      const configResponse = await api.servers.configureKimbapCore({
        host: host.trim(),
        port: portNum
      })


      if (configResponse.data?.data) {
        const { isValid, message } = configResponse.data.data

        if (isValid === 1) {
          // Success - configuration saved

          // Save access token if provided
          if (accessToken.trim()) {
            localStorage.setItem('manualAccessToken', accessToken.trim())

            // Check if server is already initialized (protocol 10002)
            try {
              const serverInfoResponse = await api.servers.getInfo()

              const serverExists = serverInfoResponse.data?.data?.proxyId

              if (serverExists) {
                // Server is initialized, attempt auto-login with access token

                const loginResponse = await api.auth.login({
                  accessToken: accessToken.trim()
                })


                if (loginResponse.data?.data?.tokenInfo) {
                  const { tokenInfo } = loginResponse.data.data

                  // Store userid
                  if (tokenInfo.userid) {
                    localStorage.setItem('userid', tokenInfo.userid)
                  }

                  // Determine role
                  const role =
                    tokenInfo.role === 1
                      ? 'Owner'
                      : tokenInfo.role === 2
                      ? 'Admin'
                      : 'Member'

                  // Store server info
                  const serverInfo = {
                    id: tokenInfo.proxyId || 'default',
                    proxyId: tokenInfo.proxyId || 'default',
                    name: tokenInfo.proxyName || 'MCP Server',
                    proxyName: tokenInfo.proxyName || 'MCP Server',
                    proxyKey: tokenInfo.proxyKey || '',
                    address: `https://server-${tokenInfo.proxyId}.mcp.local`,
                    status: 'running',
                    role: role,
                    userRole: role === 'Owner' ? 'Admin' : role
                  }

                  localStorage.setItem('selectedServer', JSON.stringify(serverInfo))
                  localStorage.setItem('auth_token', accessToken.trim())
                  localStorage.setItem('accessToken', accessToken.trim())


                  // Redirect to dashboard directly (same as login form)
                  router.push('/dashboard')
                  return
                } else {
                  // Login failed, proceed to normal login flow
                }
              }
            } catch (error) {
              // Auto-login check failed, proceed to normal login flow
              // If error, proceed to normal login flow
            }
          }

          // Call success callback (goes to login page)
          onSuccess()
        } else if (isValid === 2) {
          // Host/port invalid or cannot connect
          setError(message || 'Cannot reach this host and port')
          setIsValidating(false)
        } else if (isValid === 3) {
          // Port responding but not KIMBAP Core
          setError(message || 'This port is not running Kimbap Core')
          setIsValidating(false)
        }
      }
    } catch (error: any) {
      // Error handled below with setError()
      setError(
        error.response?.data?.error ||
          error.message ||
          'Could not validate Kimbap Core configuration'
      )
      setIsValidating(false)
    }
  }

  return (
    <div className="space-y-[12px] max-w-[460px] py-[32px] px-[24px] rounded-xl border border-border shadow-[0_0_12px_0_rgba(0,0,0,0.12)]">
      <div>
        <h2 className="text-[24px] font-bold mb-[4px]">
          Manual Server Connection
        </h2>
        <p className="text-muted-foreground text-[14px]">
          Enter the host and port to connect.
        </p>
      </div>

      <form onSubmit={handleConnect} className="space-y-[12px]">
        {/* Access Token (Optional) */}
        <div className="space-y-[4px]">
          <Label htmlFor="access-token" className="text-[14px] font-[700]">
            Access Token{' '}
            <span className="text-muted-foreground text-[14px] font-[700]">
              (Optional)
            </span>
          </Label>
          <div className="relative">
            <input
              id="access-token"
              type={showToken ? 'text' : 'password'}
              placeholder="Enter access token (optional)"
              value={accessToken}
              onChange={(e) => {
                setAccessToken(e.target.value)
                setError('')
              }}
              disabled={isValidating}
              className="h-12 w-full pl-3 pr-10 rounded-lg border border-input bg-background focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
            <button
              type="button"
              onClick={() => setShowToken(!showToken)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 rounded"
              aria-label={showToken ? 'Hide access token' : 'Show access token'}
            >
              {showToken ? (
                <EyeOff className="h-5 w-5" />
              ) : (
                <Eye className="h-5 w-5" />
              )}
            </button>
          </div>
        </div>

        {/* Host and Port */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {/* Host */}
          <div className="space-y-[4px]">
            <Label htmlFor="host" className="text-[14px] font-[700]">
              Host
            </Label>
            <input
              id="host"
              type="text"
              placeholder="e.g., localhost"
              value={host}
              onChange={(e) => {
                setHost(e.target.value)
                setError('')
              }}
              disabled={isValidating}
              className="h-12 w-full px-3 rounded-lg border border-input bg-background focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              required
            />
          </div>

          {/* Port */}
          <div className="space-y-[4px]">
            <Label htmlFor="port" className="text-[14px] font-[700]">
              Port{' '}
              <span className="text-muted-foreground text-[14px] font-[700]">
                (Optional)
              </span>
            </Label>
            <input
              id="port"
              type="number"
              placeholder="e.g., 3002"
              value={port}
              onChange={(e) => {
                setPort(e.target.value)
                setError('')
              }}
              disabled={isValidating}
              className="h-12 w-full px-3 rounded-lg border border-input bg-background focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
        </div>

        {/* Error Message */}
        {error && (
           <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
            <AlertDescription className="text-sm text-red-800 dark:text-red-200">
              {error}
            </AlertDescription>
          </Alert>
        )}

        {/* Action Buttons */}
        <div className="flex gap-3 justify-end">
          {onBack && (
            <Button
              type="button"
              onClick={onBack}
              disabled={isValidating}
              variant="outline"
              className="h-12 px-6 flex-1 bg-white border-border text-foreground hover:bg-gray-50 rounded-[8px] dark:bg-slate-800 dark:hover:bg-slate-700"
            >
              Cancel
            </Button>
          )}
          <Button
            type="submit"
            disabled={isValidating || !host.trim()}
            className="flex-1 h-12 px-6 text-base bg-slate-900 hover:bg-slate-800 disabled:bg-slate-900 disabled:opacity-30 text-white rounded-[8px] dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900 dark:disabled:bg-slate-100"
            size="lg"
          >
            {isValidating ? (
              <>
                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
                Validating...
              </>
            ) : (
              <>
                <Key className="w-4 h-4 mr-2" />
                Connect
              </>
            )}
          </Button>
        </div>
      </form>
    </div>
  )
}
