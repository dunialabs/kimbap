'use client'

import {
  Server,
  CheckCircle,
  Shield,
  FileText,
  Eye,
  Globe,
  MapPin
} from 'lucide-react'
import Link from 'next/link'
import { useEffect, useState } from 'react'

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
  toolsUsage: Array<{ name: string; count: number }>
  tokenUsage: Array<{ name: string; requests: number; successRate: number }>
  connectedClients: Array<{ name: string; ip: string; lastSeen: string }>
  recentActivity: Array<{ action: string; time: string; user: string }>
  manualConnection: string | null
  sshTunnelAddress: string | null
}

export default function DashboardPage() {
  const [serverInfo, setServerInfo] = useState<ServerInfo | null>(null)
  const [isClientsDialogOpen, setIsClientsDialogOpen] = useState(false)
  const [dashboardData, setDashboardData] = useState<DashboardData | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [pendingApprovalCount, setPendingApprovalCount] = useState(0)


  useEffect(() => {
    const fetchServerInfo = async () => {
      try {
        const { api } = await import('@/lib/api-client')

        // Try to get cached server info from localStorage first
        const selectedServer = localStorage.getItem('selectedServer')
        if (selectedServer) {
          const parsedServer = JSON.parse(selectedServer)


          // Use cached server info if available
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
            return // Use cache, no need to call API
          }
        }

        // No cache found, fetch server info using protocol 10002
        const serverInfoResponse = await api.servers.getInfo()
        if (serverInfoResponse.data?.data) {
          const data = serverInfoResponse.data.data
          setServerInfo(data)

          // Update localStorage cache with fresh server info
          const selectedServer = localStorage.getItem('selectedServer')
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
            } catch (error) {
              // Failed to update localStorage cache:
            }
          }
        }
      } catch (error) {
        // Failed to fetch server info:
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

  useEffect(() => {
    const fetchDashboardData = async () => {
      if (!serverInfo?.proxyId) return

      try {
        const { api } = await import('@/lib/api-client')

        // Fetch dashboard overview data using protocol 10023
        const overviewResponse = await api.dashboard.overview(
          serverInfo.proxyId,
          '30d'
        )

        if (overviewResponse.data?.data) {
          setDashboardData(overviewResponse.data.data)
        }
      } catch (error) {
        // Failed to fetch dashboard data:
        setDashboardData(null)
      } finally {
        setIsLoading(false)
      }
    }

    fetchDashboardData()
  }, [serverInfo])

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

  if (!serverInfo) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Server className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-semibold mb-2">No Server Connected</h3>
          <p className="text-muted-foreground mb-4">
            Connect to a server to view your dashboard.
          </p>
          <Link href="/">
            <Button>Set Up Connection</Button>
          </Link>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div aria-live="polite" className="text-center">
          <div className="w-8 h-8 border-2 border-muted-foreground/30 border-t-foreground rounded-full animate-spin mx-auto mb-4" aria-hidden="true" />
          <h3 className="text-lg font-semibold mb-2">Loading Dashboard</h3>
          <p className="text-muted-foreground">Loading server data…</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="mb-0">
            <h1 className="text-[30px] font-bold tracking-tight">Dashboard</h1>
            <p className="text-[14px] text-foreground">
              {serverInfo?.proxyName || 'MCP Server'}
            </p>
          </div>
          <p className="text-base text-muted-foreground">
            Overview of your {serverInfo?.proxyName || 'server'}
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
        <div className="actions grid grid-cols-2 md:grid-cols-4 gap-3">
          <Link
            href="/dashboard/policies"
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="w-full flex items-center gap-1 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
              <Shield className="h-4 w-4" />
              <div className="action-content">Manage Policies</div>
            </Card>
          </Link>
          <Link
            href="/dashboard/approvals"
            aria-label={pendingApprovalCount > 0 ? `Review Approvals, ${pendingApprovalCount} pending` : undefined}
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="w-full flex items-center gap-1 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
              <CheckCircle className="h-4 w-4" />
              <div className="action-content">Review Approvals</div>
              {pendingApprovalCount > 0 && (
                <span className="inline-flex items-center justify-center rounded-full bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-300 text-xs font-medium min-w-[20px] h-5 px-1.5">
                  {pendingApprovalCount > 99 ? '99+' : pendingApprovalCount}
                </span>
              )}
            </Card>
          </Link>
          <Link
            href="/dashboard/usage"
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="w-full flex items-center gap-1 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
              <Eye className="h-4 w-4" />
              <div className="action-content">View Usage</div>
            </Card>
          </Link>
          <Link
            href="/dashboard/logs"
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="w-full flex items-center gap-1 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
              <FileText className="h-4 w-4" />
              <div className="action-content">View Logs</div>
            </Card>
          </Link>
        </div>
        {/* Connection Info */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <Card className="p-4 h-full">
              <div className="flex items-center justify-between h-full gap-3">
                <div className="flex flex-col gap-1 justify-center min-w-0">
                  <p className="text-sm text-muted-foreground">
                    Local Connection
                  </p>
                  <p className="font-mono text-sm font-normal break-words">
                    {dashboardData?.manualConnection || 'Not configured'}
                  </p>
                </div>
                <Globe className="h-5 w-5 text-muted-foreground" />
              </div>
          </Card>

          <Card className="p-4 h-full">
            <div className="flex items-center justify-between h-full gap-3">
              <div className="flex flex-col gap-1 justify-center min-w-0">
                <p className="text-sm text-muted-foreground">
                  Remote Connection
                </p>
                <p className="font-mono text-sm font-normal break-words">
                  {dashboardData?.sshTunnelAddress || 'Not configured'}
                </p>
              </div>
              <Globe className="h-5 w-5 text-muted-foreground" />
            </div>
          </Card>

          <Link
            href="/dashboard/usage/token-usage"
            className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            <Card className="p-4 cursor-pointer hover:bg-muted/50 transition-colors h-full">
              <div className="flex items-center justify-between h-full">
                <div className="flex flex-col gap-1 justify-center">
                  <p className="text-sm text-muted-foreground">
                    Active Tokens
                  </p>
                  <p className="font-mono text-sm font-normal text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300">
                    {stats.activeTokens == null ? 'No data' : stats.activeTokens}
                  </p>
                </div>
                <Shield className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              </div>
            </Card>
          </Link>
        </div>
      </div>

      {/* Server Metrics */}
      <Card>
        <CardHeader className="pb-4">
          <CardTitle>Server Metrics</CardTitle>
          <CardDescription>
            Server performance overview
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
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
              href="/dashboard/usage"
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
              href="/dashboard/usage/token-usage"
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
              href="/dashboard/usage/tool-usage"
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
            <button
              type="button"
              className="text-center p-3 rounded-lg border cursor-pointer hover:bg-blue-50 hover:border-blue-200 dark:hover:bg-blue-950/50 dark:hover:border-blue-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 transition-all duration-200 h-full flex flex-col gap-1 justify-center"
              onClick={() => setIsClientsDialogOpen(true)}
              aria-haspopup="dialog"
              aria-controls="connected-clients-dialog"
            >
              <div className="text-sm text-muted-foreground">
                Recent Clients (24h)
              </div>
              <div
                className={
                  stats.connectedClients == null
                    ? 'text-sm text-muted-foreground'
                    : 'font-mono text-sm font-normal text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300'
                }
              >
                {stats.connectedClients == null ? '—' : stats.connectedClients}
              </div>
            </button>
          </div>
        </CardContent>
      </Card>

      {/* Usage Overview */}
      {/* <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-6">
            <div className="flex-shrink-0">
              <CardTitle className="text-base font-bold">
                Monthly Usage
              </CardTitle>
              <CardDescription className="mt-1">
                API requests this month
              </CardDescription>
            </div>
            <div className="flex-1 flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <div className="text-xs">{stats.monthlyUsage}% of limit</div>
                <div className="text-xs text-muted-foreground whitespace-nowrap">
                  {Math.round(
                    (stats.apiRequests * 100) / stats.monthlyUsage
                  ).toLocaleString()}{' '}
                  / 100,000 requests
                </div>
              </div>
              <Progress
                value={stats.monthlyUsage}
                className="h-[8px] [&>div]:bg-slate-900 dark:[&>div]:bg-slate-100"
              />
            </div>
          </div>
        </CardContent>
      </Card> */}

      {/* Tools Usage */}
      <Card>
        <CardHeader>
          <CardTitle>Tool Usage</CardTitle>
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
              {toolsUsage.map((tool: any) => (
                <div key={tool.name} className="contents">
                  <div
                    className="text-sm max-w-[200px] truncate"
                    title={tool.name}
                  >
                    {tool.name}
                  </div>
                  <div className="min-w-0">
                    <Progress
                      value={tool.percentage}
                      aria-label={`${tool.name} usage ${tool.percentage}%`}
                      className="h-[8px] [&>div]:bg-slate-900 dark:[&>div]:bg-slate-100"
                    />
                  </div>
                  <div className="text-sm text-muted-foreground text-right whitespace-nowrap">
                    {tool.requests}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Access Token Usage */}
      <Card>
        <CardHeader>
          <CardTitle>Access Token Usage</CardTitle>
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
              {tokenUsage.map((token: any) => (
                <div key={token.token} className="contents">
                  <div
                    className="text-sm max-w-[200px] truncate"
                    title={token.name}
                  >
                    {token.name}
                  </div>
                  <div className="min-w-0">
                    <Progress
                      value={token.percentage}
                      aria-label={`${token.name} usage ${token.percentage}%`}
                      className="h-[8px] [&>div]:bg-slate-900 dark:[&>div]:bg-slate-100"
                    />
                  </div>
                  <div className="text-sm text-muted-foreground text-right whitespace-nowrap">
                    {token.requests}
                  </div>
                  <div className="text-xs text-muted-foreground font-mono text-right whitespace-nowrap hidden md:block">
                    {token.token}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Recent Activity & Quick Actions */}
      <div className="grid grid-cols-1">
        <Card>
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
            <CardDescription>Recent server events</CardDescription>
          </CardHeader>
          <CardContent>
            {!recentActivity || recentActivity.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <p className="text-sm text-muted-foreground">No recent activity in the last 30 days.</p>
              </div>
            ) : (
              <div className="space-y-3">
                {recentActivity.map((activity: any) => (
                  <div key={`${activity.action}-${activity.time}`} className="flex items-center gap-3">
                    <div className="flex-shrink-0">
                      {activity.status === 'success' && (
                        <div className="flex items-center gap-2">
                          <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                          <span className="text-[10px] text-muted-foreground">Success</span>
                        </div>
                      )}
                      {activity.status === 'warning' && (
                        <div className="flex items-center gap-2">
                          <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
                          <span className="text-[10px] text-muted-foreground">Warning</span>
                        </div>
                      )}
                      {activity.status === 'info' && (
                        <div className="flex items-center gap-2">
                          <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                          <span className="text-[10px] text-muted-foreground">Info</span>
                        </div>
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm">
                        {activity.action}
                        <span className="text-xs text-muted-foreground gap-1 ml-2">
                          {activity.time}
                        </span>
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <Dialog open={isClientsDialogOpen} onOpenChange={setIsClientsDialogOpen}>
        <DialogContent id="connected-clients-dialog" className="max-w-4xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Globe className="h-5 w-5" />
              Recent Clients (24h) ({connectedClients.length})
            </DialogTitle>
            <DialogDescription>
              Clients seen in the last 24 hours on your{' '}
              {serverInfo?.proxyName || 'MCP server'}
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
                      <TableHead>Access Token</TableHead>
                      <TableHead>IP Address</TableHead>
                      <TableHead>Location</TableHead>
                      <TableHead>Last Active</TableHead>
                      <TableHead className="text-right">Requests</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {connectedClients.map((client: any) => (
                      <TableRow key={client.id}>
                        <TableCell>{client.name}</TableCell>
                        <TableCell>
                          <code
                            className="inline-block max-w-[180px] truncate text-xs bg-muted px-2 py-1 rounded align-middle"
                            title={
                              client.token
                                ? `${client.token.slice(0, 8)}...${client.token.slice(-4)}`
                                : '-'
                            }
                          >
                            {client.token
                              ? `${client.token.slice(0, 8)}...${client.token.slice(-4)}`
                              : '-'}
                          </code>
                        </TableCell>
                        <TableCell className="font-mono text-sm">
                          {client.ip}
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
