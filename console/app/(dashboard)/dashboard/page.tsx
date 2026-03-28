'use client'

import {
  Server,
  AlertTriangle
} from 'lucide-react'
import Link from 'next/link'
import { useEffect, useState, useCallback } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@/components/ui/card'
import { GettingStartedCard } from '@/components/getting-started-card'
import { cn, formatDisplayNumber, formatNullableText } from '@/lib/utils'

function getRequestErrorMessage(
  error: unknown,
  messages: { auth: string; network: string; fallback: string }
) {
  const requestError = error as {
    response?: { status?: number; data?: { common?: { message?: string } } }
    userMessage?: string
    message?: string
    code?: string
  }
  const status = requestError.response?.status
  const rawMessage =
    requestError.userMessage ||
    requestError.response?.data?.common?.message ||
    requestError.message ||
    ''

  if (status === 401 || status === 403) {
    return rawMessage || messages.auth
  }

  if (!requestError.response || requestError.code === 'ECONNABORTED') {
    return messages.network
  }

  return rawMessage || messages.fallback
}


function getRecentActivityStatusMeta(status: string): { dot: string; label: string } {
  const statusMap: Record<string, { dot: string; label: string }> = {
    success: { dot: 'bg-green-500', label: 'Success' },
    warning: { dot: 'bg-amber-500', label: 'Warning' },
    info: { dot: 'bg-blue-500', label: 'Info' },
    error: { dot: 'bg-red-500', label: 'Error' },
  }

  return statusMap[status] ?? { dot: 'bg-muted-foreground', label: status }
}

interface ServerInfo {
  proxyId: string
  proxyName: string
  proxyKey: string
  status: number
  createdAt: number
}

interface DashboardData {
  apiRequests: number | null
  activeTokens: number | null
  configuredTools: number | null
  uptime: string | null
  recentActivity: Array<{ action: string; time: string; status: string }>
}

