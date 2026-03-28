"use client"

import { useRouter } from "next/navigation"

import { AuthLayout } from "@/components/auth-layout"
import { MasterPasswordForm } from "@/components/master-password-form"

export default function MasterPasswordPage() {
  const router = useRouter()

  return (
    <AuthLayout>
      <div className="w-full max-w-[460px] space-y-4">
        <div className="rounded-xl border border-blue-200/70 bg-blue-50/80 p-4 text-sm dark:border-blue-900/70 dark:bg-blue-950/20">
          <h1 className="text-base font-semibold text-foreground">What is the master password?</h1>
          <p className="mt-2 leading-6 text-muted-foreground">
            It is the owner sign-in for this console. You will use it to unlock owner access when managing policies, approvals, logs, and usage from the browser.
          </p>
          <ul className="mt-3 space-y-2 leading-6 text-muted-foreground">
            <li>• Owners use this password to manage the console.</li>
            <li>• Admins and other operators usually sign in with access tokens instead.</li>
            <li>• Choose a password you can store safely because it cannot be recovered.</li>
          </ul>
        </div>

        <MasterPasswordForm onSuccess={() => router.replace('/')} />
      </div>
    </AuthLayout>
  )
}
