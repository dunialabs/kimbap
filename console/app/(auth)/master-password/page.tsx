'use client'

import { useRouter } from 'next/navigation'

import { AuthLayout } from '@/components/auth-layout'
import { MasterPasswordForm } from '@/components/master-password-form'

export default function MasterPasswordPage() {
  const router = useRouter()

  return (
    <AuthLayout>
      <div className="w-full max-w-[460px] space-y-4">
        <div className="rounded-xl border border-blue-200/70 bg-blue-50/80 p-4 text-sm dark:border-blue-900/70 dark:bg-blue-950/20">
          <h1 className="text-base font-semibold text-foreground">What is the master password?</h1>
          <p className="mt-2 leading-6 text-muted-foreground">
            The master password is your owner credential for this console. You'll use it to manage policies, approvals, logs, and usage from the browser.
          </p>
          <ul className="mt-3 list-disc space-y-2 pl-5 leading-6 text-muted-foreground">
            <li><span className="font-medium text-foreground">Owner-only:</span> Only owners can set or reset the master password.</li>
            <li><span className="font-medium text-foreground">For admins/operators:</span> Sign in with access tokens instead.</li>
            <li><span className="font-medium text-foreground">Important:</span> Store it safely — it cannot be recovered.</li>
          </ul>
        </div>

        <MasterPasswordForm onSuccess={() => router.replace('/')} />
      </div>
    </AuthLayout>
  )
}