export default function DashboardPage() {
  const [serverInfo, setServerInfo] = useState<ServerInfo | null>(null)
  const [serverFetchError, setServerFetchError] = useState<string | null>(null)
  const [isServerInfoLoading, setIsServerInfoLoading] = useState(true)

  const [dashboardData, setDashboardData] = useState<DashboardData | null>(null)
  const [dashboardLoadError, setDashboardLoadError] = useState<string | null>(null)
  const [pendingApprovalCount, setPendingApprovalCount] = useState<number | null>(null)
  const [pendingApprovalError, setPendingApprovalError] = useState<string | null>(null)
  const [isPendingApprovalLoading, setIsPendingApprovalLoading] = useState(true)


  const fetchServerInfo = useCallback(async () => {
    setIsServerInfoLoading(true)
    setServerFetchError(null)
    try {
        const { api } = await import('@/lib/api-client')

        // Try to get cached server info from localStorage first
        let selectedServer: string | null = null
        try {
          selectedServer = localStorage.getItem('selectedServer')
        } catch {
          selectedServer = null
        }
        if (selectedServer) {
          try {
            const parsedServer = JSON.parse(selectedServer)

            // Use cached server info immediately, then refresh from API
            if (parsedServer.proxyId && parsedServer.proxyName) {
              // Normalize status: convert string 'running'/'stopped' to number 1/2
              let normalizedStatus = parsedServer.status
              if (typeof normalizedStatus === 'string') {
                normalizedStatus = normalizedStatus.toLowerCase() === 'running' ? 1 : 2
              } else if (typeof normalizedStatus !== 'number') {
                normalizedStatus = 1 // Default to running if invalid
              }

              setServerInfo({
                proxyId: parsedServer.proxyId,
                proxyName: parsedServer.proxyName,
                proxyKey: parsedServer.proxyKey,
                status: normalizedStatus,
                createdAt: parsedServer.createdAt
              })
            }
          } catch {
            // Malformed localStorage — fall through to API fetch
          }
        }

        // No cache found, fetch server info using protocol 10002
        const serverInfoResponse = await api.servers.getInfo()
        if (serverInfoResponse.data?.data) {
          const data = serverInfoResponse.data.data
          setServerInfo(data)

          // Update localStorage cache with fresh server info
          if (selectedServer) {
            try {
              const parsedServer = JSON.parse(selectedServer)
              parsedServer.proxyId = data.proxyId
              parsedServer.id = data.proxyId
              parsedServer.proxyName = data.proxyName
              parsedServer.name = data.proxyName
              parsedServer.proxyKey = data.proxyKey
              parsedServer.status = data.status
              parsedServer.createdAt = data.createdAt
              localStorage.setItem(
                'selectedServer',
                JSON.stringify(parsedServer)
              )
            } catch {
              // Failed to update localStorage cache
            }
          }
        }
    } catch (error: unknown) {
      const requestError = error as { response?: { status?: number; data?: { common?: { message?: string } } }; message?: string }
      const msg = requestError.response?.data?.common?.message || requestError.message || ''
      const isNotFound = msg.toLowerCase().includes('not found') || requestError.response?.status === 404
      if (isNotFound) {
        setServerInfo(null)
        try {
          localStorage.removeItem('selectedServer')
        } catch {
          // Failed to clear stale server selection
        }
      } else {
        setServerFetchError(
          getRequestErrorMessage(error, {
            auth: 'Session expired or access revoked. Sign in again.',
            network: 'Could not reach the server. Check your connection and retry.',
            fallback: 'Could not load server details. Retry or sign in again.'
          })
        )
      }
    } finally {
      setIsServerInfoLoading(false)
    }
  }, [])

  useEffect(() => {
    void fetchServerInfo()
  }, [fetchServerInfo])

  useEffect(() => {
    document.title = 'Dashboard | Kimbap Console'
  }, [])

  const fetchPendingApprovals = useCallback(async () => {
    setIsPendingApprovalLoading(true)
    setPendingApprovalError(null)
    try {
      const { api } = await import('@/lib/api-client')
      const res = await api.approvals.countPending()
      const data = res.data?.data || res.data
      setPendingApprovalCount(data?.count || 0)
    } catch (error: unknown) {
      setPendingApprovalCount(null)
      setPendingApprovalError(
        getRequestErrorMessage(error, {
          auth: 'Session expired or access revoked. Sign in again to check approvals.',
          network: 'Could not check the approval queue. Check your connection and retry.',
          fallback: 'Could not load pending approvals. Retry.'
        })
      )
    } finally {
      setIsPendingApprovalLoading(false)
    }
  }, [])

  useEffect(() => {
    void fetchPendingApprovals()
  }, [fetchPendingApprovals])

  const fetchDashboardData = useCallback(async () => {
    if (!serverInfo?.proxyId) return

    setDashboardLoadError(null)

    try {
      const { api } = await import('@/lib/api-client')

      // Fetch dashboard overview data using protocol 10023
      const overviewResponse = await api.dashboard.overview('30d')

      if (overviewResponse.data?.data) {
        setDashboardData(overviewResponse.data.data)
        setDashboardLoadError(null)
      } else {
        setDashboardData(null)
        setDashboardLoadError('Could not load dashboard data. Retry.')
      }
    } catch (error: unknown) {
      setDashboardData(null)
      setDashboardLoadError(
        getRequestErrorMessage(error, {
          auth: 'Session expired or access revoked. Sign in again.',
          network: 'Could not load dashboard data. Check your connection and retry.',
          fallback: 'Could not load dashboard data. Retry.'
        })
      )
    }
  }, [serverInfo])

  useEffect(() => {
    void fetchDashboardData()
  }, [fetchDashboardData])

  // Use real data if available, otherwise show empty stats
  const stats = {
    apiRequests: dashboardData?.apiRequests ?? null,
    activeTokens: dashboardData?.activeTokens ?? null,
    configuredTools: dashboardData?.configuredTools ?? null,
    uptime: dashboardData?.uptime ?? null,
  }

  const recentActivity = dashboardData?.recentActivity || []
  const isDashboardLoading = !dashboardData && !dashboardLoadError
  const hasPendingApprovals = (pendingApprovalCount ?? 0) > 0
  const hasRecordedDashboardActivity =
    (stats.apiRequests ?? 0) > 0 ||
    recentActivity.length > 0 ||
    hasPendingApprovals
  const isEmptyDashboard =
    !isDashboardLoading &&
    !dashboardLoadError &&
    !pendingApprovalError &&
    (stats.configuredTools ?? 0) === 0 &&
    !hasRecordedDashboardActivity

  if (isServerInfoLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div aria-live="polite" className="text-center">
          <div className="w-8 h-8 border-2 border-muted-foreground/30 border-t-foreground rounded-full animate-spin mx-auto mb-4" aria-hidden="true" />
          <h3 className="text-lg font-semibold mb-2">Loading dashboard</h3>
          <p className="text-muted-foreground">Loading server data…</p>
        </div>
      </div>
    )
  }

  if (serverFetchError && !serverInfo) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center" role="alert">
          <Server className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-semibold mb-2">Could not load server connection</h3>
          <p className="text-muted-foreground mb-4">
            {serverFetchError}
          </p>
          <div className="flex flex-col gap-2 sm:flex-row sm:justify-center">
            <Button className="min-h-11" onClick={() => void fetchServerInfo()}>Retry</Button>
            <Link href="/" className={cn(buttonVariants({ variant: 'outline' }), 'min-h-11')}>
              Back to sign in
            </Link>
          </div>
        </div>
      </div>
    )
  }

  if (!serverInfo) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Server className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-semibold mb-2">No server connected</h3>
          <p className="text-muted-foreground mb-4">
            Sign in and connect a server to use the dashboard.
          </p>
          <Link href="/" className={cn(buttonVariants(), 'min-h-11')}>
            Back to sign in
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-[30px] font-bold tracking-tight">Dashboard</h1>
          <p className="text-sm text-muted-foreground">
            {serverInfo?.proxyName || 'Kimbap Server'}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="relative flex h-3 w-3" aria-hidden="true">
            {serverInfo?.status === 1 ? (
              <>
            <span className="absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75 motion-safe:animate-ping"></span>
            <span className="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
              </>
            ) : (
              <span className="relative inline-flex h-3 w-3 rounded-full bg-slate-400 dark:bg-slate-500"></span>
            )}
          </span>
          <Badge
            variant="secondary"
            className={
              serverInfo?.status === 1
                ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-950 dark:text-green-400 dark:border-green-900'
                : 'border-slate-200 bg-slate-50 text-slate-700 dark:border-slate-700 dark:bg-slate-900/60 dark:text-slate-200'
            }
          >
            {serverInfo?.status === 1 ? 'Running' : 'Stopped'}
          </Badge>
          <span className="sr-only">Server status: {serverInfo?.status === 1 ? 'Running' : 'Stopped'}</span>
        </div>
      </div>
      <div className="space-y-6">
        {serverFetchError && serverInfo && (
          <div role="alert" className="flex flex-col items-start gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-950/20 dark:text-amber-300 sm:flex-row sm:items-center">
            <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span>Server details could not be refreshed. The information shown may be out of date.</span>
            <Button variant="outline" size="sm" className="min-h-11 w-full sm:ml-auto sm:w-auto" onClick={() => void fetchServerInfo()}>Retry</Button>
          </div>
        )}
         {pendingApprovalError ? (
           <div role="alert" className="flex flex-col items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300 sm:flex-row sm:items-center">
             <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span>{pendingApprovalError}</span>
            <Button variant="outline" size="sm" className="min-h-11 w-full sm:ml-auto sm:w-auto" onClick={() => void fetchPendingApprovals()}>Retry</Button>
          </div>
        ) : !isPendingApprovalLoading && hasPendingApprovals ? (
          <Card className="border-amber-500/30 bg-amber-500/5 dark:border-amber-800/70 dark:bg-amber-950/20">
            <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <p className="text-sm font-semibold text-foreground">
                  {formatDisplayNumber(pendingApprovalCount)} approval{pendingApprovalCount === 1 ? '' : 's'} waiting for review
                </p>
                <p className="text-sm text-muted-foreground">
                   These requests need an operator decision before they can proceed.
                </p>
              </div>
              <Link href="/dashboard/approvals" className={cn(buttonVariants({ size: 'sm' }), 'min-h-11 px-4')}>
                Review approvals
              </Link>
            </CardContent>
          </Card>
        ) : null}

      {dashboardLoadError ? (
        <div role="alert" className="flex flex-col items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300 sm:flex-row sm:items-center">
          <Server className="h-4 w-4 shrink-0" aria-hidden="true" />
          <span>{dashboardLoadError}</span>
          <Button variant="outline" size="sm" className="min-h-11 w-full sm:ml-auto sm:w-auto" onClick={() => void fetchDashboardData()}>Retry</Button>
        </div>
      ) : null}

      {isEmptyDashboard ? (
        <>
          <Card className="border-dashed">
            <CardHeader className="pb-3">
              <CardTitle>No operator activity yet</CardTitle>
              <CardDescription>
                This server is connected but has no recorded activity yet.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <p className="text-sm text-muted-foreground">
                Set your first policy to get started. Activity will appear here once agents begin using the server.
              </p>
              <div className="flex flex-col gap-2 sm:flex-row">
                <Link href="/dashboard/policies" className={cn(buttonVariants({ size: 'sm' }), 'min-h-11 px-4')}>
                  Set first policy
                </Link>
                <Link href="/dashboard/logs" className={cn(buttonVariants({ variant: 'outline', size: 'sm' }), 'min-h-11 px-4')}>
                  Open logs
                </Link>
              </div>
            </CardContent>
          </Card>

          <GettingStartedCard />
        </>
      ) : null}

      {/* Server Metrics */}
      <Card>
        <CardHeader className="pb-4">
          <CardTitle>30-Day Activity Snapshot</CardTitle>
          <CardDescription>Server uptime plus request, token, and tool activity from the last 30 days</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div className="text-center p-3 rounded-lg border bg-background/50 h-full flex flex-col gap-1 justify-center">
              <div className="text-sm text-muted-foreground">Uptime</div>
              <div
                className={
                  stats.uptime == null
                    ? 'text-sm text-muted-foreground'
                    : 'font-mono text-sm font-normal'
                }
              >
                {stats.uptime == null ? '—' : stats.uptime}
              </div>
            </div>
            <Link
              href="/dashboard/usage?timeRange=30"
              className="text-center p-3 rounded-lg border cursor-pointer hover:bg-blue-50 hover:border-blue-200 dark:hover:bg-blue-950/50 dark:hover:border-blue-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 transition-colors duration-200 h-full flex flex-col gap-1 justify-center"
            >
              <div className="text-sm text-muted-foreground">API Requests</div>
              <div
                className={
                  stats.apiRequests == null
                    ? 'text-sm text-muted-foreground'
                    : 'font-mono text-sm font-normal text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300'
                }
              >
                {formatDisplayNumber(stats.apiRequests, { compact: true })}
              </div>
            </Link>
            <Link
              href="/dashboard/usage/token-usage?timeRange=30"
              className="text-center p-3 rounded-lg border cursor-pointer hover:bg-blue-50 hover:border-blue-200 dark:hover:bg-blue-950/50 dark:hover:border-blue-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 transition-colors duration-200 h-full flex flex-col gap-1 justify-center"
            >
              <div className="text-sm text-muted-foreground">Active Tokens</div>
              <div
                className={
                  stats.activeTokens == null
                    ? 'text-sm text-muted-foreground'
                    : 'font-mono text-sm font-normal text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300'
                }
              >
                {formatDisplayNumber(stats.activeTokens, { compact: true })}
              </div>
            </Link>
            <Link
              href="/dashboard/usage/tool-usage?timeRange=30"
              className="text-center p-3 rounded-lg border cursor-pointer hover:bg-blue-50 hover:border-blue-200 dark:hover:bg-blue-950/50 dark:hover:border-blue-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 transition-colors duration-200 h-full flex flex-col gap-1 justify-center"
            >
              <div className="text-sm text-muted-foreground">
                Configured Tools
              </div>
              <div
                className={
                  stats.configuredTools == null
                    ? 'text-sm text-muted-foreground'
                    : 'font-mono text-sm font-normal text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300'
                }
              >
                {formatDisplayNumber(stats.configuredTools, { compact: true })}
              </div>
            </Link>

          </div>
        </CardContent>
      </Card>

      </div>


      {/* Recent Activity */}
      <Card>
          <CardHeader>
            <CardTitle>
              <Link href="/dashboard/logs?timeRange=30d" className="-ml-2 inline-flex min-h-11 items-center rounded-sm px-2 transition-colors duration-200 hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">Recent Activity</Link>
            </CardTitle>
            <CardDescription>Last 30 days</CardDescription>
          </CardHeader>
          <CardContent>
            {isDashboardLoading ? (
              <div className="flex min-h-[120px] items-center justify-center" role="status" aria-live="polite">
                <div className="text-center">
                  <div className="mx-auto mb-3 h-6 w-6 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground" aria-hidden="true" />
                  <p className="text-sm text-muted-foreground">Loading recent dashboard activity…</p>
                </div>
              </div>
            ) : dashboardLoadError ? (
              <div role="alert" className="flex flex-col items-center justify-center gap-2 py-8 text-center">
                <p className="text-sm text-red-600 dark:text-red-400">Recent dashboard activity could not be loaded for the last 30 days.</p>
                <Button variant="outline" size="sm" className="min-h-11" onClick={() => void fetchDashboardData()}>Retry</Button>
              </div>
            ) : !recentActivity || recentActivity.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <p className="text-sm text-muted-foreground">No recent activity yet in the last 30 days. Logs and approvals will appear here once the server is in use.</p>
              </div>
            ) : (
              <div className="space-y-3">
                {recentActivity.map((activity) => {
                  const statusMeta = getRecentActivityStatusMeta(activity.status)

                  return (
                    <Link
                      key={`${activity.action}-${formatNullableText(activity.time)}`}
                      href="/dashboard/logs?timeRange=30d"
                      aria-label={`Open logs for ${activity.action}, ${statusMeta.label}, ${formatNullableText(activity.time)}`}
                      className="-mx-2 flex items-center gap-3 rounded-md px-2 py-3 transition-colors duration-200 hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
                    >
                      <div className="flex-shrink-0">
                        <div className="flex items-center gap-2">
                          <div className={`h-2 w-2 rounded-full ${statusMeta.dot}`} />
                          <span className="text-xs text-muted-foreground">{statusMeta.label}</span>
                        </div>
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="text-sm">{activity.action}</p>
                        <p className="mt-1 text-xs text-muted-foreground">
                          {formatNullableText(activity.time)}
                        </p>
                      </div>
                    </Link>
                  )
                })}
              </div>
            )}
          </CardContent>
      </Card>
    </div>
  )
}
