'use client'

import Link from 'next/link'

import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'

const shortcuts = [
  {
    href: '/dashboard/policies',
    title: 'Set up your first policy',
    description: 'Start here on a new server. Decide which tool calls are allowed, require approval, or are blocked.',
    actionLabel: 'Set first policy'
  },
  {
    href: '/dashboard/approvals',
    title: 'Check the approvals queue',
    description: 'If any calls need operator review, they will appear here for approval or rejection.',
    actionLabel: 'Open approvals'
  },
  {
    href: '/dashboard/logs',
    title: 'Watch the first requests in logs',
    description: 'Once agents begin using the server, logs show requests, errors, and outcomes in one place.',
    actionLabel: 'Open logs'
  }
]

export function GettingStartedCard() {
  return (
    <Card className="rounded-xl border border-blue-200/70 bg-blue-50/80 dark:border-blue-900/70 dark:bg-blue-950/20">
      <CardContent className="p-4">
        <div className="mb-3">
          <h2 className="mb-1 text-lg font-semibold text-foreground sm:text-xl">
            Getting started
          </h2>
          <p className="text-sm leading-6 text-muted-foreground">
            Follow these steps in order to set guardrails first, then verify approvals and logs before wider rollout.
          </p>
        </div>

        <ul className="space-y-3">
          {shortcuts.map((shortcut) => (
            <li
              key={shortcut.href}
              className="flex flex-wrap items-center gap-4 rounded-lg border border-blue-200/70 bg-background/90 p-4 dark:border-blue-900/70 dark:bg-background/60 sm:flex-nowrap"
            >
              <div className="flex-1">
                <h3 className="mb-1 text-sm font-semibold text-foreground">
                  {shortcut.title}
                </h3>
                <p className="text-sm leading-6 text-muted-foreground">
                  {shortcut.description}
                </p>
              </div>
              <div className="flex w-full justify-end sm:w-auto">
                <Link
                  href={shortcut.href}
                  className={cn(buttonVariants({ size: 'sm' }), 'min-h-11 w-[148px] justify-center px-4')}
                >
                  {shortcut.actionLabel}
                </Link>
              </div>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  )
}
