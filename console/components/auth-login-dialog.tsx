'use client'

import { Mail, LogIn, ArrowLeft, Loader2 } from 'lucide-react'
import { useRef, useState, useEffect } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
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

interface AuthLoginDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onLoginSuccess?: (user: { email: string; name?: string }) => void
}

export function AuthLoginDialog({
  open,
  onOpenChange,
  onLoginSuccess,
}: AuthLoginDialogProps) {
  const [step, setStep] = useState<'login' | 'verify'>('login')
  const [email, setEmail] = useState('')
  const [verificationCode, setVerificationCode] = useState('')
  const [isSendingCode, setIsSendingCode] = useState(false)
  const [isVerifyingCode, setIsVerifyingCode] = useState(false)
  const [error, setError] = useState('')
  const emailInputRef = useRef<HTMLInputElement>(null)
  const verificationCodeInputRef = useRef<HTMLInputElement>(null)

  function validateEmail(email: string) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
  }

  const trimmedEmail = email.trim()
  const emailFormatInvalid = step === 'login' && trimmedEmail.length > 0 && !validateEmail(trimmedEmail)
  const verificationCodeIncomplete = step === 'verify' && verificationCode.length > 0 && verificationCode.length < 6

  const resetForm = () => {
    setStep('login')
    setEmail('')
    setVerificationCode('')
    setError('')
    setIsSendingCode(false)
    setIsVerifyingCode(false)
  }

  useEffect(() => {
    if (!open) {
      return
    }

    const frame = window.requestAnimationFrame(() => {
      if (step === 'login') {
        emailInputRef.current?.focus()
        return
      }

      verificationCodeInputRef.current?.focus()
    })

    return () => window.cancelAnimationFrame(frame)
  }, [open, step])

  const handleSendVerificationCode = async () => {
    if (!email.trim()) {
      setError('Enter your email address.')
      return
    }

    if (!validateEmail(email)) {
      setError('Enter a valid email address.')
      return
    }

    setIsSendingCode(true)
    setError('')

    await new Promise((resolve) => setTimeout(resolve, 1500))
    setIsSendingCode(false)
    setStep('verify')
  }

  const handleVerifyCode = async () => {
    if (!verificationCode.trim()) {
      setError('Enter the verification code.')
      return
    }

    setIsVerifyingCode(true)
    setError('')

    try {
      await new Promise((resolve) => setTimeout(resolve, 1000))

      setIsVerifyingCode(false)
      onLoginSuccess?.({ email })
      onOpenChange(false)
      resetForm()
    } catch (error: any) {
      setIsVerifyingCode(false)
      setError(error.response?.data?.error || "That code didn't work. Check the latest email and try again.")
    }
  }

  const handleBackToLogin = () => {
    setStep('login')
    setVerificationCode('')
    setError('')
  }

  const handleDialogOpenChange = (nextOpen: boolean) => {
    onOpenChange(nextOpen)
    if (!nextOpen) resetForm()
  }

  return (
    <Dialog open={open} onOpenChange={handleDialogOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader className="space-y-3">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {step === 'login' ? 'Step 1 of 2' : 'Step 2 of 2'}
          </p>
          {step === 'verify' ? (
            <Button
              variant="ghost"
              size="sm"
              className="-ml-2 h-8 w-fit px-2 text-sm"
              onClick={handleBackToLogin}
            >
              <ArrowLeft className="mr-1 h-4 w-4" />
              Back
            </Button>
          ) : null}
          <DialogTitle className="flex items-center gap-2 text-2xl">
            <LogIn className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            {step === 'login' ? 'Sign in' : 'Enter code'}
          </DialogTitle>
          <DialogDescription>
            {step === 'login'
              ? 'Enter your email to receive a sign-in code.'
              : `Check ${email} for the latest verification code.`}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {step === 'login' ? (
            <form
              className="space-y-4"
              onSubmit={(e) => {
                e.preventDefault()
                void handleSendVerificationCode()
              }}
            >
              <div className="space-y-2">
                <Label htmlFor="login-email">
                  Email address
                  <span className="ml-1 text-xs font-normal text-muted-foreground">(required)</span>
                </Label>
                <Input
                  id="login-email"
                  type="email"
                  placeholder="your@email.com"
                  ref={emailInputRef}
                  value={email}
                  onChange={(e) => {
                    setEmail(e.target.value)
                    setError('')
                  }}
                  disabled={isSendingCode || isVerifyingCode}
                  aria-invalid={Boolean(error) || emailFormatInvalid}
                  aria-describedby={error || emailFormatInvalid ? 'auth-login-error auth-login-note' : 'auth-login-note'}
                  autoCapitalize="none"
                  autoCorrect="off"
                  spellCheck={false}
                  autoComplete="email"
                  className={error || emailFormatInvalid ? 'border-red-500' : ''}
                  required
                />
              </div>

              <p id="auth-login-note" className="text-xs text-muted-foreground">
                We’ll email a one-time code to this address.
              </p>

              {(error || emailFormatInvalid) && (
                <Alert id="auth-login-error" className="border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/20" role="alert">
                  <AlertDescription className="text-sm text-red-800 dark:text-red-200">
                    {error || 'Enter a valid email address.'}
                  </AlertDescription>
                </Alert>
              )}

              <Button
                type="submit"
                disabled={isSendingCode || isVerifyingCode || !trimmedEmail || emailFormatInvalid}
                className="w-full"
              >
                {isSendingCode ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Send sign-in code
                  </>
                ) : (
                  <>
                    <Mail className="mr-2 h-4 w-4" />
                    Send sign-in code
                  </>
                )}
              </Button>
            </form>
          ) : (
            <form
              className="space-y-4"
              onSubmit={(e) => {
                e.preventDefault()
                void handleVerifyCode()
              }}
            >
              <div className="text-center">
                <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-950">
                  <Mail className="h-8 w-8 text-blue-600 dark:text-blue-400" aria-hidden="true" />
                </div>
                <p className="text-sm text-muted-foreground">
                  Enter the 6-digit code from your email.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="verification-code">
                  Verification code
                  <span className="ml-1 text-xs font-normal text-muted-foreground">(required)</span>
                </Label>
                <Input
                  id="verification-code"
                  type="text"
                  placeholder="000000"
                  ref={verificationCodeInputRef}
                  value={verificationCode}
                  onChange={(e) => {
                    setVerificationCode(e.target.value.replace(/\D/g, '').slice(0, 6))
                    setError('')
                  }}
                  disabled={isSendingCode || isVerifyingCode}
                  aria-invalid={Boolean(error) || verificationCodeIncomplete}
                  aria-describedby={error || verificationCodeIncomplete ? 'auth-verify-error auth-verify-note' : 'auth-verify-note'}
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  className={`text-center text-lg tracking-widest ${error || verificationCodeIncomplete ? 'border-red-500' : ''}`}
                  maxLength={6}
                  pattern="[0-9]{6}"
                  required
                />
                <p id="auth-verify-note" className="text-center text-xs text-muted-foreground">
                  Check your email for the code.
                </p>
              </div>

              {(error || verificationCodeIncomplete) && (
                <Alert id="auth-verify-error" className="border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/20" role="alert">
                  <AlertDescription className="text-sm text-red-800 dark:text-red-200">
                    {error || 'Enter the full 6-digit verification code.'}
                  </AlertDescription>
                </Alert>
              )}

              <Button
                type="submit"
                disabled={isSendingCode || isVerifyingCode || verificationCode.length !== 6}
                className="w-full"
              >
                {isVerifyingCode ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Verify code
                  </>
                ) : (
                  <>
                    <LogIn className="mr-2 h-4 w-4" />
                    Verify code
                  </>
                )}
              </Button>

              <div className="text-center">
                <button
                  type="button"
                  onClick={() => void handleSendVerificationCode()}
                  className="rounded-sm text-sm font-medium text-blue-600 transition-colors duration-200 hover:text-blue-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:text-blue-400 dark:hover:text-blue-300"
                  disabled={isSendingCode || isVerifyingCode}
                >
                  Resend code
                </button>
              </div>
            </form>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
