'use client'

import { useRouter } from 'next/navigation'
import { useEffect } from 'react'

export default function ManualConnectPage() {
  const router = useRouter()

  useEffect(() => {
    const redirectTimer = window.setTimeout(() => {
      router.replace('/')
    }, 1400)

    return () => window.clearTimeout(redirectTimer)
  }, [router])

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted px-4" role="status" aria-live="polite" aria-atomic="true" aria-busy="true">
      <div className="text-center">
        <div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground" aria-hidden="true" />
        <h1 className="text-lg font-semibold">Manual connect moved</h1>
        <p className="text-sm text-muted-foreground">Manual connect has moved to the main sign-in screen. Open the console URL directly in your browser — there is no separate server URL field anymore.</p>
      </div>
    </div>
  )
}
