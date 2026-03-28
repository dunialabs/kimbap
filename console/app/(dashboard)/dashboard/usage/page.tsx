"use client"

import { AlertTriangle, Loader2, RefreshCw } from "lucide-react"
import Link from "next/link"
import { useState, useEffect, useCallback, useRef, Suspense } from "react"
import { usePathname, useRouter, useSearchParams } from "next/navigation"

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator
} from "@/components/ui/breadcrumb"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { api } from "@/lib/api-client"

interface OverviewSummary {
  totalRequests24h: number
  requestsChangePercent: number
  activeTokens: number
  tokensUsedLastHour: number
  toolsInUse: number
  mostActiveToolName: string
  avgResponseTime: number
  responseTimeChange: number
}

interface TopTool {
  toolName: string
  toolType: string
  requestCount: number
  percentage: number
  color: string
}

interface ActiveToken {
  tokenName: string
  tokenMask: string
  requestCount: number
  isCurrentlyActive: boolean
  lastUsedMinutesAgo: number
}

interface RecentActivity {
  eventType: string
  description: string
  details: string
  timestamp: number
  icon: string
  color: string
}

const activityColorClass: Record<string, string> = {
  green: 'bg-green-500',
  blue: 'bg-blue-500',
  purple: 'bg-purple-500',
  orange: 'bg-orange-500',
  yellow: 'bg-amber-500',
  red: 'bg-red-500',
}

function formatRelativeMinutes(totalMinutes: number) {
  if (totalMinutes === 0) return 'just now'
  if (totalMinutes < 60) {
    return `${totalMinutes} minute${totalMinutes === 1 ? '' : 's'} ago`
  }

  const hours = Math.floor(totalMinutes / 60)
  if (hours < 24) {
    return `${hours} hour${hours === 1 ? '' : 's'} ago`
  }

  const days = Math.floor(hours / 24)
  return `${days} day${days === 1 ? '' : 's'} ago`
}

function LoadingListPlaceholder({
  label,
  rows = 3,
}: {
  label: string
  rows?: number
}) {
  return (
    <div className="space-y-3" role="status" aria-live="polite">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        <span>{label}</span>
      </div>
      {Array.from({ length: rows }).map((_, index) => (
        <div key={index} className="rounded-lg border p-3">
          <div className="h-4 w-1/3 animate-pulse rounded bg-muted" />
          <div className="mt-3 h-3 w-2/3 animate-pulse rounded bg-muted/70" />
        </div>
      ))}
    </div>
  )
}

