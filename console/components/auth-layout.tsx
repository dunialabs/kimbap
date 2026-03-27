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
    <div className="flex min-h-screen p-[24px] bg-slate-100 dark:bg-slate-950">
      {/* Left Side - Branding */}
      <div className="hidden lg:flex lg:w-1/2  p-12 flex-col justify-center">
        <div className="max-w-md">
          <h1 className="text-[52px] font-bold text-orange-600 dark:text-orange-500 mb-[4px] leading-tight">
            Kimbap Console
          </h1>
          <h2 className="text-[52px] font-bold text-slate-900 dark:text-slate-100 mb-[24px] leading-tight">
            Operations Console
          </h2>
          <p className="text-muted-foreground text-[16px] leading-relaxed">
            Review logs, handle approvals, manage policies, and monitor usage from one place.
          </p>
        </div>
      </div>

      {/* Right Side - Content */}
      <div className="flex-1 flex  bg-white dark:bg-slate-900">
        <div className="w-full flex flex-col">
          {/* Logo */}
          <div className="p-[14px]">
            <Link
              href="/"
              className="inline-block rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
              aria-label="Go to sign in"
            >
              <Image src="/new_logo.svg" alt="Kimbap Logo" width={239} height={34} className="block dark:hidden" priority />
              <Image src="/darklogo.svg" alt="Kimbap Logo" width={239} height={34} className="hidden dark:block" priority />
            </Link>
          </div>

          {/* Content */}
          <div className="flex-1 flex flex-col justify-center items-center">
            {children}
          </div>
        </div>
      </div>
    </div>
  )
}
