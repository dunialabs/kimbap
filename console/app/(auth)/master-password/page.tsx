'use client'

import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'

import { AuthLayout } from '@/components/auth-layout'
import { MasterPasswordForm } from '@/components/master-password-form'
import { api } from '@/lib/api-client'

export default function MasterPasswordPage() {
  const router = useRouter()
  const [isChecking, setIsChecking] = useState(true)
  const [isInitialized, setIsInitialized] = useState(false)

  useEffect(() => {
    let mounted = true

    const checkInit = async () => {
      try {
        const response = await api.auth.checkInitStatus()
        if (!mounted) return

        if (response.data?.data?.initialized) {
          setIsInitialized(true)
          router.replace('/')
        } else {
          setIsChecking(false)
        }
      } catch {
        if (!mounted) return
        setIsChecking(false)
      }
    }

    checkInit()
    return () => { mounted = false }
  }, [router])

  if (isChecking) {
    return (
      <AuthLayout>
        <div className="flex min-h-[300px] w-full max-w-[460px] items-center justify-center" role="status" aria-live="polite">
          <div className="text-sm text-muted-foreground">Loading...</div>
        </div>
      </AuthLayout>
    )
  }

  if (isInitialized) return null

  return (
    <AuthLayout>
      <MasterPasswordForm onSuccess={() => router.replace('/')} />
    </AuthLayout>
  )
}
