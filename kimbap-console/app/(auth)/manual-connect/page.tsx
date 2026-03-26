'use client'

import { useRouter } from 'next/navigation'
import { useEffect } from 'react'

export default function ManualConnectPage() {
  const router = useRouter()

  useEffect(() => {
    router.replace('/')
  }, [router])

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted px-4" role="status" aria-live="polite">
      <div className="text-center">
        <div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground" aria-hidden="true" />
        <h1 className="text-lg font-semibold">Redirecting to manual connect</h1>
        <p className="text-sm text-muted-foreground">Taking you to the current console entry point…</p>
      </div>
    </div>
  )
}
