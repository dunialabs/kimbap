'use client'

import { useSearchParams, useRouter } from 'next/navigation'
import { Suspense, useEffect } from 'react'

function LoginRedirect() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const redirectTo = searchParams.get('redirect')

  useEffect(() => {
    const target = redirectTo ? `/?redirect=${encodeURIComponent(redirectTo)}` : '/'
    router.replace(target)
  }, [redirectTo, router])

  return null
}

export default function LoginPage() {
  return (
    <Suspense>
      <LoginRedirect />
    </Suspense>
  )
}
