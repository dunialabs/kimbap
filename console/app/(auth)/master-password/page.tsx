"use client"

import { useRouter } from "next/navigation"
import { MasterPasswordForm } from "@/components/master-password-form"
import { AuthLayout } from "@/components/auth-layout"

export default function MasterPasswordPage() {
  const router = useRouter()

  return (
    <AuthLayout>
      <MasterPasswordForm onSuccess={() => router.replace('/')} />
    </AuthLayout>
  )
}
