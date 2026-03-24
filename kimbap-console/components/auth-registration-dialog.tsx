"use client"

import { Mail, UserPlus, ArrowLeft, Loader2 } from "lucide-react"
import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"

interface AuthRegistrationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onShowLoginDialog?: () => void
  onRegistrationSuccess?: (user: { email: string; name?: string }) => void
}

export function AuthRegistrationDialog({
  open,
  onOpenChange,
  onShowLoginDialog,
  onRegistrationSuccess,
}: AuthRegistrationDialogProps) {
  const [step, setStep] = useState<"register" | "verify">("register")
  const [email, setEmail] = useState("")
  const [verificationCode, setVerificationCode] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState("")

  const resetForm = () => {
    setStep("register")
    setEmail("")
    setVerificationCode("")
    setError("")
    setIsLoading(false)
  }

  const validateEmail = (email: string) => {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
  }

  const handleGoogleSignUp = async () => {
    setIsLoading(true)
    setError("")

    // Simulate Google OAuth flow
    await new Promise(resolve => setTimeout(resolve, 2000))

    setIsLoading(false)
    onRegistrationSuccess?.({
      email: "user@gmail.com",
      name: "Google User"
    })
    onOpenChange(false)
    resetForm()
  }

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

    // Simulate sending verification code
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

    // Simulate verification
    await new Promise(resolve => setTimeout(resolve, 1000))

    setIsLoading(false)
    onRegistrationSuccess?.({
      email: email,
      name: "New User"
    })
    onOpenChange(false)
    resetForm()
  }

  const handleBackToRegister = () => {
    setStep("register")
    setVerificationCode("")
    setError("")
  }

  const handleSwitchToLogin = () => {
    onOpenChange(false)
    resetForm()
    onShowLoginDialog?.()
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
                onClick={handleBackToRegister}
                aria-label="Back to create account"
              >
                <ArrowLeft className="h-4 w-4" />
              </Button>
            )}
            <UserPlus className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            {step === "register" ? "Create Account" : "Verify Email"}
          </DialogTitle>
          <DialogDescription>
            {step === "register" 
                ? "Create your account."
              : `We've sent a verification code to ${email}`
            }
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {step === "register" ? (
            <>
              {/* Google Sign Up */}
              <div className="space-y-4">
                <Button
                  onClick={handleGoogleSignUp}
                  disabled={isLoading}
                  className="w-full"
                  variant="outline"
                >
                  {isLoading ? (
                    <div className="w-4 h-4 border-2 border-muted-foreground/30 border-t-foreground rounded-full animate-spin mr-2" />
                  ) : (
                    <svg className="mr-2 h-4 w-4" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
                      <path fill="currentColor" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
                      <path fill="currentColor" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
                      <path fill="currentColor" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
                      <path fill="currentColor" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
                    </svg>
                  )}
                  Continue with Google
                </Button>

                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <Separator className="w-full" />
                  </div>
                  <div className="relative flex justify-center text-xs uppercase">
                    <span className="bg-background px-2 text-muted-foreground">
                      Or use email
                    </span>
                  </div>
                </div>
              </div>

              {/* Email Registration */}
              <form
                className="space-y-4"
                onSubmit={(e) => {
                  e.preventDefault()
                  handleSendVerificationCode()
                }}
              >
                <div className="space-y-2">
                  <Label htmlFor="reg-email">Email Address</Label>
                  <Input
                    id="reg-email"
                    type="email"
                    placeholder="your@email.com"
                    value={email}
                    onChange={(e) => {
                      setEmail(e.target.value)
                      setError("")
                    }}
                    disabled={isLoading}
                    className={error ? "border-red-500" : ""}
                    autoFocus
                  />
                </div>

                {error && (
                  <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
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
                      Sending code...
                    </>
                  ) : (
                    <>
                      <Mail className="w-4 h-4 mr-2" />
                      Send Code
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
                    <Mail className="h-8 w-8 text-blue-600 dark:text-blue-400" />
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Enter the 6-digit code sent to your email.
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="verification-code">Verification Code</Label>
                  <Input
                    id="verification-code"
                    type="text"
                    placeholder="000000"
                    value={verificationCode}
                    onChange={(e) => {
                      setVerificationCode(e.target.value.replace(/\D/g, '').slice(0, 6))
                      setError("")
                    }}
                    disabled={isLoading}
                    className={`text-center text-lg tracking-widest ${error ? "border-red-500" : ""}`}
                    maxLength={6}
                  />
                  <p className="text-xs text-muted-foreground text-center">Check your email for the code.</p>
                </div>

                {error && (
                  <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
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
                      Verifying...
                    </>
                  ) : (
                    <>
                      <UserPlus className="w-4 h-4 mr-2" />
                      Complete Sign Up
                    </>
                  )}
                </Button>

                <div className="text-center">
                  <button
                    type="button"
                    onClick={handleSendVerificationCode}
                    className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300"
                    disabled={isLoading}
                  >
                    Didn't receive the code? Resend
                  </button>
                </div>
              </form>
            </>
          )}
        </div>

        <DialogFooter>
          <div className="w-full text-center">
            <p className="text-sm text-muted-foreground">
              Already have an account?{" "}
              <button
                type="button"
                onClick={handleSwitchToLogin}
                className="text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium"
                disabled={isLoading}
              >
                Sign In
              </button>
            </p>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
