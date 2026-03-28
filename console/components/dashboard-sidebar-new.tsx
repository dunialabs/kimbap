'use client'
import Image from 'next/image'
import { Menu, LogOut } from 'lucide-react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { clearAuthState } from '@/lib/api-client'
import { useState, useEffect, useRef, useCallback } from 'react'
import { Sheet, SheetContent, SheetDescription, SheetTrigger, SheetTitle } from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { SidebarNav } from './dashboard/sidebar-nav'
import { api } from '@/lib/api-client'
import { useUserRole } from '@/hooks/use-user-role'


export function DashboardSidebar() {
  const router = useRouter()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const [pendingApprovalCount, setPendingApprovalCount] = useState(0)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const { userRole } = useUserRole()

  const fetchPendingCount = useCallback(async () => {
    try {
      const res = await api.approvals.countPending()
      const data = res.data?.data || res.data
      setPendingApprovalCount(data?.count || 0)
    } catch {
      // Silently fail — sidebar should never show errors
    }
  }, [])

  useEffect(() => {
    fetchPendingCount()
    timerRef.current = setInterval(fetchPendingCount, 30_000)
    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
    }
  }, [fetchPendingCount])

  const handleSignOut = () => {
    clearAuthState()
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
          <SheetContent side="left" className="flex w-[280px] flex-col p-0">
            <SheetTitle className="sr-only">Navigation Menu</SheetTitle>
            <SheetDescription className="sr-only">Browse dashboard sections, documentation links, and account actions.</SheetDescription>
            <div className="flex h-14 items-center border-b px-4">
              <Link
                href="/dashboard"
                className="rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
                onClick={() => setMobileMenuOpen(false)}
                aria-label="Go to dashboard"
              >
                <Image src="/new_logo.svg" width={237} height={34} alt="Kimbap Logo" className="block h-auto max-w-full dark:hidden" priority />
                <Image src="/darklogo.svg" width={237} height={34} alt="Kimbap Logo" className="hidden h-auto max-w-full dark:block" priority />
              </Link>
            </div>
            <div className="flex-1 overflow-y-auto py-2">
              <SidebarNav onNavigate={() => setMobileMenuOpen(false)} pendingApprovalCount={pendingApprovalCount} />
            </div>
            <div className="mt-auto p-4 space-y-2">
              {userRole && (
                <p className="text-xs text-muted-foreground px-1">Role: {userRole}</p>
              )}
              <Button
                variant="outline"
                className="w-full justify-start py-2 px-3 rounded-lg"
                onClick={handleSignOut}
              >
                <LogOut className="h-5 w-5" aria-hidden="true" />
                <span className="ml-2">Sign out</span>
              </Button>
            </div>
          </SheetContent>
        </Sheet>
        <Link
          href="/dashboard"
          className="ml-3 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          aria-label="Go to dashboard"
        >
          <Image src="/new_logo.svg" width={160} height={23} alt="Kimbap Logo" className="block h-auto max-w-full dark:hidden" priority />
          <Image src="/darklogo.svg" width={160} height={23} alt="Kimbap Logo" className="hidden h-auto max-w-full dark:block" priority />
        </Link>
      </div>

      {/* Desktop sidebar */}
      <div className="fixed inset-y-0 hidden w-[220px] border-r bg-background md:block lg:w-[280px]">
        <div className="flex h-full flex-col gap-2 overflow-y-auto">
          <div className="flex h-14 items-center border-b px-4 lg:h-[60px] lg:px-6">
            <Link
              href="/dashboard"
              className="flex items-center gap-2 font-semibold flex-1 min-w-0 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
              aria-label="Go to dashboard"
            >
              <Image src="/new_logo.svg" width={237} height={34} alt="Kimbap Logo" className="block dark:hidden max-w-full h-auto" priority />
              <Image src="/darklogo.svg" width={237} height={34} alt="Kimbap Logo" className="hidden dark:block max-w-full h-auto" priority />
            </Link>
          </div>
          <div className="flex-1">
            <SidebarNav pendingApprovalCount={pendingApprovalCount} />
          </div>
          <div className="mt-auto p-4 space-y-2">
            {userRole && (
              <p className="text-xs text-muted-foreground px-1">Role: {userRole}</p>
            )}
            <Button
              variant="outline"
              className="w-full justify-start py-2 px-3 rounded-lg"
              onClick={handleSignOut}
            >
              <LogOut className="h-5 w-5" aria-hidden="true" />
              <span className="ml-2">Sign out</span>
            </Button>
          </div>
        </div>
      </div>
    </>
  )
}
