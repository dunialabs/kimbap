'use client'

import type React from 'react'
import { useState, useEffect, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Eye, EyeOff, Loader2, LogIn } from 'lucide-react'
import { MasterPasswordManager } from '@/lib/crypto'
import { renderErrorMessageWithLinks } from '@/lib/error-utils'

interface LoginFormProps {
  onSuccess: () => void
  defaultToken?: string
}

const safeStorageSet = (key: string, value: string): boolean => {
  try {
    localStorage.setItem(key, value)
    return true
  } catch {
    return false
  }
}

const safeStorageRemove = (key: string): void => {
  try {
    localStorage.removeItem(key)
  } catch {
    return
  }
}

const generateSessionCookieValue = () => {
  const bytes = new Uint8Array(16)
  window.crypto.getRandomValues(bytes)
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
}

function getLoginErrorMessage(
  error: unknown,
  mode: 'token' | 'password'
): string {
  const requestError = error as {
    response?: { status?: number; data?: { common?: { message?: string } } }
    userMessage?: string
    message?: string
    code?: string
  }
  const status = requestError.response?.status
  const rawMessage =
    requestError.userMessage ||
    requestError.response?.data?.common?.message ||
    requestError.message ||
    ''

  if (rawMessage === 'member_link') {
    return rawMessage
  }

  if (status === 401 || status === 403) {
    return rawMessage || (mode === 'password'
      ? "We couldn't verify that master password. Check it and try again."
      : "We couldn't verify that access token. Check it and try again.")
  }

  if (!requestError.response || requestError.code === 'ECONNABORTED') {
    return mode === 'password'
      ? 'Could not reach the console to verify your master password. Check your connection and try again.'
      : 'Could not reach the console to verify your access token. Check your connection and try again.'
  }

  return rawMessage || 'Could not sign in. Check your details and try again.'
}