function UsagePageContent() {
  const searchParams = useSearchParams()
  const pathname = usePathname()
  const router = useRouter()
  const [overviewSummary, setOverviewSummary] = useState<OverviewSummary | null>(null)
  const [topTools, setTopTools] = useState<TopTool[]>([])
  const [activeTokens, setActiveTokens] = useState<ActiveToken[]>([])
  const [recentActivity, setRecentActivity] = useState<RecentActivity[]>([])
  const [topToolsError, setTopToolsError] = useState<string | null>(null)
  const [activeTokensError, setActiveTokensError] = useState<string | null>(null)
  const [recentActivityError, setRecentActivityError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [timeRange, setTimeRange] = useState(() => {
    const param = searchParams.get('timeRange')
    const num = param ? Number(param) : NaN
    return [1, 7, 30].includes(num) ? num : 1
  })
  const timeRangeLabel = timeRange === 1 ? '24 hours' : `${timeRange} days`
  const logsTimeRange = timeRange === 1 ? '24h' : `${timeRange}d`
  const hasDataRef = useRef(false)

  useEffect(() => {
    document.title = 'Usage Overview | Kimbap Console'
  }, [])

  useEffect(() => {
    if (timeRange) {
      hasDataRef.current = false
    }
  }, [timeRange])

  useEffect(() => {
    const currentParam = searchParams.get('timeRange')
    const nextParam = String(timeRange)

    if (currentParam === nextParam) {
      return
    }

    const params = new URLSearchParams(searchParams.toString())
    params.set('timeRange', nextParam)
    router.replace(`${pathname}?${params.toString()}`, { scroll: false })
  }, [pathname, router, searchParams, timeRange])

  const fetchUsageData = useCallback(async () => {
    try {
      if (!hasDataRef.current) setLoading(true)

      const [summaryResult, toolsResult, tokensResult] = await Promise.allSettled([
        api.usage.getOverviewSummary({ timeRange }),
        api.usage.getTopTools({ timeRange, limit: 4 }),
        api.usage.getActiveTokens({ timeRange, limit: 3 })
      ])

      let summaryFailure = false

      if (summaryResult.status === 'fulfilled' && summaryResult.value.data?.common?.code === 0 && summaryResult.value.data?.data) {
        setOverviewSummary(summaryResult.value.data.data)
      } else {
        setOverviewSummary(null)
        summaryFailure = true
      }

      if (toolsResult.status === 'fulfilled' && toolsResult.value.data?.common?.code === 0 && toolsResult.value.data?.data?.tools) {
        setTopTools(toolsResult.value.data.data.tools)
        setTopToolsError(null)
      } else {
        setTopTools([])
        setTopToolsError('Unable to load tool usage. Check your connection and try again.')
      }

      if (tokensResult.status === 'fulfilled' && tokensResult.value.data?.common?.code === 0 && tokensResult.value.data?.data?.tokens) {
        setActiveTokens(tokensResult.value.data.data.tokens)
        setActiveTokensError(null)
      } else {
        setActiveTokens([])
        setActiveTokensError('Unable to load active tokens. Check your connection and try again.')
      }

      try {
        const activityRes = await api.usage.getRecentActivity({ timeRange, limit: 5 })
        if (activityRes.data?.common?.code === 0 && activityRes.data?.data?.activities) {
          setRecentActivity(activityRes.data.data.activities)
          setRecentActivityError(null)
        } else {
          setRecentActivity([])
          setRecentActivityError('Unable to load recent activity. Check your connection and try again.')
        }
      } catch {
        setRecentActivity([])
        setRecentActivityError('Unable to load recent activity. Check your connection and try again.')
      }
      if (!summaryFailure) {
        setLoadError(null)
      } else {
        setLoadError('Unable to load usage data. Check your connection and try again.')
      }
      hasDataRef.current = true
    } catch (error) {
      setOverviewSummary(null)
      setTopTools([])
      setTopToolsError('Unable to load tool usage. Check your connection and try again.')
      setActiveTokens([])
      setActiveTokensError('Unable to load active tokens. Check your connection and try again.')
      setRecentActivity([])
      setRecentActivityError('Unable to load recent activity. Check your connection and try again.')
      setLoadError('Unable to load usage data. Check your connection and try again.')
    } finally {
      setLoading(false)
    }
  }, [timeRange])

  const fetchRecentActivity = useCallback(async () => {
    try {
      const activityRes = await api.usage.getRecentActivity({ timeRange, limit: 5 })
      if (activityRes.data?.common?.code === 0 && activityRes.data?.data?.activities) {
        setRecentActivity(activityRes.data.data.activities)
        setRecentActivityError(null)
      } else {
        setRecentActivity([])
        setRecentActivityError('Unable to load recent activity. Check your connection and try again.')
      }
    } catch {
      setRecentActivity([])
      setRecentActivityError('Unable to load recent activity. Check your connection and try again.')
    }
  }, [timeRange])

  const handleRefresh = async () => {
    setRefreshing(true)
    await fetchUsageData()
    setRefreshing(false)
  }

  useEffect(() => {
    fetchUsageData()
  }, [fetchUsageData])

  useEffect(() => {
    const summaryInterval = setInterval(() => {
      fetchUsageData()
    }, 30000)
    const activityInterval = setInterval(() => {
      fetchRecentActivity()
    }, 15000)

    return () => {
      clearInterval(summaryInterval)
      clearInterval(activityInterval)
    }
  }, [fetchRecentActivity, fetchUsageData])


  return (
    <div className="space-y-4">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link href="/dashboard">Dashboard</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>Usage Overview</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <div className="space-y-0">
        <h1 className="text-[30px] font-bold">Usage Overview</h1>
        <p className="text-base text-muted-foreground">Check request volume, token activity, and recent changes.</p>
      </div>
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
        <Select value={String(timeRange)} onValueChange={(value) => setTimeRange(Number(value))}>
          <SelectTrigger className="w-full sm:w-[180px]" aria-label="Time range">
            <SelectValue placeholder="Time range" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="1">Last 24 hours</SelectItem>
            <SelectItem value="7">Last 7 days</SelectItem>
            <SelectItem value="30">Last 30 days</SelectItem>
          </SelectContent>
        </Select>
        <Button className="w-full sm:w-auto" variant="outline" onClick={handleRefresh} disabled={loading || refreshing}>
          <RefreshCw className={`mr-2 h-4 w-4 ${loading || refreshing ? 'animate-spin' : ''}`} />
          {refreshing ? 'Refreshing...' : loading ? 'Loading overview...' : 'Refresh'}
        </Button>
      </div>
      {!loading && loadError ? (
        <div role="alert" className="flex flex-col items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300 sm:flex-row sm:items-center">
          <AlertTriangle className="h-4 w-4" aria-hidden="true" />
          <span>{loadError}</span>
          <Button variant="outline" size="sm" className="w-full sm:ml-auto sm:w-auto" onClick={handleRefresh}>
            Retry
          </Button>
        </div>
      ) : null}


      {/* API Usage Summary */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
        <Card className="h-full">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total Requests</CardTitle>
            <CardDescription className="text-xs">Last {timeRangeLabel}</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-1 justify-center">
            <div className={loading || overviewSummary?.totalRequests24h == null ? (loadError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
              {loading
                ? '—'
                : overviewSummary?.totalRequests24h == null
                ? (loadError ? 'Unavailable' : '—')
                : overviewSummary.totalRequests24h.toLocaleString()}
            </div>
            <p className="text-xs text-muted-foreground">
              {!loading && overviewSummary && overviewSummary.requestsChangePercent !== undefined ? (
                <>
                  <span className={overviewSummary.requestsChangePercent >= 0 ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"}>
                    {overviewSummary.requestsChangePercent >= 0 ? '+' : ''}{overviewSummary.requestsChangePercent.toFixed(1)}%
                  </span> from previous period
                </>
              ) : null}
            </p>
          </CardContent>
        </Card>

        <Card className="h-full">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Active Tokens</CardTitle>
            <CardDescription className="text-xs">Last {timeRangeLabel}</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-1 justify-center">
            <div className={loading || overviewSummary?.activeTokens == null ? (loadError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
              {loading
                ? '—'
                : overviewSummary?.activeTokens == null
                ? (loadError ? 'Unavailable' : '—')
                : overviewSummary.activeTokens}
            </div>
            <p className="text-xs text-muted-foreground">
              {!loading && overviewSummary && overviewSummary.tokensUsedLastHour != null && `${overviewSummary.tokensUsedLastHour.toLocaleString()} tokens used in last hour`}
            </p>
          </CardContent>
        </Card>

        <Card className="h-full">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Tools in Use</CardTitle>
            <CardDescription className="text-xs">Last {timeRangeLabel}</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-1 justify-center">
            <div className={loading || overviewSummary?.toolsInUse == null ? (loadError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
              {loading
                ? '—'
                : overviewSummary?.toolsInUse == null
                ? (loadError ? 'Unavailable' : '—')
                    : overviewSummary.toolsInUse.toLocaleString()}
            </div>
            <p className="text-xs text-muted-foreground">
              {loading
                ? ''
                : overviewSummary?.mostActiveToolName
                ? `Most active: ${overviewSummary.mostActiveToolName}`
                : (loadError ? 'Unavailable' : '—')}
            </p>
          </CardContent>
        </Card>

        <Card className="h-full">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Average Response Time</CardTitle>
            <CardDescription className="text-xs">Last {timeRangeLabel}</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-1 justify-center">
            <div className={loading || overviewSummary?.avgResponseTime == null ? (loadError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
              {loading
                ? '—'
                : overviewSummary?.avgResponseTime == null
                ? (loadError ? 'Unavailable' : '—')
                : `${Math.round(overviewSummary.avgResponseTime)}ms`}
            </div>
            <p className="text-xs text-muted-foreground">
              {!loading && overviewSummary && overviewSummary.responseTimeChange !== undefined ? (
                (() => {
                  const roundedChange = Math.round(overviewSummary.responseTimeChange)
                  return (
                    <>
                      <span className={roundedChange <= 0 ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"}>
                        {roundedChange <= 0 ? '' : '+'}{roundedChange}ms
                      </span> from previous period
                    </>
                  )
                })()
              ) : null}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Top Tools by Usage */}
      <Card>
        <CardHeader>
            <CardTitle><Link href={`/dashboard/usage/tool-usage?timeRange=${timeRange}`} className="rounded-sm hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">Top Tools</Link></CardTitle>
            <CardDescription>Most-used tools in the last {timeRangeLabel}</CardDescription>
        </CardHeader>
        <CardContent>
          {loading || !topTools || topTools.length === 0 ? (
            <div className="flex items-center justify-center py-8">
              {loading ? (
                <LoadingListPlaceholder label="Loading top tool activity..." />
              ) : topToolsError ? (
                <div className="text-center">
                  <p className="text-sm text-red-600 dark:text-red-400">{topToolsError}</p>
                  <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No tool requests in the selected range.</p>
              )}
            </div>
          ) : (
            <div className="space-y-4">
              {topTools.map((tool) => (
                <Link key={tool.toolName} href={`/dashboard/usage/tool-usage?timeRange=${timeRange}`} className="flex flex-col gap-2 rounded-md p-1 -m-1 hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 sm:flex-row sm:items-center sm:justify-between">
                  <div className="flex min-w-0 items-center gap-3">
                    <div className={`w-2 h-2 rounded-full`} style={{ backgroundColor: tool.color }}></div>
                    <span className="truncate font-medium">{tool.toolName}</span>
                  </div>
                  <div className="w-full text-left sm:w-auto sm:text-right">
                    <div className="font-medium">{tool.requestCount.toLocaleString()} requests</div>
                    <div className="text-sm text-muted-foreground">{tool.percentage.toFixed(1)}% of total</div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Token Activity & Recent Usage */}
      <div className="grid lg:grid-cols-2 gap-3">
        <Card>
          <CardHeader>
            <CardTitle><Link href={`/dashboard/usage/token-usage?timeRange=${timeRange}`} className="rounded-sm hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">Active Tokens</Link></CardTitle>
            <CardDescription>Most active tokens in the last {timeRangeLabel}</CardDescription>
          </CardHeader>
          <CardContent>
            {loading || !activeTokens || activeTokens.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                {loading ? (
                  <LoadingListPlaceholder label="Loading active token activity..." />
                ) : activeTokensError ? (
                  <div className="text-center">
                    <p className="text-sm text-red-600 dark:text-red-400">{activeTokensError}</p>
                    <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No active tokens in the selected range.</p>
                )}
              </div>
            ) : (
              <div className="space-y-3">
                {activeTokens.map((token) => (
                  <Link key={token.tokenMask} href={`/dashboard/usage/token-usage?timeRange=${timeRange}`} className="flex flex-col items-start gap-3 rounded-lg border p-3 hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 sm:flex-row sm:items-center sm:justify-between">
                    <div className="min-w-0">
                      <div className="font-medium">{token.tokenName}</div>
                      <div className="mt-1 font-mono text-xs text-muted-foreground">{token.tokenMask}</div>
                    </div>
                    <div className="w-full text-left sm:w-auto sm:text-right">
                      <div className="font-medium">{token.requestCount.toLocaleString()} requests</div>
                      <div className={`text-sm ${token.isCurrentlyActive ? 'text-green-600 dark:text-green-400' : 'text-muted-foreground'}`}>
                        {token.isCurrentlyActive ? 'Used' : 'Last used'} {formatRelativeMinutes(token.lastUsedMinutesAgo)}
                      </div>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle><Link href={`/dashboard/logs?timeRange=${logsTimeRange}`} className="rounded-sm hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">Recent Activity</Link></CardTitle>
            <CardDescription>Recent events in the last {timeRangeLabel}</CardDescription>
          </CardHeader>
          <CardContent>
            {loading || !recentActivity || recentActivity.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                {loading ? (
                  <LoadingListPlaceholder label="Loading recent usage events..." />
                ) : recentActivityError ? (
                  <div className="text-center">
                    <p className="text-sm text-red-600 dark:text-red-400">{recentActivityError}</p>
                    <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No recent activity in the selected range.</p>
                )}
              </div>
            ) : (
              <div className="space-y-3">
                {recentActivityError ? (
                  <p className="text-xs text-amber-600 dark:text-amber-400">{recentActivityError}</p>
                ) : null}
                {recentActivity.map((activity) => (
                  <Link key={`${activity.timestamp}-${activity.description}`} href={`/dashboard/logs?timeRange=${logsTimeRange}`} className="flex items-start gap-3 rounded-md p-1 -m-1 hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">
                    <div className={`w-2 h-2 rounded-full mt-2 ${activityColorClass[activity.color] || 'bg-muted-foreground'}`}></div>
                    <div className="flex-1">
                      <div className="text-sm font-medium">{activity.description}</div>
                      <div className="text-xs text-muted-foreground">{activity.details}</div>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export default function UsagePage() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center py-10" role="status" aria-live="polite">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            <span>Loading usage overview dashboard...</span>
          </div>
        </div>
      }
    >
      <UsagePageContent />
    </Suspense>
  )
}
