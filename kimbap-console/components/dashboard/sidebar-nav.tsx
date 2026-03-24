'use client';

import {
  LayoutDashboard,
  Shield,
  UserCheck,
  Activity,
  BookOpen,
  Download,
  LinkIcon,
  Blocks,
  Plug,
  Key,
  Settings,
  ClipboardList,
  BarChart3,
} from 'lucide-react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';

// PRD Section 11: Primary navigation — operations and observability
export const primaryNavItems = [
  { href: '/dashboard', icon: LayoutDashboard, label: 'Overview' },
  { href: '/dashboard/approvals', icon: UserCheck, label: 'Approvals' },
  { href: '/dashboard/audit', icon: ClipboardList, label: 'Audit' },
  { href: '/dashboard/logs', icon: Activity, label: 'Logs' },
  { href: '/dashboard/stats', icon: BarChart3, label: 'Stats' },
  { href: '/dashboard/integrations', icon: Plug, label: 'Integrations' },
];

// PRD Section 11: Secondary navigation — configuration and management
export const secondaryNavItems = [
  { href: '/dashboard/policies', icon: Shield, label: 'Policies' },
  { href: '/dashboard/tokens', icon: Key, label: 'Tokens / Sessions' },
  { href: '/dashboard/skills', icon: Blocks, label: 'Skills / Packages' },
  { href: '/dashboard/settings', icon: Settings, label: 'Settings' },
];

export const navItems = [...primaryNavItems, ...secondaryNavItems];

const LEGACY_PATH_PREFIXES: [string, string][] = [
  ['/dashboard/usage', '/dashboard/stats'],
  ['/dashboard/connectors', '/dashboard/integrations'],
];

function resolveLegacyPath(pathname: string): string {
  for (const [legacy, canonical] of LEGACY_PATH_PREFIXES) {
    if (pathname === legacy || pathname.startsWith(legacy + '/')) {
      return canonical + pathname.slice(legacy.length);
    }
  }
  return pathname;
}

interface SidebarNavProps {
  onNavigate?: () => void;
}

function NavLink({
  item,
  pathname,
  onNavigate,
}: {
  item: (typeof primaryNavItems)[number];
  pathname: string;
  onNavigate?: () => void;
}) {
  const resolved = resolveLegacyPath(pathname);
  const isActive =
    item.href === '/dashboard'
      ? resolved === '/dashboard'
      : resolved === item.href || resolved.startsWith(item.href + '/');

  return (
    <Link
      href={item.href}
      onClick={onNavigate}
      aria-current={isActive ? 'page' : undefined}
      className={cn(
        'flex items-center gap-3 rounded-lg px-3 py-2 text-foreground transition-all hover:bg-slate-100 dark:hover:bg-slate-800',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2',
        isActive && 'bg-slate-100 dark:bg-slate-800',
      )}
    >
      <item.icon className="h-4 w-4" aria-hidden="true" focusable="false" />
      {item.label}
    </Link>
  );
}

export function SidebarNav({ onNavigate }: SidebarNavProps) {
  const pathname = usePathname();

  return (
    <nav className="grid items-start px-2 text-sm font-medium lg:px-4">
      {primaryNavItems.map((item) => (
        <div key={item.label}>
          <NavLink item={item} pathname={pathname} onNavigate={onNavigate} />
        </div>
      ))}

      <div className="mt-4">
        <div className="px-3 py-2">
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider border-b border-muted pb-2 mb-3">
            Configuration
          </h3>
        </div>
        {secondaryNavItems.map((item) => (
          <div key={item.label}>
            <NavLink item={item} pathname={pathname} onNavigate={onNavigate} />
          </div>
        ))}
      </div>

      <div className="mt-4">
        <div className="px-3 py-2">
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider border-b border-muted pb-2 mb-3">
            Resources
          </h3>
        </div>
        <a
          href="https://docs.kimbap.io"
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
                href="https://www.kimbap.io/quick-start/#install-desk"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-3 rounded-lg px-3 py-2 text-muted-foreground transition-all hover:bg-slate-100 dark:hover:bg-slate-800 mb-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
              >
                <Download className="h-4 w-4" aria-hidden="true" focusable="false" />
                <span className="font-medium">Download Kimbap Desk</span>
                <LinkIcon
                  className="h-3 w-3 ml-auto opacity-60"
                  aria-hidden="true"
                  focusable="false"
                />
              </a>
            </TooltipTrigger>
            <TooltipContent side="right">
              <p>Download Kimbap Desk (Opens quick start guide)</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </nav>
  );
}
