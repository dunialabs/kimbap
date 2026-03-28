'use client'

import { useSearchParams, useRouter } from 'next/navigation'
import { Suspense, useEffect } from 'react'

function RedirectFallback() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-muted px-4" role="status" aria-live="polite" aria-atomic="true" aria-busy="true">
      <div className="text-center">
        <div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground" aria-hidden="true" />
        <h1 className="text-lg font-semibold">Redirecting to sign in</h1>
        <p className="text-sm text-muted-foreground">Taking you to the sign-in screen…</p>
      </div>
    </div>
  )
}

function LoginRedirect() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const redirectTo = searchParams.get('redirect')

  useEffect(() => {
    const target = redirectTo ? `/?redirect=${encodeURIComponent(redirectTo)}` : '/'
    router.replace(target)
  }, [redirectTo, router])

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted px-4" role="status" aria-live="polite" aria-atomic="true" aria-busy="true">
      <div className="text-center">
        <div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground" aria-hidden="true" />
        <h1 className="text-lg font-semibold">Redirecting to sign in</h1>
        <p className="text-sm text-muted-foreground">Taking you to the sign-in screen…</p>
      </div>
    </div>
  )
}

export default function LoginPage() {
  return (
    <Suspense fallback={<RedirectFallback />}>
      <LoginRedirect />
    </Suspense>
  )
}
