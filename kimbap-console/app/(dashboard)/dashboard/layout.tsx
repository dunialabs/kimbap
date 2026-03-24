/*
 * @Author: xudada 1820064201@qq.com
 * @Date: 2025-11-21 12:46:59
 * @LastEditors: xudada 1820064201@qq.com
 * @LastEditTime: 2025-11-24 15:29:13
 * @FilePath: /kimbap-console/app/(dashboard)/dashboard/layout.tsx
 * @Description: Dashboard layout with sidebar navigation
 */
'use client'

import type React from 'react'
import { useEffect, useState } from 'react'

import { DashboardSidebar } from '@/components/dashboard-sidebar-new'
import { MasterPasswordProvider } from '@/contexts/master-password-context'
import { GlobalFooter } from '@/components/global-footer'


export default function DashboardLayout({
  children
}: {
  children: React.ReactNode
}) {
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    // Wait for client-side hydration
    setIsLoading(false)
  }, [])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div aria-live="polite" className="flex flex-col items-center gap-2">
          <div className="w-8 h-8 border-2 border-muted-foreground/30 border-t-foreground rounded-full animate-spin" aria-hidden="true" />
          <span className="sr-only">Loading dashboard</span>
        </div>
      </div>
    )
  }

  // Member access removed - Members now have access to dashboard
  // All users (including Members) can access the dashboard

  // For admin/owner users, show full dashboard with sidebar
  return (
    <MasterPasswordProvider>
      <div className="min-h-screen w-full">
        <a
          href="#dashboard-main-content"
          className="sr-only focus:not-sr-only focus:fixed focus:top-4 focus:left-4 focus:z-50 focus:bg-background focus:border focus:px-3 focus:py-2 focus:rounded"
        >
          Skip to main content
        </a>
        <DashboardSidebar />
        <div className="md:pl-[220px] lg:pl-[280px]">
          <main id="dashboard-main-content" className="flex flex-1 flex-col bg-muted/40 min-h-screen">
            <div className="mx-auto w-full px-4 py-4 sm:px-6 sm:py-6 lg:px-8 max-w-[1080px] flex-1">
              {children}
            </div>
            <GlobalFooter />
          </main>
        </div>
      </div>
    </MasterPasswordProvider>
  )
}
