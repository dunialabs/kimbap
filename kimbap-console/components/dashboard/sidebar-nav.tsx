'use client'

import {
  LayoutDashboard,
  Shield,
  UserCheck,
  TrendingUp,
  Activity,
  ChevronDown,
  ChevronRight,
  Download,
  LinkIcon,
  BookOpen
} from 'lucide-react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useState, useEffect } from 'react'

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger
} from '@/components/ui/collapsible'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

export const navItems = [
  { href: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
  { href: '/dashboard/policies', icon: Shield, label: 'Policies' },
  { href: '/dashboard/approvals', icon: UserCheck, label: 'Approvals' },
  {
    href: '/dashboard/usage',
    icon: TrendingUp,
    label: 'Usage',
    subItems: [
      { href: '/dashboard/usage', label: 'Overview' },
      { href: '/dashboard/usage/tool-usage', label: 'Tool usage' },
      { href: '/dashboard/usage/token-usage', label: 'Access token usage' },
    ]
  },
  { href: '/dashboard/logs', icon: Activity, label: 'Logs & Monitoring' }
]

interface SidebarNavProps {
  onNavigate?: () => void
}

export function SidebarNav({ onNavigate }: SidebarNavProps) {
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
    <nav className="grid items-start px-2 text-sm font-medium lg:px-4">
      {navItems.map((item) => (
        <div key={item.label}>
          {item.subItems ? (
            <Collapsible
              open={expandedItems.includes(item.href)}
              onOpenChange={() => toggleExpanded(item.href)}
            >
              <CollapsibleTrigger asChild>
                <button
                  type="button"
                  className={cn(
                    'flex items-center gap-3 rounded-lg px-3 py-2 text-foreground transition-all w-full hover:bg-slate-100 dark:hover:bg-slate-800',
                    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2',
                    pathname.startsWith(item.href) &&
                      'bg-slate-100 dark:bg-slate-800'
                  )}
                >
                  <item.icon className="h-4 w-4" aria-hidden="true" focusable="false" />
                  {item.label}
                  {expandedItems.includes(item.href) ? (
                    <ChevronDown className="h-4 w-4 ml-auto" aria-hidden="true" />
                  ) : (
                    <ChevronRight className="h-4 w-4 ml-auto" aria-hidden="true" />
                  )}
                </button>
              </CollapsibleTrigger>
              <CollapsibleContent className="ml-6 mt-1 space-y-1">
                {item.subItems.map((subItem) => (
                  <Link
                    key={subItem.href}
                    href={subItem.href}
                    onClick={onNavigate}
                    aria-current={pathname === subItem.href ? 'page' : undefined}
                    className={cn(
                      'flex items-center gap-3 rounded-lg px-3 py-2 text-foreground transition-all hover:bg-slate-100 dark:hover:bg-slate-800 text-sm',
                      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2',
                      pathname === subItem.href && 'bg-slate-100 dark:bg-slate-800'
                    )}
                  >
                    {subItem.label}
                  </Link>
                ))}
              </CollapsibleContent>
            </Collapsible>
          ) : (
            <Link
              href={item.href}
              onClick={onNavigate}
              aria-current={pathname === item.href ? 'page' : undefined}
              className={cn(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-foreground transition-all hover:bg-slate-100 dark:hover:bg-slate-800',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2',
                pathname === item.href && 'bg-slate-100 dark:bg-slate-800'
              )}
            >
              <item.icon className="h-4 w-4" aria-hidden="true" focusable="false" />
              {item.label}
            </Link>
          )}
        </div>
      ))}

      {/* Resources Section */}
      <div className="mt-6">
        <div className="px-3 py-2">
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider border-b border-muted pb-2 mb-3">
            Resources
          </h3>
        </div>
        <a
          href="https://docs.kimbap.sh"
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-3 rounded-lg px-3 py-2 text-muted-foreground transition-all hover:bg-slate-100 dark:hover:bg-slate-800 mb-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
        >
          <BookOpen className="h-4 w-4" aria-hidden="true" focusable="false" />
          <span className="font-medium">Documentation</span>
          <LinkIcon className="h-3 w-3 ml-auto opacity-60" aria-hidden="true" focusable="false" />
        </a>

        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <a
                href="https://www.kimbap.sh/quick-start/#install-desk"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-3 rounded-lg px-3 py-2 text-muted-foreground transition-all hover:bg-slate-100 dark:hover:bg-slate-800 mb-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
              >
                <Download className="h-4 w-4" aria-hidden="true" focusable="false" />
                <span className="font-medium">Download Kimbap Desk</span>
                <LinkIcon className="h-3 w-3 ml-auto opacity-60" aria-hidden="true" focusable="false" />
              </a>
            </TooltipTrigger>
            <TooltipContent side="right">
              <p>Download Kimbap Desk (Opens quick start guide)</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </nav>
  )
}
