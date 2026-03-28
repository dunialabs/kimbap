/*
 * @Author: xudada 1820064201@qq.com
 * @Date: 2025-10-31 12:10:28
 * @LastEditors: xudada 1820064201@qq.com
 * @LastEditTime: 2025-10-31 12:29:58
 * @FilePath: /kimbap-console/components/auth-layout.tsx
 * @Description: Authentication page layout wrapper
 */
import Image from 'next/image'
import Link from 'next/link'
import type React from 'react'

interface AuthLayoutProps {
  children: React.ReactNode
}

export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="flex min-h-full flex-1 flex-col bg-gradient-to-br from-orange-50 via-background to-amber-50/70 px-4 py-4 dark:from-background dark:via-background dark:to-background sm:px-6 sm:py-6 lg:flex-row">
      <div className="hidden lg:flex lg:w-1/2 lg:flex-col lg:justify-center lg:p-12">
        <div className="max-w-md">
          <h1 className="mb-1 text-[52px] font-bold leading-tight text-orange-600 dark:text-orange-400">
            Kimbap Console
          </h1>
          <h2 className="mb-6 text-[40px] font-bold leading-tight text-slate-900 dark:text-foreground xl:text-[52px]">
            Operator Workspace
          </h2>
          <p className="mb-3 text-[16px] leading-relaxed text-muted-foreground">
            Manage policies, approvals, logs, and usage for your Kimbap server from one place.
          </p>
          <p className="text-sm leading-6 text-muted-foreground">
            Owners usually sign in with the master password. Admins can use an access token. Member tokens are for using the server, not managing this console.
          </p>
        </div>
      </div>

      <main className="flex flex-1 rounded-xl border border-border/60 bg-card shadow-sm">
        <div className="flex w-full flex-col">
          <div className="p-4">
            <Link
              href="/"
              className="inline-block rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
              aria-label="Go to sign in"
            >
              <Image src="/new_logo.svg" alt="Kimbap Logo" width={239} height={34} className="block h-auto max-w-full dark:hidden" priority />
              <Image src="/darklogo.svg" alt="Kimbap Logo" width={239} height={34} className="hidden h-auto max-w-full dark:block" priority />
            </Link>
          </div>

          <div className="border-t border-border/60 px-4 pt-4 lg:hidden">
            <h1 className="text-2xl font-semibold tracking-tight">Kimbap Console</h1>
            <p className="mt-2 text-sm leading-6 text-muted-foreground">
              Operator workspace for policies, approvals, logs, and usage. Owners usually sign in with the master password; admins use access tokens.
            </p>
          </div>

          <div className="flex flex-1 flex-col items-center justify-center px-4 py-6 sm:px-6">
            {children}
          </div>
        </div>
      </main>
    </div>
  )
}
