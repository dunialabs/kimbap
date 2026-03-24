'use client'
import { Menu } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useState } from 'react'
import { Sheet, SheetContent, SheetTrigger, SheetTitle } from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { SidebarNav } from './dashboard/sidebar-nav'

const LogoutIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M9.99654 2.5H2.5V17.5H10" stroke="currentColor" strokeWidth="1.66667" strokeLinejoin="round"/>
    <path d="M13.75 13.75L17.5 10L13.75 6.25" stroke="currentColor" strokeWidth="1.66667" strokeLinejoin="round"/>
    <path d="M6.6665 9.99707H17.4998" stroke="currentColor" strokeWidth="1.66667" strokeLinejoin="round"/>
  </svg>
)

export function DashboardSidebar() {
  const router = useRouter()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  const handleLogout = () => {
    localStorage.removeItem('userid')
    localStorage.removeItem('token')
    localStorage.removeItem('auth_token')
    localStorage.removeItem('accessToken')
    localStorage.removeItem('manualAccessToken')
    localStorage.removeItem('selectedServer')
    router.push('/')
  }

  return (
    <>
      {/* Mobile header with hamburger */}
      <div className="sticky top-0 z-40 flex h-14 items-center border-b bg-background px-4 md:hidden">
        <Sheet open={mobileMenuOpen} onOpenChange={setMobileMenuOpen}>
          <SheetTrigger asChild>
            <Button variant="outline" size="icon" className="shrink-0 bg-transparent">
              <Menu className="h-5 w-5" />
              <span className="sr-only">Toggle navigation menu</span>
            </Button>
          </SheetTrigger>
          <SheetContent side="left" className="flex flex-col p-0 w-[280px]">
            <SheetTitle className="sr-only">Navigation Menu</SheetTitle>
            <div className="flex h-14 items-center border-b px-4">
              <img src="/new_logo.svg" width={237} alt="Kimbap Logo" className="block dark:hidden" />
              <img src="/darklogo.svg" width={237} alt="Kimbap Logo" className="hidden dark:block" />
            </div>
            <div className="flex-1 overflow-y-auto py-2">
              <SidebarNav onNavigate={() => setMobileMenuOpen(false)} />
            </div>
            <div className="mt-auto p-4">
              <Button
                variant="outline"
                className="w-full justify-start py-2 px-3 rounded-lg"
                onClick={handleLogout}
              >
                <LogoutIcon />
                <span className="ml-2">Logout</span>
              </Button>
            </div>
          </SheetContent>
        </Sheet>
        <div className="ml-3">
          <img src="/new_logo.svg" width={160} alt="Kimbap Logo" className="block dark:hidden" />
          <img src="/darklogo.svg" width={160} alt="Kimbap Logo" className="hidden dark:block" />
        </div>
      </div>

      {/* Desktop sidebar */}
      <div className="hidden border-r bg-background md:block fixed h-full w-[220px] lg:w-[280px]">
        <div className="flex h-full max-h-screen flex-col gap-2">
          <div className="flex h-14 items-center border-b px-4 lg:h-[60px] lg:px-6">
            <div className="flex items-center gap-2 font-semibold flex-1">
              <img src="/new_logo.svg" width={237} alt="Kimbap Logo" className="block dark:hidden" />
              <img src="/darklogo.svg" width={237} alt="Kimbap Logo" className="hidden dark:block" />
            </div>
          </div>
          <div className="flex-1">
            <SidebarNav />
          </div>
          <div className="mt-auto p-4">
            <Button
              variant="outline"
              className="w-full justify-start py-2 px-3 rounded-lg"
              onClick={handleLogout}
            >
              <LogoutIcon />
              <span className="ml-2">Logout</span>
            </Button>
          </div>
        </div>
      </div>
    </>
  )
}
