"use client"

import { Mail, LogIn, ArrowLeft, Loader2 } from "lucide-react"
import { useRef, useState, useEffect } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

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
  const [step, setStep] = useState<"login" | "verify">("login")
  const [email, setEmail] = useState("")
  const [verificationCode, setVerificationCode] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState("")
  const emailInputRef = useRef<HTMLInputElement>(null)
  const verificationCodeInputRef = useRef<HTMLInputElement>(null)

  const resetForm = () => {
    setStep("login")
    setEmail("")
    setVerificationCode("")
    setError("")
    setIsLoading(false)
  }

  const validateEmail = (email: string) => {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
  }

  useEffect(() => {
    if (!open) {
      return
    }

    const frame = window.requestAnimationFrame(() => {
      if (step === "login") {
        emailInputRef.current?.focus()
        return
      }

      verificationCodeInputRef.current?.focus()
    })

    return () => window.cancelAnimationFrame(frame)
  }, [open, step])

  const handleSendVerificationCode = async () => {
    if (!email.trim()) {
      setError("Please enter your email address")
      return
    }

    if (!validateEmail(email)) {
      setError("Please enter a valid email address")
      return
    }

    setIsLoading(true)
    setError("")

    await new Promise(resolve => setTimeout(resolve, 1500))
    setIsLoading(false)
    setStep("verify")
  }

  const handleVerifyCode = async () => {
    if (!verificationCode.trim()) {
      setError("Please enter the verification code")
      return
    }

    setIsLoading(true)
    setError("")

    try {
      await new Promise(resolve => setTimeout(resolve, 1000))

      setIsLoading(false)
      onLoginSuccess?.({ email })
      onOpenChange(false)
      resetForm()
    } catch (error: any) {
      setIsLoading(false)
      setError(error.response?.data?.error || "That code didn't work. Check the latest email and try again.")
    }
  }

  const handleBackToLogin = () => {
    setStep("login")
    setVerificationCode("")
    setError("")
  }

  return (
    <Dialog open={open} onOpenChange={(open) => {
      onOpenChange(open)
      if (!open) resetForm()
    }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="text-2xl flex items-center gap-2">
            {step === "verify" && (
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 -ml-2"
                onClick={handleBackToLogin}
                aria-label="Back to sign in"
              >
                <ArrowLeft className="h-4 w-4" />
              </Button>
            )}
            <LogIn className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            {step === "login" ? "Sign in" : "Enter code"}
          </DialogTitle>
          <DialogDescription>
            {step === "login" 
              ? "Enter your email to receive a sign-in code."
              : `Check ${email} for the latest verification code.`
            }
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {step === "login" ? (
            <>
              {/* Email Login */}
              <form
                className="space-y-4"
                onSubmit={(e) => {
                  e.preventDefault()
                  handleSendVerificationCode()
                }}
              >
                <div className="space-y-2">
                  <Label htmlFor="login-email">Email address</Label>
                  <Input
                    id="login-email"
                    type="email"
                    placeholder="your@email.com"
                    ref={emailInputRef}
                    value={email}
                    onChange={(e) => {
                      setEmail(e.target.value)
                      setError("")
                    }}
                    disabled={isLoading}
                    aria-invalid={Boolean(error)}
                    aria-describedby={error ? 'auth-login-error' : 'auth-login-note'}
                    autoCapitalize="none"
                    autoCorrect="off"
                    spellCheck={false}
                    autoComplete="email"
                    className={error ? "border-red-500" : ""}
                  />
                </div>

                <p id="auth-login-note" className="text-xs text-muted-foreground">
                  We’ll email a one-time code to this address.
                </p>

                {error && (
                  <Alert id="auth-login-error" className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
                    <AlertDescription className="text-sm text-red-800 dark:text-red-200">
                      {error}
                    </AlertDescription>
                  </Alert>
                )}

                <Button
                  type="submit"
                  disabled={isLoading || !email.trim()}
                  className="w-full"
                >
                  {isLoading ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Sending sign-in code...
                    </>
                  ) : (
                    <>
                      <Mail className="w-4 h-4 mr-2" />
                      Send sign-in code
                    </>
                  )}
                </Button>
              </form>
            </>
          ) : (
            <>
              {/* Verification Code Input */}
              <form
                className="space-y-4"
                onSubmit={(e) => {
                  e.preventDefault()
                  handleVerifyCode()
                }}
              >
                <div className="text-center">
                  <div className="w-16 h-16 bg-blue-100 dark:bg-blue-950 rounded-full flex items-center justify-center mx-auto mb-4">
                    <Mail className="h-8 w-8 text-blue-600 dark:text-blue-400" aria-hidden="true" />
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Enter the 6-digit code from your email.
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="verification-code">Verification code</Label>
                  <Input
                    id="verification-code"
                    type="text"
                    placeholder="000000"
                    ref={verificationCodeInputRef}
                    value={verificationCode}
                    onChange={(e) => {
                      setVerificationCode(e.target.value.replace(/\D/g, '').slice(0, 6))
                      setError("")
                    }}
                    disabled={isLoading}
                    aria-invalid={Boolean(error)}
                    aria-describedby={error ? 'auth-verify-error' : 'auth-verify-note'}
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    className={`text-center text-lg tracking-widest ${error ? "border-red-500" : ""}`}
                    maxLength={6}
                  />
                  <p id="auth-verify-note" className="text-xs text-muted-foreground text-center">
                    Check your email for the code.
                  </p>
                </div>

                {error && (
                  <Alert id="auth-verify-error" className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
                    <AlertDescription className="text-sm text-red-800 dark:text-red-200">
                      {error}
                    </AlertDescription>
                  </Alert>
                )}

                <Button
                  type="submit"
                  disabled={isLoading || verificationCode.length !== 6}
                  className="w-full"
                >
                  {isLoading ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Verifying code...
                    </>
                  ) : (
                    <>
                      <LogIn className="w-4 h-4 mr-2" />
                      Sign in
                    </>
                  )}
                </Button>

                <div className="text-center">
                  <button
                    type="button"
                    onClick={handleSendVerificationCode}
                    className="rounded-sm text-sm text-blue-600 hover:text-blue-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 dark:text-blue-400 dark:hover:text-blue-300"
                    disabled={isLoading}
                  >
                    Resend code
                  </button>
                </div>
              </form>
            </>
          )}
        </div>


      </DialogContent>
    </Dialog>
  )
}
