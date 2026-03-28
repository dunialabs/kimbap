'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Card, CardContent } from '@/components/ui/card'
import { Button, buttonVariants } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const STORAGE_KEY = 'kimbap_getting_started_dismissed'

export function GettingStartedCard() {
  const [dismissed, setDismissed] = useState(false)
  const [isReady, setIsReady] = useState(false)

  useEffect(() => {
      try {
        setDismissed(localStorage.getItem(STORAGE_KEY) === 'true')
      } catch {
        setDismissed(false)
      }
      setIsReady(true)
    }, [])

  const handleDismiss = () => {
      setDismissed(true)
      try {
        localStorage.setItem(STORAGE_KEY, 'true')
      } catch {
        return
      }
    }

  if (!isReady || dismissed) {
    return null
  }

  return (
    <Card className="rounded-xl border border-blue-200/70 bg-blue-50/80 dark:border-blue-900/70 dark:bg-blue-950/20">
      <CardContent className="p-4">
        <div className="mb-3 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex-1">
            <h2 className="mb-1 text-lg font-semibold text-foreground sm:text-xl">
              Operator shortcuts
            </h2>
            <p className="text-sm leading-6 text-muted-foreground">
              Open these common operator views to get oriented quickly.
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleDismiss}
            className="shrink-0"
          >
            Dismiss
          </Button>
        </div>

        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-4 rounded-lg border border-blue-200/70 bg-background/90 p-4 dark:border-blue-900/70 dark:bg-background/60 sm:flex-nowrap">
            <div className="flex-1">
              <h3 className="mb-1 text-sm font-semibold text-foreground">
                Set Up Access Policies
              </h3>
              <p className="text-sm leading-6 text-muted-foreground">
                Define which tool calls are allowed, need approval, or are blocked.
              </p>
            </div>
            <div className="w-full sm:w-auto flex justify-end">
              <Link
                href="/dashboard/policies"
                className={cn(
                  buttonVariants({ size: 'sm' }),
                  'w-[140px] justify-center'
                )}
              >
                Open Policies
              </Link>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-4 rounded-lg border border-blue-200/70 bg-background/90 p-4 dark:border-blue-900/70 dark:bg-background/60 sm:flex-nowrap">
            <div className="flex-1">
              <h3 className="mb-1 text-sm font-semibold text-foreground">
                Review Pending Approvals
              </h3>
              <p className="text-sm leading-6 text-muted-foreground">
                Check whether any requests are waiting on an operator decision.
              </p>
            </div>
            <div className="w-full sm:w-auto flex justify-end">
              <Link
                href="/dashboard/approvals"
                className={cn(
                  buttonVariants({ size: 'sm' }),
                  'w-[140px] justify-center'
                )}
              >
                Open Approvals
              </Link>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-4 rounded-lg border border-blue-200/70 bg-background/90 p-4 dark:border-blue-900/70 dark:bg-background/60 sm:flex-nowrap">
            <div className="flex-1">
              <h3 className="mb-1 text-sm font-semibold text-foreground">
                Check Recent Logs
              </h3>
              <p className="text-sm leading-6 text-muted-foreground">
                Open live request and error logs to investigate issues quickly.
              </p>
            </div>
            <div className="w-full sm:w-auto flex justify-end">
              <Link
                href="/dashboard/logs"
                className={cn(
                  buttonVariants({ size: 'sm' }),
                  'w-[140px] justify-center'
                )}
              >
                Open Logs
              </Link>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
