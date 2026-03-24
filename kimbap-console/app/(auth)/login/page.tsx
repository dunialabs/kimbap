"use client"

import { useRouter } from "next/navigation"

import { useState } from "react"
import Image from "next/image"
import { LogIn, UserPlus } from "lucide-react"

import { AuthLoginDialog } from "@/components/auth-login-dialog"
import { AuthRegistrationDialog } from "@/components/auth-registration-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export default function LoginPage() {
  const router = useRouter()
  const [showLoginDialog, setShowLoginDialog] = useState(false)
  const [showRegistrationDialog, setShowRegistrationDialog] = useState(false)


  return (
    <div className="flex items-center justify-center min-h-screen relative">
      <Card className="w-full max-w-sm mx-4 backdrop-blur-xl bg-white/90 dark:bg-slate-900/90 border-white/20 dark:border-slate-800/50">
        <CardHeader className="text-center">
          <div className="flex items-center justify-center mb-4">
            <Image
              src="/logo-icon.png"
              alt="Kimbap"
              width={64}
              height={64}
              className="h-16 w-16"
            />
          </div>
          <CardTitle className="text-2xl font-bold">Welcome Back</CardTitle>
          <CardDescription>Sign in to access your MCP Console account</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <Button 
              onClick={() => setShowLoginDialog(true)}
              className="w-full bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white shadow-lg"
              size="lg"
            >
              <LogIn className="mr-2 h-5 w-5" />
              Sign In to Your Account
            </Button>
            
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t border-slate-200/50 dark:border-slate-700/50" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-white dark:bg-slate-900 px-2 text-muted-foreground">Or</span>
              </div>
            </div>
            
            <Button
              onClick={() => setShowRegistrationDialog(true)}
              variant="outline"
              className="w-full border-slate-200/50 dark:border-slate-700/50"
              size="lg"
            >
              <UserPlus className="mr-2 h-5 w-5" />
              Create New Account
            </Button>
            
            <p className="text-xs text-center text-muted-foreground">
              By signing in, you agree to our{" "}
              <a href="/terms" className="text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium">
                Terms of Service
              </a>{" "}
              and{" "}
              <a href="/privacy" className="text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium">
                Privacy Policy
              </a>
            </p>
          </div>
        </CardContent>
      </Card>
      
      <AuthLoginDialog
        open={showLoginDialog}
        onOpenChange={setShowLoginDialog}
        onShowRegistrationDialog={() => {
          setShowLoginDialog(false)
          setShowRegistrationDialog(true)
        }}
        onLoginSuccess={() => {
          router.push("/master-password")
        }}
      />
      
      <AuthRegistrationDialog
        open={showRegistrationDialog}
        onOpenChange={setShowRegistrationDialog}
        onShowLoginDialog={() => {
          setShowRegistrationDialog(false)
          setShowLoginDialog(true)
        }}
        onRegistrationSuccess={() => {
          router.push("/master-password")
        }}
      />
    </div>
  )
}
