'use client'

import {
  Server,
  CheckCircle,
  Shield,
  Activity,
  Globe,
  MapPin,
  Users,
  TrendingUp,
  Copy,
  AlertTriangle
} from 'lucide-react'
import Link from 'next/link'
import { useEffect, useState, useCallback } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { Progress } from '@/components/ui/progress'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'
import { GettingStartedCard } from '@/components/getting-started-card'
import { cn, formatDisplayNumber, formatNullableText, formatPercentage } from '@/lib/utils'

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
  connectedClientsCount: number | null
  uptime: string | null
  monthlyUsage: number
  toolsUsage: Array<{ name: string; percentage: number; requests: number }>
  tokenUsage: Array<{ name: string; token: string; percentage: number; requests: number }>
  connectedClients: Array<{ id: string; name: string; ip: string; location: string; lastActive: string; requests: number }>
  recentActivity: Array<{ action: string; time: string; status: string }>
  manualConnection: string | null
  sshTunnelAddress: string | null
}

export default function DashboardPage() {
  const [serverInfo, setServerInfo] = useState<ServerInfo | null>(null)
  const [serverFetchError, setServerFetchError] = useState<string | null>(null)
  const [isServerInfoLoading, setIsServerInfoLoading] = useState(true)
  const [isClientsDialogOpen, setIsClientsDialogOpen] = useState(false)
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
      if (!isNotFound) {
        setServerFetchError(
          getRequestErrorMessage(error, {
            auth: 'Your session expired or you no longer have access to this server. Sign in again and reconnect.',
            network: 'Could not reach this server. Check your network connection and retry.',
            fallback: 'Could not load server details for this dashboard. Retry or sign in again.'
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
          auth: 'Could not check pending approvals because your session expired or your access changed. Sign in again to continue.',
          network: 'Could not check the approval queue. Check your connection and retry.',
          fallback: 'Could not load the pending approval count right now. Retry to check the queue again.'
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
        setDashboardLoadError('Could not load dashboard metrics for the last 30 days. Retry to load usage, recent activity, and addresses again.')
      }
    } catch (error: unknown) {
      setDashboardData(null)
      setDashboardLoadError(
        getRequestErrorMessage(error, {
          auth: 'Could not load dashboard metrics because your session expired or your access changed. Sign in again and retry.',
          network: 'Could not load dashboard metrics. Check your connection and retry.',
          fallback: 'Could not load dashboard metrics for the last 30 days. Retry to load usage, recent activity, and addresses again.'
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
    connectedClients: dashboardData?.connectedClientsCount ?? null,
    uptime: dashboardData?.uptime ?? null,
    monthlyUsage: dashboardData?.monthlyUsage ?? 0
  }

  const toolsUsage = dashboardData?.toolsUsage || []
  const tokenUsage = dashboardData?.tokenUsage || []
  const connectedClients = dashboardData?.connectedClients || []
  const recentActivity = dashboardData?.recentActivity || []
  const localAddress = dashboardData?.manualConnection ?? null
  const remoteAddress = dashboardData?.sshTunnelAddress ?? null
  const isDashboardLoading = !dashboardData && !dashboardLoadError
  const hasPendingApprovals = (pendingApprovalCount ?? 0) > 0
  const approvalSummaryText = isPendingApprovalLoading
    ? 'Checking the approval queue…'
    : hasPendingApprovals
    ? `Review ${formatDisplayNumber(pendingApprovalCount)} request${pendingApprovalCount === 1 ? '' : 's'} waiting on a decision.`
    : 'No approvals yet. Requests that need an operator decision will appear here.'
  const localAddressText = isDashboardLoading ? 'Loading address…' : formatNullableText(localAddress)
  const remoteAddressText = isDashboardLoading ? 'Loading address…' : formatNullableText(remoteAddress)
  const hasRecordedDashboardActivity =
    (stats.apiRequests ?? 0) > 0 ||
    (stats.connectedClients ?? 0) > 0 ||
    toolsUsage.length > 0 ||
    tokenUsage.length > 0 ||
    recentActivity.length > 0 ||
    hasPendingApprovals
  const isEmptyDashboard =
    !isDashboardLoading &&
    !dashboardLoadError &&
    !pendingApprovalError &&
    (stats.configuredTools ?? 0) === 0 &&
    !hasRecordedDashboardActivity


  const copyConnectionAddress = async (label: string, value: string | null) => {
    if (!value) {
      return
    }

    try {
      if (!navigator?.clipboard?.writeText) {
        toast.error('Clipboard is unavailable in this browser.')
        return
      }

      await navigator.clipboard.writeText(value)
      toast.success(`${label} copied`)
    } catch {
      toast.error(`Could not copy ${label.toLowerCase()}. Try again.`)
    }
  }

  if (isServerInfoLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div aria-live="polite" className="text-center">
          <div className="w-8 h-8 border-2 border-muted-foreground/30 border-t-foreground rounded-full animate-spin mx-auto mb-4" aria-hidden="true" />
          <h3 className="text-lg font-semibold mb-2">Loading dashboard overview</h3>
          <p className="text-muted-foreground">Fetching server metrics, addresses, and recent activity…</p>
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
            Return to sign in and reconnect to start using the dashboard.
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
          <span className="relative flex h-3 w-3">
            {serverInfo?.status === 1 ? (
              <>
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
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
        </div>
      </div>
      <div className="space-y-6">
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
                  Requests that need an operator decision are waiting in the approval queue.
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
                This server is connected, but no tools, approvals, logs, or recent clients have shown up yet.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <p className="text-sm text-muted-foreground">
                Start by setting your first policy. When agents begin using the server, activity will appear here automatically.
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
          <CardTitle>Server Metrics</CardTitle>
          <CardDescription>Last 30 days</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div className="text-center p-3 rounded-lg border bg-muted/20 h-full flex flex-col gap-1 justify-center">
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
              className="text-center p-3 rounded-lg border cursor-pointer hover:bg-blue-50 hover:border-blue-200 dark:hover:bg-blue-950/50 dark:hover:border-blue-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 transition-all duration-200 h-full flex flex-col gap-1 justify-center"
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
              className="text-center p-3 rounded-lg border cursor-pointer hover:bg-blue-50 hover:border-blue-200 dark:hover:bg-blue-950/50 dark:hover:border-blue-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 transition-all duration-200 h-full flex flex-col gap-1 justify-center"
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
              className="text-center p-3 rounded-lg border cursor-pointer hover:bg-blue-50 hover:border-blue-200 dark:hover:bg-blue-950/50 dark:hover:border-blue-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 transition-all duration-200 h-full flex flex-col gap-1 justify-center"
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

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <Link
            href="/dashboard/approvals"
            aria-label={!isPendingApprovalLoading && hasPendingApprovals ? `Review Approvals, ${pendingApprovalCount} pending` : undefined}
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card
              className={cn(
                'h-full cursor-pointer transition-colors',
                hasPendingApprovals
                  ? 'border-amber-300 bg-amber-50/80 hover:bg-amber-50 dark:border-amber-800 dark:bg-amber-950/20 dark:hover:bg-amber-950/30'
                  : 'hover:bg-muted/50'
              )}
            >
              <div className="flex h-full items-start justify-between gap-3 p-4">
                <div className="min-w-0 space-y-1 text-left">
                  <div className="flex flex-wrap items-center gap-2">
                    <CheckCircle className="h-4 w-4" aria-hidden="true" />
                    <span className="text-sm font-semibold">Approvals</span>
                    {hasPendingApprovals && (
                      <span className="inline-flex items-center justify-center rounded-full bg-amber-100 px-1.5 text-xs font-medium text-amber-800 dark:bg-amber-900 dark:text-amber-300">
                        {(pendingApprovalCount ?? 0) > 99 ? '99+' : pendingApprovalCount} pending
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {approvalSummaryText}
                  </p>
                </div>
              </div>
            </Card>
          </Link>
          <Link
            href="/dashboard/logs"
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="h-full cursor-pointer transition-colors hover:bg-muted/50">
              <div className="flex h-full items-start justify-between gap-3 p-4">
                <div className="min-w-0 space-y-1 text-left">
                  <div className="flex items-center gap-2">
                    <Activity className="h-4 w-4" aria-hidden="true" />
                    <span className="text-sm font-semibold">Logs</span>
                  </div>
                  <p className="text-xs text-muted-foreground">Watch first requests, investigate errors, and inspect live activity.</p>
                </div>
              </div>
            </Card>
          </Link>
          <Link
            href="/dashboard/usage?timeRange=30"
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="h-full cursor-pointer transition-colors hover:bg-muted/50">
              <div className="flex h-full items-start justify-between gap-3 p-4">
                <div className="min-w-0 space-y-1 text-left">
                  <div className="flex items-center gap-2">
                    <TrendingUp className="h-4 w-4" aria-hidden="true" />
                    <span className="text-sm font-semibold">Usage</span>
                  </div>
                  <p className="text-xs text-muted-foreground">Review request volume, token activity, and tool trends once traffic starts.</p>
                </div>
              </div>
            </Card>
          </Link>
          <Link
            href="/dashboard/policies"
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="h-full cursor-pointer transition-colors hover:bg-muted/50">
              <div className="flex h-full items-start justify-between gap-3 p-4">
                <div className="min-w-0 space-y-1 text-left">
                  <div className="flex items-center gap-2">
                    <Shield className="h-4 w-4" aria-hidden="true" />
                    <span className="text-sm font-semibold">Policies</span>
                  </div>
                  <p className="text-xs text-muted-foreground">Create your first policy or adjust allow, approval, and block rules for tool calls.</p>
                </div>
              </div>
            </Card>
          </Link>
        </div>
        {/* Connection Info */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <Card className="p-4 h-full">
            <div className="flex items-center justify-between h-full gap-3">
              <div className="flex flex-col gap-1 justify-center min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <p className="text-sm text-muted-foreground">
                    Local address
                  </p>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="min-h-11 px-3 text-xs"
                    disabled={!localAddress || isDashboardLoading}
                    onClick={() => void copyConnectionAddress('Local address', localAddress)}
                  >
                    <Copy className="mr-1 h-3.5 w-3.5" aria-hidden="true" />
                    Copy
                  </Button>
                </div>
                <p className={localAddress && !isDashboardLoading ? "font-mono text-sm font-normal break-words" : "text-sm text-muted-foreground"}>
                  {localAddressText}
                </p>
              </div>
              <MapPin className="h-5 w-5 text-muted-foreground" />
            </div>
          </Card>

          <Card className="p-4 h-full">
            <div className="flex items-center justify-between h-full gap-3">
              <div className="flex flex-col gap-1 justify-center min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <p className="text-sm text-muted-foreground">
                    Remote address
                  </p>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="min-h-11 px-3 text-xs"
                    disabled={!remoteAddress || isDashboardLoading}
                    onClick={() => void copyConnectionAddress('Remote address', remoteAddress)}
                  >
                    <Copy className="mr-1 h-3.5 w-3.5" aria-hidden="true" />
                    Copy
                  </Button>
                </div>
                <p className={remoteAddress && !isDashboardLoading ? "font-mono text-sm font-normal break-words" : "text-sm text-muted-foreground"}>
                  {remoteAddressText}
                </p>
              </div>
              <Globe className="h-5 w-5 text-muted-foreground" />
            </div>
          </Card>

          <button
            type="button"
            className="block w-full rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
            onClick={() => setIsClientsDialogOpen(true)}
            aria-haspopup="dialog"
            aria-controls="connected-clients-dialog"
            aria-label={`Open recent clients dialog${stats.connectedClients == null ? "" : `, ${formatDisplayNumber(stats.connectedClients)} clients seen in the last 24 hours`}`}
          >
            <Card className="p-4 cursor-pointer hover:bg-muted/50 transition-colors h-full">
              <div className="flex items-center justify-between h-full">
                <div className="flex flex-col gap-1 justify-center">
                  <p className="text-sm text-muted-foreground">
                    Recent clients (24 hours)
                  </p>
                  <p className={stats.connectedClients == null ? 'text-sm text-muted-foreground' : 'font-mono text-sm font-normal'}>
                    {formatDisplayNumber(stats.connectedClients, { compact: true })}
                  </p>
                </div>
                <Users className="h-5 w-5 text-muted-foreground" />
              </div>
            </Card>
          </button>
        </div>
      </div>


      {/* Tools Usage */}
      <Card>
        <CardHeader>
          <CardTitle>
            <Link href="/dashboard/usage/tool-usage?timeRange=30" className="-ml-2 inline-flex min-h-11 items-center rounded-sm px-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">Tool Usage</Link>
          </CardTitle>
          <CardDescription>
            Requests by tool over the last 30 days
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isDashboardLoading ? (
            <div className="flex min-h-[120px] items-center justify-center" role="status" aria-live="polite">
              <div className="text-center">
                <div className="mx-auto mb-3 h-6 w-6 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground" aria-hidden="true" />
                <p className="text-sm text-muted-foreground">Loading tool activity for the last 30 days…</p>
              </div>
            </div>
          ) : dashboardLoadError ? (
            <div role="alert" className="flex flex-col items-center justify-center gap-2 py-8 text-center">
              <p className="text-sm text-red-600 dark:text-red-400">Tool usage could not be loaded for the last 30 days.</p>
              <Button variant="outline" size="sm" className="min-h-11" onClick={() => void fetchDashboardData()}>Retry</Button>
            </div>
          ) : !toolsUsage || toolsUsage.length === 0 ? (
            <div className="flex items-center justify-center py-8">
              <p className="text-sm text-muted-foreground">No tool requests yet in the last 30 days. Activity will appear here after the first calls run.</p>
            </div>
          ) : (
            <div className="space-y-2">
              {toolsUsage.map((tool) => (
                <Link
                  key={tool.name}
                  href="/dashboard/usage/tool-usage?timeRange=30"
                  className="group block rounded-md px-2 py-3 -mx-2 transition-colors hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
                >
                  <div className="flex items-center gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-medium group-hover:underline" title={tool.name}>
                        {tool.name}
                      </div>
                      <div className="mt-2">
                        <Progress
                          value={tool.percentage}
                          aria-label={`${tool.name} usage ${formatPercentage(tool.percentage)}`}
                          className="h-[8px] [&>div]:bg-slate-900 dark:[&>div]:bg-slate-100"
                        />
                      </div>
                    </div>
                    <div className="whitespace-nowrap text-sm text-muted-foreground">
                      {typeof tool.requests === 'number' ? formatDisplayNumber(tool.requests, { compact: true }) : tool.requests}
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Access Token Usage */}
      <Card>
        <CardHeader>
          <CardTitle>
            <Link href="/dashboard/usage/token-usage?timeRange=30" className="-ml-2 inline-flex min-h-11 items-center rounded-sm px-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">Access Token Usage</Link>
          </CardTitle>
          <CardDescription>
            Requests by token over the last 30 days
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isDashboardLoading ? (
            <div className="flex min-h-[120px] items-center justify-center" role="status" aria-live="polite">
              <div className="text-center">
                <div className="mx-auto mb-3 h-6 w-6 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground" aria-hidden="true" />
                <p className="text-sm text-muted-foreground">Loading access token activity for the last 30 days…</p>
              </div>
            </div>
          ) : dashboardLoadError ? (
            <div role="alert" className="flex flex-col items-center justify-center gap-2 py-8 text-center">
              <p className="text-sm text-red-600 dark:text-red-400">Access token activity could not be loaded for the last 30 days.</p>
              <Button variant="outline" size="sm" className="min-h-11" onClick={() => void fetchDashboardData()}>Retry</Button>
            </div>
          ) : !tokenUsage || tokenUsage.length === 0 ? (
            <div className="flex items-center justify-center py-8">
              <p className="text-sm text-muted-foreground">No access token activity yet in the last 30 days.</p>
            </div>
          ) : (
            <div className="space-y-2">
              {tokenUsage.map((token) => (
                <Link
                  key={`${token.name || 'token'}-${token.token?.trim() || ''}`}
                  href="/dashboard/usage/token-usage?timeRange=30"
                  className="group block rounded-md px-2 py-3 -mx-2 transition-colors hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
                >
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-3">
                    <div className="min-w-0 sm:w-[220px]">
                      <div className="truncate text-sm font-medium group-hover:underline" title={token.name}>
                        {token.name}
                      </div>
                      <div className="mt-1 font-mono text-xs text-muted-foreground sm:hidden">
                        {formatNullableText(token.token)}
                      </div>
                    </div>
                    <div className="min-w-0 flex-1">
                      <Progress
                        value={token.percentage}
                        aria-label={`${token.name} usage ${formatPercentage(token.percentage)}`}
                        className="h-[8px] [&>div]:bg-slate-900 dark:[&>div]:bg-slate-100"
                      />
                    </div>
                    <div className="flex items-center justify-between gap-3 sm:block sm:text-right">
                      <div className="whitespace-nowrap text-sm text-muted-foreground">
                        {typeof token.requests === 'number' ? formatDisplayNumber(token.requests, { compact: true }) : token.requests}
                      </div>
                      <div className="hidden whitespace-nowrap font-mono text-xs text-muted-foreground sm:block">
                        {formatNullableText(token.token)}
                      </div>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Recent Activity */}
      <Card>
          <CardHeader>
            <CardTitle>
              <Link href="/dashboard/logs?timeRange=30d" className="-ml-2 inline-flex min-h-11 items-center rounded-sm px-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">Recent Activity</Link>
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
                {recentActivity.map((activity) => (
                  <Link
                    key={`${activity.action}-${formatNullableText(activity.time)}`}
                    href="/dashboard/logs?timeRange=30d"
                    className="flex items-center gap-3 rounded-md px-2 py-3 -mx-2 hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
                  >
                    <div className="flex-shrink-0">
                      {(() => {
                        const statusMap: Record<string, { dot: string; label: string }> = {
                          success: { dot: 'bg-green-500', label: 'Success' },
                          warning: { dot: 'bg-amber-500', label: 'Warning' },
                          info:    { dot: 'bg-blue-500',  label: 'Info' },
                          error:   { dot: 'bg-red-500',   label: 'Error' },
                        }
                        const s = statusMap[activity.status] ?? { dot: 'bg-muted-foreground', label: activity.status }
                        return (
                          <div className="flex items-center gap-2">
                            <div className={`w-2 h-2 rounded-full ${s.dot}`} />
                            <span className="text-xs text-muted-foreground">{s.label}</span>
                          </div>
                        )
                      })()}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm">
                        {activity.action}
                        <span className="text-xs text-muted-foreground gap-1 ml-2">
                          {formatNullableText(activity.time)}
                        </span>
                      </p>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
      </Card>


      <Dialog open={isClientsDialogOpen} onOpenChange={setIsClientsDialogOpen}>
        <DialogContent id="connected-clients-dialog" className="max-w-4xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex flex-wrap items-center gap-2">
              <Users className="h-5 w-5" />
              Recent Clients (24 hours) ({formatDisplayNumber(connectedClients.length)})
            </DialogTitle>
            <DialogDescription>
              Clients seen in the last 24 hours on your{' '}
              {serverInfo?.proxyName || 'Kimbap Server'}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {isDashboardLoading ? (
              <div className="flex min-h-[240px] items-center justify-center" role="status" aria-live="polite">
                <div className="text-center">
                  <div className="mx-auto mb-3 h-6 w-6 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground" aria-hidden="true" />
                  <p className="text-sm text-muted-foreground">Loading recent client details…</p>
                </div>
              </div>
            ) : dashboardLoadError ? (
              <div role="alert" className="flex flex-col items-center justify-center gap-2 py-8 text-center">
                <p className="text-sm text-red-600 dark:text-red-400">Recent client details could not be loaded.</p>
                <Button variant="outline" size="sm" className="min-h-11" onClick={() => void fetchDashboardData()}>Retry</Button>
              </div>
            ) : connectedClients.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <p className="text-sm text-muted-foreground">No clients seen yet in the last 24 hours.</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <Table className="min-w-[720px]">
                  <TableHeader>
                    <TableRow>
                      <TableHead scope="col">Client Name</TableHead>
                      <TableHead scope="col">IP Address</TableHead>
                      <TableHead scope="col">Location</TableHead>
                      <TableHead scope="col">Last Active</TableHead>
                      <TableHead scope="col" className="text-right">Requests</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {connectedClients.map((client) => (
                      <TableRow key={client.id}>
                        <TableCell>{formatNullableText(client.name)}</TableCell>
                        <TableCell className="font-mono text-sm">
                          <div className="flex items-center gap-2">
                            <span>{client.ip}</span>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              className="min-h-11 px-3 text-xs"
                              onClick={async () => {
                                try {
                                  if (!navigator?.clipboard?.writeText) {
                                    toast.error('Clipboard is unavailable in this browser.')
                                    return
                                  }
                                  await navigator.clipboard.writeText(client.ip)
                                  toast.success('Client IP copied')
                                } catch {
                                  toast.error('Could not copy the client IP. Try again.')
                                }
                              }}
                            >
                              <Copy className="mr-1 h-3.5 w-3.5" aria-hidden="true" />
                              Copy
                            </Button>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1">
                            <MapPin className="h-3 w-3 text-muted-foreground" />
                            <span className="text-sm">{formatNullableText(client.location)}</span>
                          </div>
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {formatNullableText(client.lastActive)}
                        </TableCell>
                        <TableCell className="text-right">
                          {formatDisplayNumber(client.requests, { compact: true })}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
