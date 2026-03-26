'use client'

import Image from 'next/image'
import { useRouter, useSearchParams } from 'next/navigation'
import { Suspense, useEffect, useState } from 'react'

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
    <div className="flex min-h-screen p-[24px] pb-0 bg-[#F7F7F3] dark:bg-background flex-col">
      <div className="flex flex-1">
        <div className="hidden lg:flex lg:w-1/2 p-12 flex-col justify-center items-center">
          <div className="max-w-[780px]">
            <h1 className="text-[52px] font-bold text-[#F56711] leading-[60px] mb-[4px]">Kimbap Console</h1>
            <h2 className="text-[40px] font-bold text-[#26251E] dark:text-foreground leading-[48px] mb-[24px]">
              Operations Console
            </h2>
            <p className="text-muted-foreground leading-[24px] text-[16px]">
              Review logs, handle approvals, manage policies, and monitor usage from one place.
            </p>
          </div>
        </div>

        <div className="flex-1 flex bg-white dark:bg-slate-900 rounded-[12px]">
          <div className="w-full flex flex-col">
            <div className="p-[14px]">
              <Image src="/new_logo.svg" alt="Kimbap Logo" width={226} height={32} className="block dark:hidden" priority />
              <Image src="/darklogo.svg" alt="Kimbap Logo" width={226} height={32} className="hidden dark:block" priority />
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

      <footer className="w-full py-4 border-t border-slate-200 dark:border-slate-800">
        <div className="text-center text-xs text-slate-500 dark:text-slate-400">
          <span>© 2026 </span>
          <a
            href="https://kimbap.sh"
            target="_blank"
            rel="noopener noreferrer"
            aria-label="Dunia Labs website (opens in new tab)"
            className="rounded-sm text-slate-600 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 dark:text-slate-300"
          >
            Dunia Labs, Inc.
          </a>
          <span> Operations console for the Kimbap platform.</span>
        </div>
      </footer>
    </div>
  )
}

export default function WelcomePage() {
  return (
    <Suspense>
      <WelcomePageContent />
    </Suspense>
  )
}
