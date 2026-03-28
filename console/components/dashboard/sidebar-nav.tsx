'use client'

import {
  LayoutDashboard,
  Shield,
  UserCheck,
  TrendingUp,
  Activity,
  ChevronDown,
  ChevronRight,
  LinkIcon,
  BookOpen
} from 'lucide-react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useState, useEffect } from 'react'

import {
  Collapsible,
  CollapsibleContent
} from '@/components/ui/collapsible'
import { cn } from '@/lib/utils'

export const navItems = [
  { href: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
  { href: '/dashboard/approvals', icon: UserCheck, label: 'Approvals' },
  { href: '/dashboard/logs', icon: Activity, label: 'Logs & Monitoring' },
  {
    href: '/dashboard/usage',
    icon: TrendingUp,
    label: 'Usage',
    subItems: [
      { href: '/dashboard/usage', label: 'Overview' },
      { href: '/dashboard/usage/tool-usage', label: 'Tool Usage' },
      { href: '/dashboard/usage/token-usage', label: 'Access Token Usage' },
    ]
  },
  { href: '/dashboard/policies', icon: Shield, label: 'Policies' }
]

interface SidebarNavProps {
  onNavigate?: () => void
  pendingApprovalCount?: number
}

export function SidebarNav({ onNavigate, pendingApprovalCount = 0 }: SidebarNavProps) {
  const pathname = usePathname()
  const [expandedItems, setExpandedItems] = useState<string[]>([])

  useEffect(() => {
    if (pathname.startsWith('/dashboard/usage')) {
      setExpandedItems(['/dashboard/usage'])
    } else {
      setExpandedItems([])
    }
  }, [pathname])

  const toggleExpanded = (href: string) => {
    setExpandedItems((prev) =>
      prev.includes(href)
        ? prev.filter((item) => item !== href)
        : [...prev, href]
    )
  }

  return (
    <nav className="px-2 text-sm font-medium lg:px-4" aria-label="Dashboard">
      <ul className="grid items-start gap-1">
        {navItems.map((item) => {
          const isExpanded = expandedItems.includes(item.href)
          const isSectionActive = pathname.startsWith(item.href)
          const isCurrentPage = pathname === item.href
          const subnavId = `${item.label.toLowerCase().replace(/[^a-z0-9]+/g, '-')}-subnav`

          return (
            <li key={item.label}>
              {item.subItems ? (
                <Collapsible open={isExpanded}>
                  <div className="flex items-center gap-1">
                    <Link
                      href={item.href}
                      onClick={onNavigate}
                      aria-current={isCurrentPage ? 'page' : undefined}
                      className={cn(
                        'flex min-h-11 min-w-0 flex-1 items-center gap-3 rounded-lg px-3 py-2 text-foreground transition-colors duration-200',
                        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2',
                        isSectionActive
                          ? 'bg-accent text-accent-foreground shadow-sm'
                          : 'text-muted-foreground hover:bg-accent/70 hover:text-accent-foreground'
                      )}
                    >
                      <item.icon className="h-4 w-4" aria-hidden="true" focusable="false" />
                      {item.label}
                    </Link>
                    <button
                      type="button"
                      onClick={() => toggleExpanded(item.href)}
                      aria-expanded={isExpanded}
                      aria-controls={subnavId}
                      aria-label={`${isExpanded ? 'Collapse' : 'Expand'} ${item.label} section`}
                      className={cn(
                        'h-11 w-11 rounded-lg transition-colors duration-200',
                        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2',
                        isSectionActive
                          ? 'bg-accent text-accent-foreground shadow-sm'
                          : 'text-muted-foreground hover:bg-accent/70 hover:text-accent-foreground'
                      )}
                    >
                      {isExpanded ? (
                        <ChevronDown className="h-4 w-4" aria-hidden="true" focusable="false" />
                      ) : (
                        <ChevronRight className="h-4 w-4" aria-hidden="true" focusable="false" />
                      )}
                    </button>
                  </div>
                  <CollapsibleContent id={subnavId}>
                    <ul className="ml-6 mt-1 space-y-1">
                      {item.subItems.map((subItem) => (
                        <li key={subItem.href}>
                          <Link
                            href={subItem.href}
                            onClick={onNavigate}
                            aria-current={pathname === subItem.href ? 'page' : undefined}
                            className={cn(
                              'flex min-h-11 items-center gap-3 rounded-lg px-3 py-2 text-sm text-foreground transition-colors duration-200',
                              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2',
                              pathname === subItem.href
                                ? 'bg-accent text-accent-foreground shadow-sm'
                                : 'text-muted-foreground hover:bg-accent/70 hover:text-accent-foreground'
                            )}
                          >
                            {subItem.label}
                          </Link>
                        </li>
                      ))}
                    </ul>
                  </CollapsibleContent>
                </Collapsible>
              ) : (
                <Link
                  href={item.href}
                  onClick={onNavigate}
                  aria-current={pathname === item.href ? 'page' : undefined}
                  aria-label={item.label === 'Approvals' && pendingApprovalCount > 0 ? `Approvals, ${pendingApprovalCount} pending` : undefined}
                  className={cn(
                    'flex min-h-11 items-center gap-3 rounded-lg px-3 py-2 text-foreground transition-colors duration-200',
                    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2',
                    pathname === item.href
                      ? 'bg-accent text-accent-foreground shadow-sm'
                      : 'text-muted-foreground hover:bg-accent/70 hover:text-accent-foreground'
                  )}
                >
                  <item.icon className="h-4 w-4" aria-hidden="true" focusable="false" />
                  {item.label}
                  {item.label === 'Approvals' && pendingApprovalCount > 0 && (
                    <span className="ml-auto inline-flex h-5 min-w-[20px] items-center justify-center rounded-full bg-amber-100 px-1.5 text-xs font-medium text-amber-800 dark:bg-amber-900 dark:text-amber-300">
                      {pendingApprovalCount > 99 ? '99+' : pendingApprovalCount}
                    </span>
                  )}
                </Link>
              )}
            </li>
          )
        })}
      </ul>

      <section className="mt-6" aria-labelledby="sidebar-resources-heading">
        <div className="px-3 py-2">
          <h3 id="sidebar-resources-heading" className="mb-3 border-b border-muted pb-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Resources
          </h3>
        </div>
        <ul>
          <li>
            <a
              href="https://docs.kimbap.sh"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="Documentation (opens in new tab)"
              className="mb-1 flex min-h-11 items-center gap-3 rounded-lg px-3 py-2 text-muted-foreground transition-colors duration-200 hover:bg-accent/70 hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
            >
              <BookOpen className="h-4 w-4" aria-hidden="true" focusable="false" />
              <span className="font-medium">Documentation</span>
              <LinkIcon className="ml-auto h-3 w-3 opacity-60" aria-hidden="true" focusable="false" />
            </a>
          </li>
        </ul>
      </section>
    </nav>
  )
}
