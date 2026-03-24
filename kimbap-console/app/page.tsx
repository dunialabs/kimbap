'use client'

import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'

import { LoginForm } from '@/components/login-form'

export default function WelcomePage() {
  const router = useRouter()
  const [checkingAuth, setCheckingAuth] = useState(true)

  useEffect(() => {
    const userid = localStorage.getItem('userid')
    const authToken = localStorage.getItem('auth_token')

    if (userid && authToken) {
      router.push('/dashboard')
      return
    }

    if (userid && !authToken) {
      localStorage.removeItem('userid')
    }

    setCheckingAuth(false)
  }, [router])

  if (checkingAuth) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground" />
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
              Audit activity, handle approvals, manage policies, and monitor usage from one place.
            </p>
          </div>
        </div>

        <div className="flex-1 flex bg-white dark:bg-slate-900 rounded-[12px]">
          <div className="w-full flex flex-col">
            <div className="p-[14px]">
              <img src="/new_logo.svg" alt="Kimbap Logo" className="block dark:hidden" />
              <img src="/darklogo.svg" alt="Kimbap Logo" className="hidden dark:block" />
            </div>
            <div className="flex-1 flex flex-col justify-center items-center">
              <LoginForm
                onSuccess={() => {
                  router.push('/dashboard')
                }}
                onManualConnect={() => {}}
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
            className="hover:underline text-slate-600 dark:text-slate-300"
          >
            Dunia Labs, Inc.
          </a>
          <span> Operations console for approvals, policies, logs, and usage.</span>
        </div>
      </footer>
    </div>
  )
}
