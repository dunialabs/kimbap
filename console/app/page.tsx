'use client'

import Image from 'next/image'
import { useRouter, useSearchParams } from 'next/navigation'
import { Suspense, useEffect, useState } from 'react'

import { GlobalFooter } from '@/components/global-footer'
import { LoginForm } from '@/components/login-form'
import { clearAuthState } from '@/lib/api-client'

const safeStorageGet = (key: string): string | null => {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

function WelcomePageContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [checkingAuth, setCheckingAuth] = useState(true)

  useEffect(() => {
    const userid = safeStorageGet('userid')
    const authToken = safeStorageGet('auth_token')
    const hasSessionCookie = document.cookie
      .split('; ')
      .some((cookie) => cookie.startsWith('kimbap_session='))

    if (userid && authToken && hasSessionCookie) {
      router.push('/dashboard')
      return
    }

    if (userid && (!authToken || !hasSessionCookie)) {
      clearAuthState()
    }

    setCheckingAuth(false)
  }, [router])

  if (checkingAuth) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted px-4" role="status" aria-live="polite">
        <div className="text-center">
          <div
            className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground"
            aria-hidden="true"
          />
          <h1 className="text-lg font-semibold">Checking your session</h1>
          <p className="text-sm text-muted-foreground">Confirming whether this browser is already connected…</p>
        </div>
      </div>
    )
  }

  return (
    <main className="flex min-h-screen flex-col bg-gradient-to-br from-orange-50 via-background to-amber-50/70 px-4 pb-0 pt-4 dark:from-background dark:via-background dark:to-background sm:px-6 sm:pt-6">
      <div className="flex flex-1">
        <div className="hidden lg:flex lg:w-1/2 lg:flex-col lg:justify-center lg:p-12">
          <div className="max-w-md">
            <h1 className="mb-1 text-[52px] font-bold leading-tight text-orange-600 dark:text-orange-400">
              Kimbap Console
            </h1>
            <h2 className="mb-6 text-[40px] font-bold leading-tight text-slate-900 dark:text-foreground xl:text-[52px]">
              Operations Console
            </h2>
            <p className="text-[16px] leading-relaxed text-muted-foreground">
              Review logs, handle approvals, manage policies, and monitor usage from one place.
            </p>
          </div>
        </div>

        <div className="flex-1 rounded-xl border border-border/60 bg-card shadow-sm">
          <div className="w-full flex flex-col">
            <div className="p-4">
              <Image src="/new_logo.svg" alt="Kimbap Logo" width={226} height={32} className="block h-auto max-w-full dark:hidden" priority />
              <Image src="/darklogo.svg" alt="Kimbap Logo" width={226} height={32} className="hidden h-auto max-w-full dark:block" priority />
            </div>
            <div className="flex-1 flex flex-col justify-center items-center">
              <LoginForm
                onSuccess={() => {
                  const redirectTo = searchParams.get('redirect')
                  router.push(redirectTo?.startsWith('/dashboard') ? redirectTo : '/dashboard')
                }}
              />
            </div>
          </div>
        </div>
      </div>

      <GlobalFooter />
    </main>
  )
}

export default function WelcomePage() {
  return (
    <Suspense>
      <WelcomePageContent />
    </Suspense>
  )
}
