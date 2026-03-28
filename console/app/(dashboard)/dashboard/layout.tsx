import type React from 'react'

import { DashboardSidebar } from '@/components/dashboard-sidebar-new'
import { GlobalFooter } from '@/components/global-footer'

export default function DashboardLayout({
  children
}: {
  children: React.ReactNode
}) {
  return (
    <div className="min-h-screen w-full">
      <a
        href="#dashboard-main-content"
        className="sr-only focus-visible:not-sr-only focus-visible:fixed focus-visible:left-4 focus-visible:top-4 focus-visible:z-50 focus-visible:rounded focus-visible:border focus-visible:bg-background focus-visible:px-3 focus-visible:py-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
      >
        Skip to main content
      </a>
      <DashboardSidebar />
      <div className="md:pl-[220px] lg:pl-[280px]">
        <main id="dashboard-main-content" tabIndex={-1} className="flex min-h-screen flex-1 flex-col bg-muted/40">
          <div className="mx-auto w-full px-4 py-4 sm:px-6 sm:py-6 lg:px-8 max-w-[1080px] flex-1">
            {children}
          </div>
          <GlobalFooter />
        </main>
      </div>
    </div>
  )
}
