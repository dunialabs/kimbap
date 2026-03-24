'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { buttonVariants } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const STORAGE_KEY = 'kimbap_getting_started_steps'

interface GettingStartedSteps {
  addTool: boolean
  createToken: boolean
}

export function GettingStartedCard() {
  const [steps, setSteps] = useState<GettingStartedSteps>({
    addTool: false,
    createToken: false
  })

  // Load from localStorage on mount
  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) {
      try {
        setSteps(JSON.parse(saved))
      } catch (e) {
        // Parse failed - use default steps
      }
    }
  }, [])

  // Save to localStorage when steps change
  const updateStep = (
    stepName: keyof GettingStartedSteps,
    completed: boolean
  ) => {
    const newSteps = { ...steps, [stepName]: completed }
    setSteps(newSteps)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(newSteps))
  }

  const totalSteps = 2
  const completed = Object.values(steps).filter(Boolean).length
  const progress = (completed / totalSteps) * 100

  // SVG circle progress calculation
  const radius = 20
  const circumference = 2 * Math.PI * radius
  const strokeDashoffset = circumference - (progress / 100) * circumference

  // Don't show the card if all steps are completed
  if (completed === totalSteps) {
    return null
  }

  return (
    <Card className="rounded-xl border border-blue-200 bg-blue-50 dark:border-blue-900 dark:bg-blue-950/30">
      <CardContent className="p-4">
        {/* Header */}
        <div className="flex items-center gap-4 mb-6">
          <div className="relative flex items-center justify-center w-12 h-12">
            <svg
              className="transform -rotate-90"
              width="48"
              height="48"
              viewBox="0 0 48 48"
              aria-hidden="true"
            >
              {/* Background circle */}
              <circle
                cx="24"
                cy="24"
                r={radius}
                fill="none"
                stroke="rgb(191, 219, 254)"
                strokeWidth="4"
                className="dark:stroke-blue-900"
              />
              {/* Progress circle */}
              <circle
                cx="24"
                cy="24"
                r={radius}
                fill="none"
                stroke="rgb(29, 78, 216)"
                strokeWidth="4"
                strokeDasharray={circumference}
                strokeDashoffset={strokeDashoffset}
                strokeLinecap="round"
                className="dark:stroke-blue-400"
              />
            </svg>
            <span className="absolute text-lg font-bold text-blue-700 dark:text-blue-400">
              {completed}/{totalSteps}
            </span>
          </div>
          <div className="flex-1">
            <h2 className="text-[24px] font-bold text-blue-700 dark:text-blue-100 mb-[4px]">
              Getting Started
            </h2>
            <p className="text-[14px] text-foreground dark:text-blue-300">
              Complete these steps to set up your server
            </p>
          </div>
        </div>

        {/* Steps */}
        <div className="space-y-3">
          {/* Step 1: Add Your First Tool */}
          {!steps.addTool && (
            <div className="flex items-center gap-4 px-[16px] py-[12px] bg-white dark:bg-slate-800 rounded-lg border border-blue-100 dark:border-blue-900 flex-wrap sm:flex-nowrap">
              <div className="flex-1">
                <h3 className="font-[700] text-[14px] text-foreground dark:text-slate-100 mb-1">
                  Add Your First Tool
                </h3>
                <p className="text-[14px] text-muted-foreground dark:text-slate-400">
                  Add MCP tools to your server
                </p>
              </div>
              <div className="flex items-center gap-2 w-full sm:w-auto justify-end">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => updateStep('addTool', true)}
                  className="w-[80px] text-slate-600 dark:text-slate-400"
                >
                  Skip
                </Button>
                <Link
                  href="/dashboard/policies"
                  onClick={() => updateStep('addTool', true)}
                  className={cn(
                    buttonVariants({ size: 'sm' }),
                    'w-[80px] bg-slate-900 hover:bg-slate-800 dark:bg-slate-100 dark:hover:bg-slate-200 text-white dark:text-slate-900'
                  )}
                >
                  Open
                </Link>
              </div>
            </div>
          )}

          {/* Step 2: Create Access Token */}
          {!steps.createToken && (
            <div className="flex items-center gap-4 px-[16px] py-[12px] bg-white dark:bg-slate-800 rounded-lg border border-blue-100 dark:border-blue-900 flex-wrap sm:flex-nowrap">
              <div className="flex-1">
                <h3 className="font-[700] text-[14px] text-foreground dark:text-slate-100 mb-1">
                  Create Access Token
                </h3>
                <p className="text-[14px] text-muted-foreground dark:text-slate-400">
                  Create tokens for apps and team members
                </p>
              </div>
              <div className="flex items-center gap-2 w-full sm:w-auto justify-end">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => updateStep('createToken', true)}
                  className="w-[80px] text-slate-600 dark:text-slate-400"
                >
                  Skip
                </Button>
                <Link
                  href="/dashboard/approvals"
                  onClick={() => updateStep('createToken', true)}
                  className={cn(
                    buttonVariants({ size: 'sm' }),
                    'w-[80px] bg-slate-900 hover:bg-slate-800 dark:bg-slate-100 dark:hover:bg-slate-200 text-white dark:text-slate-900'
                  )}
                >
                  Open
                </Link>
              </div>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
