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
  Copy
} from 'lucide-react'
import Link from 'next/link'
import { useEffect, useState, useCallback } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  const [serverFetchError, setServerFetchError] = useState(false)
  const [isServerInfoLoading, setIsServerInfoLoading] = useState(true)
  const [isClientsDialogOpen, setIsClientsDialogOpen] = useState(false)
  const [dashboardData, setDashboardData] = useState<DashboardData | null>(null)
  const [dashboardLoadError, setDashboardLoadError] = useState(false)
  const [pendingApprovalCount, setPendingApprovalCount] = useState(0)


  useEffect(() => {
    const fetchServerInfo = async () => {
      setServerFetchError(false)
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
      } catch (err: any) {
        const msg: string = err?.response?.data?.common?.message || err?.message || '';
        const isNotFound = msg.toLowerCase().includes('not found') || err?.response?.status === 404;
        if (!isNotFound) {
          setServerFetchError(true)
        }
      } finally {
        setIsServerInfoLoading(false)
      }
    }

    fetchServerInfo()
  }, [])

  useEffect(() => {
    const fetchPendingApprovals = async () => {
      try {
        const { api } = await import('@/lib/api-client')
        const res = await api.approvals.countPending()
        const data = res.data?.data || res.data
        setPendingApprovalCount(data?.count || 0)
      } catch {
        // Non-critical — don't block dashboard
      }
    }
    fetchPendingApprovals()
  }, [])

  const fetchDashboardData = useCallback(async () => {
    if (!serverInfo?.proxyId) return

    try {
      const { api } = await import('@/lib/api-client')

      // Fetch dashboard overview data using protocol 10023
      const overviewResponse = await api.dashboard.overview('30d')

      if (overviewResponse.data?.data) {
        setDashboardData(overviewResponse.data.data)
        setDashboardLoadError(false)
      } else {
        setDashboardData(null)
        setDashboardLoadError(true)
      }
    } catch {
      setDashboardData(null)
      setDashboardLoadError(true)
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

  const copyConnectionAddress = async (label: string, value: string | null) => {
    if (!value) {
      return
    }

    try {
      if (!navigator?.clipboard?.writeText) {
        toast.error('Clipboard not available')
        return
      }

      await navigator.clipboard.writeText(value)
      toast.success(`${label} copied`)
    } catch {
      toast.error(`Could not copy ${label.toLowerCase()}`)
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
        <div className="text-center">
          <Server className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-semibold mb-2">Could not reach server</h3>
          <p className="text-muted-foreground mb-4">
            Check your connection and retry. If the problem persists, return to sign in and reconnect.
          </p>
          <Button onClick={() => window.location.reload()}>Retry</Button>
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
          <Link href="/">
            <Button>Back to sign in</Button>
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-[30px] font-bold tracking-tight">Dashboard</h1>
          <p className="text-[14px] text-foreground">
            {serverInfo?.proxyName || 'Kimbap Server'}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <span className="relative flex h-3 w-3">
            {serverInfo?.status === 1 ? (
              <>
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
              </>
            ) : (
              <span className="relative inline-flex rounded-full h-3 w-3 bg-gray-400 dark:bg-gray-500"></span>
            )}
          </span>
          <Badge
            variant="secondary"
            className={
              serverInfo?.status === 1
                ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-950 dark:text-green-400 dark:border-green-900'
                : 'bg-gray-50 text-gray-700 border-gray-200 dark:bg-gray-950 dark:text-gray-300 dark:border-gray-800'
            }
          >
            {serverInfo?.status === 1 ? 'Running' : 'Stopped'}
          </Badge>
        </div>
      </div>
      {/* Header */}
      <div className="space-y-4">
        <GettingStartedCard />

        <div className="actions grid grid-cols-2 md:grid-cols-4 gap-3">
          <Link
            href="/dashboard/approvals"
            aria-label={pendingApprovalCount > 0 ? `Review Approvals, ${pendingApprovalCount} pending` : undefined}
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="w-full flex items-center gap-1 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
              <CheckCircle className="h-4 w-4" />
              <span className="text-sm font-medium">Review Approvals</span>
              {pendingApprovalCount > 0 && (
                <span className="inline-flex items-center justify-center rounded-full bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-300 text-xs font-medium min-w-[20px] h-5 px-1.5">
                  {pendingApprovalCount > 99 ? '99+' : pendingApprovalCount}
                </span>
              )}
            </Card>
          </Link>
          <Link
            href="/dashboard/logs"
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="w-full flex items-center gap-1 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
              <Activity className="h-4 w-4" />
              <span className="text-sm font-medium">Open Logs</span>
            </Card>
          </Link>
          <Link
            href="/dashboard/usage?timeRange=30"
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="w-full flex items-center gap-1 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
              <TrendingUp className="h-4 w-4" />
              <span className="text-sm font-medium">Open Usage</span>
            </Card>
          </Link>
          <Link
            href="/dashboard/policies"
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="w-full flex items-center gap-1 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
              <Shield className="h-4 w-4" />
              <span className="text-sm font-medium">Open Policies</span>
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
                    className="h-7 px-2 text-xs"
                    disabled={!localAddress}
                    onClick={() => void copyConnectionAddress('Local address', localAddress)}
                  >
                    <Copy className="mr-1 h-3.5 w-3.5" aria-hidden="true" />
                    Copy
                  </Button>
                </div>
                <p className={localAddress ? "font-mono text-sm font-normal break-words" : "text-sm text-muted-foreground"}>
                  {localAddress || 'Not configured'}
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
                    className="h-7 px-2 text-xs"
                    disabled={!remoteAddress}
                    onClick={() => void copyConnectionAddress('Remote address', remoteAddress)}
                  >
                    <Copy className="mr-1 h-3.5 w-3.5" aria-hidden="true" />
                    Copy
                  </Button>
                </div>
                <p className={remoteAddress ? "font-mono text-sm font-normal break-words" : "text-sm text-muted-foreground"}>
                  {remoteAddress || 'Not configured'}
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
            aria-label={`Open recent clients dialog${stats.connectedClients == null ? "" : `, ${stats.connectedClients} clients seen in the last 24 hours`}`}
          >
            <Card className="p-4 cursor-pointer hover:bg-muted/50 transition-colors h-full">
              <div className="flex items-center justify-between h-full">
                <div className="flex flex-col gap-1 justify-center">
                  <p className="text-sm text-muted-foreground">
                    Recent clients (24h)
                  </p>
                  <p className="font-mono text-sm font-normal">
                    {stats.connectedClients == null ? '—' : stats.connectedClients}
                  </p>
                </div>
                <Users className="h-5 w-5 text-muted-foreground" />
              </div>
            </Card>
          </button>
        </div>
      </div>

      {dashboardLoadError ? (
        <div className="flex items-center gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300">
          <Server className="h-4 w-4 shrink-0" aria-hidden="true" />
          <span>Could not load dashboard metrics. Check your connection and try again.</span>
          <Button variant="outline" size="sm" className="ml-auto" onClick={() => void fetchDashboardData()}>Retry</Button>
        </div>
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
                {stats.apiRequests == null
                  ? '—'
                  : stats.apiRequests.toLocaleString()}
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
                {stats.activeTokens == null ? '—' : stats.activeTokens}
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
                {stats.configuredTools == null ? '—' : stats.configuredTools}
              </div>
            </Link>

          </div>
        </CardContent>
      </Card>

      {/* Tools Usage */}
      <Card>
        <CardHeader>
          <CardTitle>
            <Link href="/dashboard/usage/tool-usage?timeRange=30" className="rounded-sm hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">Tool Usage</Link>
          </CardTitle>
          <CardDescription>
            Requests by tool over the last 30 days
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!toolsUsage || toolsUsage.length === 0 ? (
            <div className="flex items-center justify-center py-8">
              <p className="text-sm text-muted-foreground">No tool requests in the last 30 days.</p>
            </div>
          ) : (
            <div
              className="grid gap-y-3 gap-x-2 items-center"
              style={{ gridTemplateColumns: 'max-content 1fr max-content' }}
            >
              {toolsUsage.map((tool) => (
                <Link
                  key={tool.name}
                  href="/dashboard/usage/tool-usage?timeRange=30"
                  className="group contents focus-visible:outline-none"
                >
                  <div className="text-sm max-w-[200px] truncate rounded-sm group-hover:underline group-focus-visible:underline" title={tool.name}>{tool.name}</div>
                  <div className="min-w-0 rounded-sm">
                    <Progress
                      value={tool.percentage}
                      aria-label={`${tool.name} usage ${tool.percentage}%`}
                      className="h-[8px] [&>div]:bg-slate-900 dark:[&>div]:bg-slate-100"
                    />
                  </div>
                  <div className="text-sm text-muted-foreground text-right whitespace-nowrap rounded-sm">
                    {typeof tool.requests === 'number' ? tool.requests.toLocaleString() : tool.requests}
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
            <Link href="/dashboard/usage/token-usage?timeRange=30" className="rounded-sm hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">Access Token Usage</Link>
          </CardTitle>
          <CardDescription>
            Requests by token over the last 30 days
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!tokenUsage || tokenUsage.length === 0 ? (
            <div className="flex items-center justify-center py-8">
              <p className="text-sm text-muted-foreground">No token requests in the last 30 days.</p>
            </div>
          ) : (
            <div
              className="grid gap-y-3 gap-x-2 items-center grid-cols-[max-content_1fr_max-content] md:grid-cols-[max-content_1fr_max-content_max-content]"
            >
              {tokenUsage.map((token) => (
                <Link
                  key={`${token.name || 'token'}-${token.token?.trim() || ''}`}
                  href="/dashboard/usage/token-usage?timeRange=30"
                  className="group contents focus-visible:outline-none"
                >
                  <div
                    className="text-sm max-w-[200px] truncate rounded-sm group-hover:underline group-focus-visible:underline"
                    title={token.name}
                  >
                    {token.name}
                  </div>
                  <div className="min-w-0 rounded-sm">
                    <Progress
                      value={token.percentage}
                      aria-label={`${token.name} usage ${token.percentage}%`}
                      className="h-[8px] [&>div]:bg-slate-900 dark:[&>div]:bg-slate-100"
                    />
                  </div>
                  <div className="text-sm text-muted-foreground text-right whitespace-nowrap rounded-sm">
                    {typeof token.requests === 'number' ? token.requests.toLocaleString() : token.requests}
                  </div>
                  <div className="text-xs text-muted-foreground font-mono text-right whitespace-nowrap hidden md:block rounded-sm">
                    {token.token?.trim() || '-'}
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
              <Link href="/dashboard/logs?timeRange=30d" className="rounded-sm hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">Recent Activity</Link>
            </CardTitle>
            <CardDescription>Last 30 days</CardDescription>
          </CardHeader>
          <CardContent>
            {!recentActivity || recentActivity.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <p className="text-sm text-muted-foreground">No recent activity in the last 30 days.</p>
              </div>
            ) : (
              <div className="space-y-3">
                {recentActivity.map((activity) => (
                  <Link
                    key={`${activity.action}-${activity.time}`}
                    href="/dashboard/logs?timeRange=30d"
                    className="flex items-center gap-3 rounded-md p-1 -m-1 hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
                  >
                    <div className="flex-shrink-0">
                      {(() => {
                        const statusMap: Record<string, { dot: string; label: string }> = {
                          success: { dot: 'bg-green-500', label: 'Success' },
                          warning: { dot: 'bg-yellow-500', label: 'Warning' },
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
                          {activity.time}
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
            <DialogTitle className="flex items-center gap-2">
              <Users className="h-5 w-5" />
              Recent Clients (24h) ({connectedClients.length.toLocaleString()})
            </DialogTitle>
            <DialogDescription>
              Clients seen in the last 24 hours on your{' '}
              {serverInfo?.proxyName || 'Kimbap Server'}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {connectedClients.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <p className="text-sm text-muted-foreground">No clients seen in the last 24 hours.</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Client Name</TableHead>
                      <TableHead>IP Address</TableHead>
                      <TableHead>Location</TableHead>
                      <TableHead>Last Active</TableHead>
                      <TableHead className="text-right">Requests</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {connectedClients.map((client) => (
                      <TableRow key={client.id}>
                        <TableCell>{client.name}</TableCell>
                        <TableCell className="font-mono text-sm">
                          <div className="flex items-center gap-2">
                            <span>{client.ip}</span>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              className="h-7 px-2 text-xs"
                              onClick={async () => {
                                try {
                                  if (!navigator?.clipboard?.writeText) {
                                    toast.error('Clipboard not available')
                                    return
                                  }
                                  await navigator.clipboard.writeText(client.ip)
                                  toast.success('Client IP copied')
                                } catch {
                                  toast.error('Could not copy client IP')
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
                            <span className="text-sm">{client.location}</span>
                          </div>
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {client.lastActive}
                        </TableCell>
                        <TableCell className="text-right">
                          {client.requests.toLocaleString()}
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
