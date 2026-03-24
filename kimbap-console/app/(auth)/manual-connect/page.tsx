'use client'

import Link from 'next/link'

import { AuthLayout } from '@/components/auth-layout'
import { Button } from '@/components/ui/button'

export default function ManualConnectPage() {
  return (
    <AuthLayout>
      <div className="space-y-4 text-center">
        <h1 className="text-2xl font-semibold">Manual connection is no longer available</h1>
        <p className="text-sm text-muted-foreground">Use the standard sign-in flow to access the operations console.</p>
        <Link href="/">
          <Button>Back to sign in</Button>
        </Link>
      </div>
    </AuthLayout>
  )
}
