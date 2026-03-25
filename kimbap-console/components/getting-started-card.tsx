'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { buttonVariants } from '@/components/ui/button'
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
    <Card className="rounded-xl border border-blue-200 bg-blue-50 dark:border-blue-900 dark:bg-blue-950/30">
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-4 mb-4">
          <div className="flex-1">
            <h2 className="text-[24px] font-bold text-blue-700 dark:text-blue-100 mb-[4px]">
              Start here
            </h2>
            <p className="text-[14px] text-foreground dark:text-blue-300">
              Open these common operator views to get oriented quickly.
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleDismiss}
            className="shrink-0 text-slate-600 dark:text-slate-300"
          >
            Dismiss
          </Button>
        </div>

        <div className="space-y-3">
          <div className="flex items-center gap-4 px-[16px] py-[12px] bg-white dark:bg-slate-800 rounded-lg border border-blue-100 dark:border-blue-900 flex-wrap sm:flex-nowrap">
            <div className="flex-1">
              <h3 className="font-[700] text-[14px] text-foreground dark:text-slate-100 mb-1">
                Review Pending Approvals
              </h3>
              <p className="text-[14px] text-muted-foreground dark:text-slate-400">
                Check whether any requests are waiting on an operator decision.
              </p>
            </div>
            <div className="w-full sm:w-auto flex justify-end">
              <Link
                href="/dashboard/approvals"
                className={cn(
                  buttonVariants({ size: 'sm' }),
                  'w-[140px] bg-slate-900 hover:bg-slate-800 dark:bg-slate-100 dark:hover:bg-slate-200 text-white dark:text-slate-900'
                )}
              >
                Open approvals
              </Link>
            </div>
          </div>

          <div className="flex items-center gap-4 px-[16px] py-[12px] bg-white dark:bg-slate-800 rounded-lg border border-blue-100 dark:border-blue-900 flex-wrap sm:flex-nowrap">
            <div className="flex-1">
              <h3 className="font-[700] text-[14px] text-foreground dark:text-slate-100 mb-1">
                Check Usage Trends
              </h3>
              <p className="text-[14px] text-muted-foreground dark:text-slate-400">
                See request volume and activity changes over the last 30 days.
              </p>
            </div>
            <div className="w-full sm:w-auto flex justify-end">
              <Link
                href="/dashboard/usage"
                className={cn(
                  buttonVariants({ size: 'sm' }),
                  'w-[140px] bg-slate-900 hover:bg-slate-800 dark:bg-slate-100 dark:hover:bg-slate-200 text-white dark:text-slate-900'
                )}
              >
                Open usage
              </Link>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
