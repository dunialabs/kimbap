'use client'

import type React from 'react'
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Eye, EyeOff, Key } from 'lucide-react'
import { MasterPasswordManager } from '@/lib/crypto'
import { renderErrorMessageWithLinks } from '@/lib/error-utils'

interface LoginFormProps {
  onSuccess: () => void
  onManualConnect: () => void
  defaultToken?: string
}

export function LoginForm({
  onSuccess,
  onManualConnect,
  defaultToken = ''
}: LoginFormProps) {
  const [hasMasterPassword, setHasMasterPassword] = useState(false)
  const [loginMode, setLoginMode] = useState<'token' | 'password'>('password')
  const [token, setToken] = useState(defaultToken)
  const [loginMasterPassword, setLoginMasterPassword] = useState('')
  const [showLoginPassword, setShowLoginPassword] = useState(false)
  const [loginError, setLoginError] = useState('')
  const [tokenError, setTokenError] = useState('')
  const [isLoggingIn, setIsLoggingIn] = useState(false)

  useEffect(() => {
    const checkMasterPassword = MasterPasswordManager.hasMasterPassword()
    setHasMasterPassword(checkMasterPassword)
    // If no master password is set, switch to token mode
    if (!checkMasterPassword) {
      setLoginMode('token')
    }
  }, [])

  const handleLogin = async () => {
    setIsLoggingIn(true)
    setLoginError('')
    setTokenError('')

    try {
      const { api } = await import('@/lib/api-client')
      const { MasterPasswordManager } = await import('@/lib/crypto')

      // Use protocol 10015 for both login methods
      const params: any = {}

      if (loginMode === 'password') {
        // Master password login
        if (!loginMasterPassword.trim()) {
          setLoginError('Please enter master password')
          setIsLoggingIn(false)
          return
        }
        params.masterPwd = loginMasterPassword
      } else {
        // Token login
        if (!token.trim()) {
          setTokenError('Please enter a valid token')
          setIsLoggingIn(false)
          return
        }
        params.accessToken = token
      }

      const response = await api.auth.login(params)

      if (!response.data?.data?.tokenInfo) {
        if (loginMode === 'password') {
          setLoginError('Invalid master password')
        } else {
          setTokenError('Invalid token')
        }
        setIsLoggingIn(false)
        return
      }

      const { tokenInfo, accessToken: masterPwdAccessToken } = response.data.data

      // Store userid in localStorage for API requests
      if (tokenInfo.userid) {
        localStorage.setItem('userid', tokenInfo.userid)
      }

      // If using master password, it's always Owner role
      // For token login, get role from tokenInfo
      const role =
        loginMode === 'password'
          ? 'Owner'
          : tokenInfo.role === 1
          ? 'Owner'
          : tokenInfo.role === 2
          ? 'Admin'
          : 'Member'

      // If login with master password, set and cache it
      // No need to verify locally since backend already validated it
      if (loginMode === 'password') {
        await MasterPasswordManager.setMasterPassword(loginMasterPassword)
      }

      // Store server info with role
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

      if (loginMode === 'token') {
        localStorage.setItem('auth_token', token.trim())
        localStorage.setItem('accessToken', token.trim())
      } else if (masterPwdAccessToken) {
        localStorage.setItem('auth_token', masterPwdAccessToken)
      }

      onSuccess()
    } catch (error: any) {
      // Error handled below via UI state
      if (loginMode === 'password') {
        setLoginError(error.message || 'Could not log in. Try again.')
      } else {
        setTokenError(error.message || 'Could not log in. Try again.')
      }
      setIsLoggingIn(false)
    }
  }

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        handleLogin()
      }}
      className="space-y-[12px] w-full max-w-[460px] py-[32px] px-[24px] min-h-[480px]"
    >
      <div>
        <h2 className="text-[24px] font-bold mb-[4px]">Login to Server</h2>
        <p className="text-muted-foreground text-[14px]">
          {loginMode === 'password'
            ? 'Enter your master password.'
            : 'Enter your access token.'}
        </p>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-border">
        <button
          type="button"
          onClick={() => {
            setLoginMode('password')
            setLoginError('')
            setTokenError('')
          }}
          className={`p-[12px] pl-[0] text-[14px] transition-colors ${
            loginMode === 'password'
              ? 'text-foreground font-bold'
              : 'text-foreground/60 font-[400]'
          }`}
        >
          Master Password
        </button>
        <button
          type="button"
          onClick={() => {
            setLoginMode('token')
            setLoginError('')
            setTokenError('')
          }}
          className={`p-[12px] pl-[0] text-[14px] transition-colors ${
            loginMode === 'token'
              ? 'text-foreground font-bold'
              : 'text-foreground/60 font-[400]'
          }`}
        >
          Access Token
        </button>
      </div>

      {/* Input Field */}
      <div className="space-y-[4px]">
        <div className="relative">
          <input
            type={
              loginMode === 'password'
                ? showLoginPassword
                  ? 'text'
                  : 'password'
                : 'text'
            }
            placeholder={
              loginMode === 'password'
                ? 'Enter Master Password'
                : 'Enter Access Token'
            }
            value={loginMode === 'password' ? loginMasterPassword : token}
            onChange={(e) => {
              if (loginMode === 'password') {
                setLoginMasterPassword(e.target.value)
                setLoginError('')
              } else {
                setToken(e.target.value)
                setTokenError('')
              }
            }}
            disabled={isLoggingIn}
            className="h-12 w-full pl-3 pr-10 rounded-lg border border-input bg-background focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-foreground"
            autoComplete={loginMode === 'password' ? 'current-password' : 'off'}
          />
          {loginMode === 'password' && (
            <button
              type="button"
              onClick={() => setShowLoginPassword(!showLoginPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 rounded"
              aria-label={showLoginPassword ? 'Hide password' : 'Show password'}
            >

              {showLoginPassword ? (
                <EyeOff className="h-5 w-5" />
              ) : (
                <Eye className="h-5 w-5" />
              )}
            </button>
          )}
        </div>

        {/* Note and Error Messages */}
        {loginMode === 'password' && !loginError && (
          <Alert className="border-blue-200 bg-blue-50 dark:bg-blue-950/20 dark:border-blue-900">
            <AlertDescription className="text-sm text-blue-700 dark:text-blue-200">
              <span className="font-[700]">Note:</span> The master password is
              required to access the console.
            </AlertDescription>
          </Alert>
        )}

        {loginError && loginMode === 'password' && (
           <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
            <AlertDescription className="text-sm text-red-800 dark:text-red-200">
              {renderErrorMessageWithLinks(loginError)}
            </AlertDescription>
          </Alert>
        )}

        {tokenError && loginMode === 'token' && (
           <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
            <AlertDescription className="text-sm text-red-800 dark:text-red-200">
              {tokenError === 'member_link' ? (
                <>
                  Members can only use the server. For configuration, please
                  switch token or{' '}
                  <a
                    href="https://www.kimbap.io/quick-start/#install-desk"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 underline cursor-pointer font-medium"
                  >
                    download Kimbap Desk
                  </a>
                  .
                </>
              ) : (
                renderErrorMessageWithLinks(tokenError)
              )}
            </AlertDescription>
          </Alert>
        )}
      </div>

      {/* Login Button */}
      <Button
        type="submit"
        disabled={
          isLoggingIn ||
          (loginMode === 'password'
            ? !loginMasterPassword.trim()
            : !token.trim())
        }
        className="w-full h-12 text-base bg-slate-900 hover:bg-slate-800 disabled:bg-slate-900 disabled:opacity-30 text-white rounded-[8px] dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900 dark:disabled:bg-slate-100"
        size="lg"
      >
        {isLoggingIn ? (
          <>
            <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
            Logging in...
          </>
        ) : (
          <>
            <Key className="w-4 h-4 mr-2" />
            Login
          </>
        )}
      </Button>

      {/* Manual Server Connection Link - Show if has master password */}
      {hasMasterPassword && (
        <div className="text-center">
          <button
            type="button"
            onClick={onManualConnect}
            disabled={isLoggingIn}
            className="px-6 w-full py-3 text-[14px] text-foreground bg-white border border-border rounded-lg shadow-sm hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors dark:bg-slate-800 dark:hover:bg-slate-700"
          >
            Manual Connection
          </button>
        </div>
      )}
    </form>
  )
}