export function LoginForm({
  onSuccess,
  defaultToken = ''
}: LoginFormProps) {
  const [loginMode, setLoginMode] = useState<'token' | 'password'>(() =>
    defaultToken.trim() ? 'token' : 'password'
  )
  const [token, setToken] = useState(defaultToken)
  const [loginMasterPassword, setLoginMasterPassword] = useState('')
  const [showLoginPassword, setShowLoginPassword] = useState(false)
  const [loginError, setLoginError] = useState('')
  const [tokenError, setTokenError] = useState('')
  const [isLoggingIn, setIsLoggingIn] = useState(false)
  const credentialInputRef = useRef<HTMLInputElement>(null)
  const activeError = loginMode === 'password' ? loginError : tokenError
  const activeErrorId = loginMode === 'password' ? 'login-password-error' : 'login-token-error'
  const activeNoteId = loginMode === 'password' ? 'login-password-note' : 'login-token-note'
  const credentialDescribedBy = [activeNoteId, activeError ? activeErrorId : ''].filter(Boolean).join(' ') || undefined


  useEffect(() => {
    const frame = window.requestAnimationFrame(() => {
      credentialInputRef.current?.focus()
    })

    return () => window.cancelAnimationFrame(frame)
  }, [loginMode])

  const handleLogin = async () => {
    setIsLoggingIn(true)
    setLoginError('')
    setTokenError('')

    try {
      const { api } = await import('@/lib/api-client')

      // Use protocol 10015 for both login methods
      const params: any = {}

      if (loginMode === 'password') {
        // Master password login
        if (!loginMasterPassword.trim()) {
          setLoginError('Enter your master password.')
          setIsLoggingIn(false)
          return
        }
        params.masterPwd = loginMasterPassword
      } else {
        // Token login
        const trimmedToken = token.trim()

        if (!trimmedToken) {
          setTokenError('Enter an access token.')
          setIsLoggingIn(false)
          return
        }
        params.accessToken = trimmedToken
      }

      const response = await api.auth.login(params)

      if (!response.data?.data?.tokenInfo) {
        if (loginMode === 'password') {
          setLoginError("We couldn't verify that master password. Check it and try again.")
        } else {
          setTokenError("We couldn't verify that access token. Check it and try again.")
        }
        setIsLoggingIn(false)
        return
      }

      const { tokenInfo, accessToken: masterPwdAccessToken } = response.data.data

      // Store userid in localStorage for API requests
      if (tokenInfo.userid && !safeStorageSet('userid', tokenInfo.userid)) {
        throw new Error('Unable to persist user session state in browser storage')
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
        name: tokenInfo.proxyName || 'Kimbap Server',
        proxyName: tokenInfo.proxyName || 'Kimbap Server',
        proxyKey: tokenInfo.proxyKey || '',
        address: `https://server-${tokenInfo.proxyId}.mcp.local`,
        status: 'running',
        role: role,
        userRole: role === 'Owner' ? 'Admin' : role
      }

      if (!safeStorageSet('selectedServer', JSON.stringify(serverInfo))) {
        throw new Error('Unable to persist selected server in browser storage')
      }

      if (loginMode === 'token') {
        if (!safeStorageSet('auth_token', params.accessToken)) {
          throw new Error('Unable to persist auth token in browser storage')
        }
        safeStorageRemove('accessToken')
      } else if (masterPwdAccessToken) {
        if (!safeStorageSet('auth_token', masterPwdAccessToken)) {
          throw new Error('Unable to persist auth token in browser storage')
        }
        safeStorageRemove('accessToken')
      }

      document.cookie = `kimbap_session=${generateSessionCookieValue()}; path=/; max-age=${60 * 60 * 24 * 7}; SameSite=Lax`

      onSuccess()
    } catch (error: unknown) {
      const message = getLoginErrorMessage(error, loginMode)
      if (loginMode === 'password') {
        setLoginError(message)
      } else {
        setTokenError(message)
      }
      setIsLoggingIn(false)
    }
  }

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        void handleLogin()
      }}
      className="w-full max-w-[460px] space-y-3 px-6 py-8 sm:min-h-[480px]"
    >
      <div>
        <h2 className="mb-1 text-2xl font-bold">Sign in to Kimbap Console</h2>
        <p className="text-sm text-muted-foreground">
          {loginMode === 'password'
              ? 'Start here if you own this server. Use the master password to sign in, even from a new browser.'
              : 'Use an owner or admin access token that was issued to you for this server.'}
        </p>
      </div>
      <fieldset className="m-0 border-0 p-0">
        <legend className="sr-only">Login method</legend>
        <div className="grid grid-cols-2 gap-2 rounded-xl border border-border bg-muted/30 p-1" role="radiogroup" aria-label="Login method">
          <label
            className={`flex min-h-11 w-full cursor-pointer items-center justify-center rounded-lg px-4 py-2.5 text-sm transition-colors focus-within:ring-2 focus-within:ring-blue-500 focus-within:ring-offset-2 ${
              loginMode === 'password'
                ? 'bg-background font-semibold text-foreground shadow-sm'
                : 'text-muted-foreground hover:bg-background/60 hover:text-foreground'
            }`}
          >
            <input
              type="radio"
              name="login-method"
              value="password"
              checked={loginMode === 'password'}
              onChange={() => {
                setLoginMode('password')
                setLoginError('')
                setTokenError('')
              }}
              className="sr-only"
            />
            Master Password
          </label>
          <label
            className={`flex min-h-11 w-full cursor-pointer items-center justify-center rounded-lg px-4 py-2.5 text-sm transition-colors focus-within:ring-2 focus-within:ring-blue-500 focus-within:ring-offset-2 ${
              loginMode === 'token'
                ? 'bg-background font-semibold text-foreground shadow-sm'
                : 'text-muted-foreground hover:bg-background/60 hover:text-foreground'
            }`}
          >
            <input
              type="radio"
              name="login-method"
              value="token"
              checked={loginMode === 'token'}
              onChange={() => {
                setLoginMode('token')
                setLoginError('')
                setTokenError('')
              }}
              className="sr-only"
            />
            Access Token
          </label>
        </div>
      </fieldset>

      <p className="text-xs leading-5 text-muted-foreground">
        {loginMode === 'password'
          ? 'Master Password is the default for server owners.'
          : 'Access tokens are for admins and operators who received one from the owner.'}
      </p>

      {/* Input Field */}
      <div className="space-y-1">
        <Label htmlFor="login-credential" className="text-sm font-medium">
          {loginMode === 'password' ? 'Master Password' : 'Access Token'}
          <span className="ml-1 text-xs font-normal text-muted-foreground">(required)</span>
        </Label>
        <div className="relative">
          <Input
            id="login-credential"
            type={
              loginMode === 'password'
                ? showLoginPassword
                  ? 'text'
                  : 'password'
                : 'text'
            }
            placeholder={
              loginMode === 'password'
                ? 'e.g., correct-horse-battery-staple'
                : 'e.g., kimbap_admin_123abc'
            }
            aria-describedby={credentialDescribedBy}
            aria-invalid={Boolean(activeError)}
            ref={credentialInputRef}
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
            className="h-12 rounded-lg px-3 pr-10 text-foreground transition-colors duration-200 focus-visible:border-blue-500 focus-visible:ring-blue-500"
            autoComplete={loginMode === 'password' ? 'current-password' : 'off'}
            autoCapitalize="none"
            autoCorrect="off"
            spellCheck={false}
            required
          />
          {loginMode === 'password' && (
            <button
              type="button"
              onClick={() => setShowLoginPassword(!showLoginPassword)}
              className="absolute right-1 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded text-muted-foreground transition-colors duration-200 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              aria-label={showLoginPassword ? 'Hide password' : 'Show password'}
               disabled={isLoggingIn}
            >

              {showLoginPassword ? (
                <EyeOff className="h-5 w-5" aria-hidden="true" />
              ) : (
                <Eye className="h-5 w-5" aria-hidden="true" />
              )}
            </button>
          )}
        </div>

        {/* Note and Error Messages */}
        {loginMode === 'password' && !loginError && (
          <p id="login-password-note" className="text-xs text-muted-foreground">
            Enter the master password you set when configuring this server.
          </p>
        )}

        {loginMode === 'token' && !tokenError && (
          <p id="login-token-note" className="text-xs text-muted-foreground">
            Paste the token you received from the server owner.
          </p>
        )}

        {loginError && loginMode === 'password' && (
          <Alert id="login-password-error" className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20" role="alert">
            <AlertDescription className="text-sm text-red-800 dark:text-red-200">
              {renderErrorMessageWithLinks(loginError)}
            </AlertDescription>
          </Alert>
        )}

        {tokenError && loginMode === 'token' && (
           <Alert id="login-token-error" className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20" role="alert">
            <AlertDescription className="text-sm text-red-800 dark:text-red-200">
              {tokenError === 'member_link' ? (
                <>
                  Members can use the server but not this console. Sign in with an owner or admin token, or{' '}
                  <a
                    href="https://kimbap.sh/quick-start/#install-desk"
                    target="_blank"
                    rel="noopener noreferrer"
                    aria-label="Download the Kimbap Desk quick start guide (opens in a new tab)"
                    className="rounded-sm text-blue-600 underline hover:text-blue-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 dark:text-blue-400 dark:hover:text-blue-300 cursor-pointer font-medium"
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
        className="h-12 w-full rounded-lg text-base"
        size="lg"
        aria-busy={isLoggingIn}
      >
        {isLoggingIn ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
            Sign in
          </>
        ) : (
          <>
            <LogIn className="w-4 h-4 mr-2" aria-hidden="true" />
            Sign in
          </>
        )}
      </Button>

    </form>
  )
}
