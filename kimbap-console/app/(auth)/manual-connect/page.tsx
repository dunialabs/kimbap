"use client"

import { useRouter } from "next/navigation"
import { ManualConnectForm } from "@/components/manual-connect-form"
import { AuthLayout } from "@/components/auth-layout"

export default function ManualConnectPage() {
  const router = useRouter()

  return (
    <AuthLayout>
      <ManualConnectForm
        onSuccess={() => router.push('/')}
        onBack={() => router.back()}
      />
    </AuthLayout>
  )
}
