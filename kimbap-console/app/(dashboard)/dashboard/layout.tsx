import type React from 'react'

import { DashboardSidebar } from '@/components/dashboard-sidebar-new'
import { MasterPasswordProvider } from '@/contexts/master-password-context'
import { GlobalFooter } from '@/components/global-footer'

export default function DashboardLayout({
  children
}: {
  children: React.ReactNode
}) {
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
