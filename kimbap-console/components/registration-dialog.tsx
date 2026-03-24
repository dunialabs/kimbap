"use client"

import { Mail, Eye, EyeOff, CheckCircle, UserPlus, Loader2 } from "lucide-react"
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

interface RegistrationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onShowLoginDialog?: () => void
  onRegistrationSuccess?: (user: { email: string; name: string }) => void
}

export function RegistrationDialog({
  open,
  onOpenChange,
  onShowLoginDialog,
  onRegistrationSuccess,
}: RegistrationDialogProps) {
  const [registrationMethod, setRegistrationMethod] = useState<"email" | "gmail">("email")
  const [formData, setFormData] = useState({
    name: "",
    email: "",
    password: "",
    confirmPassword: "",
  })
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const [isRegistering, setIsRegistering] = useState(false)
  const [errors, setErrors] = useState<{ [key: string]: string }>({})

  const resetForm = () => {
    setFormData({
      name: "",
      email: "",
      password: "",
      confirmPassword: "",
    })
    setErrors({})
    setShowPassword(false)
    setShowConfirmPassword(false)
  }

  const validateForm = () => {
    const newErrors: { [key: string]: string } = {}

    if (!formData.name.trim()) {
      newErrors.name = "Name is required"
    }

    if (!formData.email.trim()) {
      newErrors.email = "Email is required"
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = "Please enter a valid email address"
    }

    if (registrationMethod === "email") {
      if (!formData.password) {
        newErrors.password = "Password is required"
      } else if (formData.password.length < 8) {
        newErrors.password = "Password must be at least 8 characters"
      }

      if (!formData.confirmPassword) {
        newErrors.confirmPassword = "Please confirm your password"
      } else if (formData.password !== formData.confirmPassword) {
        newErrors.confirmPassword = "Passwords do not match"
      }
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleEmailRegistration = async () => {
    if (!validateForm()) return

    setIsRegistering(true)

    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 2000))

    setIsRegistering(false)
    onRegistrationSuccess?.({
      email: formData.email,
      name: formData.name,
    })
    onOpenChange(false)
    resetForm()
  }

  const handleGmailRegistration = async () => {
    if (!formData.name.trim() || !formData.email.trim()) {
      setErrors({
        name: !formData.name.trim() ? "Name is required" : "",
        email: !formData.email.trim() ? "Email is required" : "",
      })
      return
    }

    setIsRegistering(true)

    // Simulate OAuth flow
    await new Promise(resolve => setTimeout(resolve, 1500))

    setIsRegistering(false)
    onRegistrationSuccess?.({
      email: formData.email,
      name: formData.name,
    })
    onOpenChange(false)
    resetForm()
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
            <UserPlus className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            Create Account
          </DialogTitle>
          <DialogDescription>
            Create your account.
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (registrationMethod === "email") {
              handleEmailRegistration()
              return
            }
            handleGmailRegistration()
          }}
        >
          <div className="space-y-6">
          {/* Registration Method Toggle */}
          <div className="flex bg-muted rounded-lg p-1">
            <button
              type="button"
              onClick={() => setRegistrationMethod("email")}
              className={`flex-1 py-2 px-4 rounded-md text-sm font-medium transition-colors ${
                registrationMethod === "email"
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <Mail className="mr-2 h-4 w-4 inline" />
              Email
            </button>
            <button
              type="button"
              onClick={() => setRegistrationMethod("gmail")}
              className={`flex-1 py-2 px-4 rounded-md text-sm font-medium transition-colors ${
                registrationMethod === "gmail"
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <Mail className="mr-2 h-4 w-4 inline" />
              Gmail
            </button>
          </div>

          {/* Registration Form */}
          <div className="space-y-4">
            {/* Name Field */}
            <div className="space-y-2">
              <Label htmlFor="reg-name">Full Name *</Label>
              <Input
                id="reg-name"
                placeholder="Jane Doe"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                disabled={isRegistering}
                className={errors.name ? "border-red-500" : ""}
                autoFocus
              />
              {errors.name && (
                <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
                  <AlertDescription className="text-sm text-red-800 dark:text-red-200">
                    {errors.name}
                  </AlertDescription>
                </Alert>
              )}
            </div>

            {/* Email Field */}
            <div className="space-y-2">
              <Label htmlFor="reg-email">Email Address *</Label>
              <Input
                id="reg-email"
                type="email"
                placeholder={registrationMethod === "gmail" ? "your-gmail@gmail.com" : "your@email.com"}
                value={formData.email}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                disabled={isRegistering}
                className={errors.email ? "border-red-500" : ""}
              />
              {errors.email && (
                <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
                  <AlertDescription className="text-sm text-red-800 dark:text-red-200">
                    {errors.email}
                  </AlertDescription>
                </Alert>
              )}
            </div>

            {/* Password Fields (only for email registration) */}
            {registrationMethod === "email" && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="reg-password">Password *</Label>
                  <div className="relative">
                    <Input
                      id="reg-password"
                      type={showPassword ? "text" : "password"}
                      placeholder="At least 8 characters"
                      value={formData.password}
                      onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                      disabled={isRegistering}
                      className={errors.password ? "border-red-500 pr-10" : "pr-10"}
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                      disabled={isRegistering}
                      aria-label={showPassword ? 'Hide password' : 'Show password'}
                    >

                      {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </button>
                  </div>
                  {errors.password && (
                    <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
                      <AlertDescription className="text-sm text-red-800 dark:text-red-200">
                        {errors.password}
                      </AlertDescription>
                    </Alert>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="reg-confirm-password">Confirm Password *</Label>
                  <div className="relative">
                    <Input
                      id="reg-confirm-password"
                      type={showConfirmPassword ? "text" : "password"}
                      placeholder="Repeat password"
                      value={formData.confirmPassword}
                      onChange={(e) => setFormData({ ...formData, confirmPassword: e.target.value })}
                      disabled={isRegistering}
                      className={errors.confirmPassword ? "border-red-500 pr-10" : "pr-10"}
                    />
                    <button
                      type="button"
                      onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                      className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                      disabled={isRegistering}
                      aria-label={showConfirmPassword ? 'Hide password' : 'Show password'}
                    >

                      {showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </button>
                  </div>
                  {errors.confirmPassword && (
                    <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
                      <AlertDescription className="text-sm text-red-800 dark:text-red-200">
                        {errors.confirmPassword}
                      </AlertDescription>
                    </Alert>
                  )}
                </div>
              </>
            )}

            {/* Gmail Registration Note */}
            {registrationMethod === "gmail" && (
              <div className="p-3 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-md">
                <div className="flex items-start gap-2">
                  <CheckCircle className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                      Gmail Sign Up
                    </p>
                    <p className="text-xs text-blue-700 dark:text-blue-200 mt-1">
                      Use Google OAuth to create an account without a password.
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
          </div>

          <DialogFooter className="flex-col space-y-3">
            <Button
              type="submit"
              disabled={isRegistering}
              className="w-full"
            >
            {isRegistering ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating account...
              </>
            ) : (
              <>
                <UserPlus className="w-4 h-4 mr-2" />
                {registrationMethod === "gmail" ? "Create with Gmail" : "Create Account"}
              </>
            )}
            </Button>

            <Separator />

            <div className="text-center">
              <p className="text-sm text-muted-foreground">
                Already have an account?{" "}
                <button
                  type="button"
                  onClick={handleSwitchToLogin}
                  className="text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium"
                  disabled={isRegistering}
                >
                  Sign In
                </button>
              </p>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
