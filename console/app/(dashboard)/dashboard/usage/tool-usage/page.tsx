"use client"

import { Wrench, TrendingUp, CheckCircle, XCircle, Clock, Activity, Zap, RefreshCw, AlertTriangle } from "lucide-react"
import Link from "next/link"
import { Suspense, useState, useEffect, useCallback, useRef } from "react"
import { usePathname, useRouter, useSearchParams } from "next/navigation"
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  LineChart,
  Line,
  PieChart,
  Pie,
  Cell,
  Legend,
} from "recharts"

import { Badge } from "@/components/ui/badge"
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
import { Progress } from "@/components/ui/progress"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { api } from '@/lib/api-client'
import { getActionLabel } from '@/lib/log-utils'
import { formatDateTime, formatDisplayNumber, formatNullableText, formatPercentage } from '@/lib/utils'

interface ToolUsageData {
  toolId?: string
  toolName: string
  totalRequests: number
  successfulRequests: number
  failedRequests: number
  averageResponseTime: number
  successRate: number
  lastUsed: string
  status: "active" | "inactive" | "error"
  errorTypes: Array<{
    type: string
    count: number
  }>
}

interface ToolTrendData {
  date: string
  [key: string]: string | number
}

interface ToolErrorAnalysis {
  toolId: string
  toolName: string
  totalErrors: number
  errorTypes: Array<{
    errorCode: string
    errorMessage: string
    count: number
    percentage: number
  }>
}

interface ActionLog {
  logId: string
  actionType: number
  actionName: string
  toolId: string
  toolName: string
  userId: string
  userName: string
  timestamp: number
  responseTime: number
  status: number
  errorMessage: string
  details: string
}

interface ActionLogsResponse {
  logs: ActionLog[]
  totalCount: number
}

interface ToolMetricsResponse {
  tools: ToolUsageData[]
  totalCount: number
}

interface ToolUsageSummary {
  totalTools: number
  activeTools: number
  totalRequests: number
  avgSuccessRate: number
  avgResponseTime: number
}

const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8', '#82CA9D', '#FF6B6B', '#4ECDC4', '#A855F7', '#F97316']
const HEALTHY_SUCCESS_RATE_THRESHOLD = 95

function ToolUsagePageContent() {
  const searchParams = useSearchParams()
  const pathname = usePathname()
  const router = useRouter()
  const [timeRange, setTimeRange] = useState(() => {
    const param = searchParams.get('timeRange')
    const num = param ? Number(param) : NaN
    return [1, 7, 30].includes(num) ? num : 1
  })
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState(() => {
    const requestedTab = searchParams.get('tab')
    return ['overview', 'performance', 'errors', 'trends', 'actions'].includes(requestedTab || '') ? requestedTab || 'overview' : 'overview'
  })
  const [actionLoading, setActionLoading] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [toolDataError, setToolDataError] = useState<string | null>(null)
  const [trendDataError, setTrendDataError] = useState<string | null>(null)
  const [errorDataError, setErrorDataError] = useState<string | null>(null)
  const [actionLogsError, setActionLogsError] = useState<string | null>(null)
  const [actionTypeOptionsError, setActionTypeOptionsError] = useState<string | null>(null)
  const [summary, setSummary] = useState<ToolUsageSummary | null>(null)
  const [toolUsageData, setToolUsageData] = useState<ToolUsageData[]>([])
  const [trendData, setTrendData] = useState<ToolTrendData[]>([])
  const [errorData, setErrorData] = useState<ToolErrorAnalysis[]>([])
  const [actionLogs, setActionLogs] = useState<ActionLog[]>([])
  const [actionLogsTotal, setActionLogsTotal] = useState(0)
  const [actionLogsPage, setActionLogsPage] = useState(1)
  const [actionLogStatus, setActionLogStatus] = useState<'all' | 'success' | 'failed'>('all')
  const [actionLogToolId, setActionLogToolId] = useState('all')
  const [actionLogType, setActionLogType] = useState('all')
  const [actionTypeOptions, setActionTypeOptions] = useState<number[]>([])
  const [actionToolOptions, setActionToolOptions] = useState<Array<{ toolId: string; toolName: string }>>([])
  const timeRangeLabel = timeRange === 1 ? '24 hours' : `${timeRange} days`
  const hasDataRef = useRef(false)

  useEffect(() => {
    const tabTitles: Record<string, string> = {
      overview: 'Tool Usage',
      performance: 'Tool Usage Performance',
      errors: 'Tool Usage Error Analysis',
      trends: 'Tool Usage Trends',
      actions: 'Tool Action Logs'
    }
    const tabTitle = tabTitles[activeTab] || 'Tool Usage'

    document.title = `${tabTitle} | Kimbap Console`
  }, [activeTab])

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

  useEffect(() => {
    const currentTab = searchParams.get('tab')
    if (currentTab === activeTab) {
      return
    }

    const params = new URLSearchParams(searchParams.toString())
    params.set('tab', activeTab)
    router.replace(`${pathname}?${params.toString()}`, { scroll: false })
  }, [activeTab, pathname, router, searchParams])

  const fetchToolUsageData = useCallback(async () => {
    try {
      if (!hasDataRef.current) setLoading(true)

      const [summaryResult, trendsResult, errorsResult, metricsCountResult] = await Promise.allSettled([
        api.usage.getToolUsageSummary({ timeRange }),
        api.usage.getToolTrends({ timeRange, granularity: 2 }),
        api.usage.getToolErrors({ timeRange }),
        api.usage.getToolMetrics({ timeRange, page: 1, pageSize: 1 })
      ])

      let summaryFailure = false

      if (summaryResult.status === 'fulfilled' && summaryResult.value.data?.common?.code === 0 && summaryResult.value.data?.data) {
        setSummary(summaryResult.value.data.data)
      } else {
        setSummary(null)
        summaryFailure = true
      }

      if (trendsResult.status === 'fulfilled' && trendsResult.value.data?.common?.code === 0 && trendsResult.value.data?.data?.trends) {
        setTrendData(trendsResult.value.data.data.trends)
        setTrendDataError(null)
      } else {
        setTrendData([])
        setTrendDataError('Unable to load usage trends. Check your connection and try again.')
      }

      if (errorsResult.status === 'fulfilled' && errorsResult.value.data?.common?.code === 0 && errorsResult.value.data?.data?.toolErrors) {
        setErrorData(errorsResult.value.data.data.toolErrors)
        setErrorDataError(null)
      } else {
        setErrorData([])
        setErrorDataError('Unable to load tool error analysis. Check your connection and try again.')
      }

      let metricsFetchSucceeded = false
      let metricsTools: ToolUsageData[] = []
      if (metricsCountResult.status === 'fulfilled' && metricsCountResult.value.data?.common?.code === 0) {
        const metricsCountData = metricsCountResult.value.data?.data as ToolMetricsResponse | undefined
        const metricsTotalCount = metricsCountData?.totalCount ?? 0
        if (metricsTotalCount === 0) {
          metricsFetchSucceeded = true
        } else {
          try {
            const metricsRes = await api.usage.getToolMetrics({ timeRange, page: 1, pageSize: metricsTotalCount })
            if (metricsRes.data?.common?.code === 0 && metricsRes.data?.data?.tools) {
              metricsTools = metricsRes.data.data.tools as ToolUsageData[]
              metricsFetchSucceeded = true
            }
          } catch {
            metricsFetchSucceeded = false
          }
        }
      }

      if (metricsFetchSucceeded) {
        setToolUsageData(metricsTools)
        const tools = metricsTools
          .filter((tool) => !!tool.toolId)
          .map((tool) => ({ toolId: tool.toolId as string, toolName: tool.toolName }))
        setActionToolOptions(tools)
        setToolDataError(null)
      } else {
        setToolUsageData([])
        setActionToolOptions([])
        setToolDataError('Unable to load tool usage data. Check your connection and try again.')
      }

      if (!summaryFailure) {
        setLoadError(null)
      } else {
        setLoadError('Unable to load tool usage data. Check your connection and try again.')
      }
      hasDataRef.current = true
    } catch (error) {
      setSummary(null)
      setToolUsageData([])
      setToolDataError('Unable to load tool usage data. Check your connection and try again.')
      setTrendData([])
      setTrendDataError('Unable to load usage trends. Check your connection and try again.')
      setErrorData([])
      setErrorDataError('Unable to load tool error analysis. Check your connection and try again.')
      setActionToolOptions([])
      setLoadError('Unable to load tool usage data. Check your connection and try again.')
    } finally {
      setLoading(false)
    }
  }, [timeRange])

  const fetchActionLogsData = useCallback(async () => {
    try {
      setActionLoading(true)

      const actionFilters: { timeRange: number; page: number; pageSize: number; toolIds?: string[]; actionTypes?: number[]; status?: number } = {
        timeRange,
        page: actionLogsPage,
        pageSize: 20
      }

      if (actionLogToolId !== 'all') {
        actionFilters.toolIds = [actionLogToolId]
      }
      if (actionLogStatus === 'success') {
        actionFilters.status = 1
      }
      if (actionLogStatus === 'failed') {
        actionFilters.status = 2
      }
      if (actionLogType !== 'all') {
        actionFilters.actionTypes = [Number(actionLogType)]
      }

      const actionLogsRes = await api.usage.getToolOperationLogs(actionFilters)

      if (actionLogsRes.data?.common?.code === 0 && actionLogsRes.data?.data) {
        const actionData = actionLogsRes.data.data as ActionLogsResponse
        const safeTotalPages = Math.max(Math.ceil((actionData.totalCount || 0) / 20), 1)
        const clampedActionPage = Math.min(Math.max(actionLogsPage, 1), safeTotalPages)
        if (clampedActionPage !== actionLogsPage) {
          setActionLogsPage(clampedActionPage)
          return
        }
        setActionLogs(actionData.logs)
        setActionLogsTotal(actionData.totalCount)
        setActionLogsError(null)

        const typeSet = new Set<number>()
        for (const log of actionData.logs) {
          typeSet.add(log.actionType)
        }
        setActionTypeOptions(Array.from(typeSet).sort((a, b) => a - b))
        setActionTypeOptionsError(null)
      } else {
        setActionLogs([])
        setActionLogsTotal(0)
        setActionTypeOptions([])
        setActionLogsError('Unable to load tool action logs. Check your connection and try again.')
        setActionTypeOptionsError('Unable to load action filter options. Check your connection and try again.')
      }
    } catch {
      setActionLogs([])
      setActionLogsTotal(0)
      setActionTypeOptions([])
      setActionLogsError('Unable to load tool action logs. Check your connection and try again.')
      setActionTypeOptionsError('Unable to load action filter options. Check your connection and try again.')
    } finally {
      setActionLoading(false)
    }
  }, [actionLogsPage, actionLogStatus, actionLogToolId, actionLogType, timeRange])

  const handleRefresh = async () => {
    setRefreshing(true)
    await Promise.all([
      fetchToolUsageData(),
      activeTab === 'actions' ? fetchActionLogsData() : Promise.resolve()
    ])
    setRefreshing(false)
  }

  useEffect(() => {
    fetchToolUsageData()
  }, [fetchToolUsageData])

  useEffect(() => {
    if (activeTab !== 'actions') {
      return
    }
    fetchActionLogsData()
  }, [activeTab, fetchActionLogsData])

  const pieData = toolUsageData.map((tool) => ({
    name: tool.toolName,
    value: tool.totalRequests,
  }))

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "active":
        return <CheckCircle className="h-4 w-4 text-green-500 dark:text-green-400" />
      case "inactive":
        return <Clock className="h-4 w-4 text-gray-500 dark:text-gray-400" />
      case "error":
        return <XCircle className="h-4 w-4 text-red-500 dark:text-red-400" />
      default:
        return <Activity className="h-4 w-4" />
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "active":
        return <Badge className="bg-green-100 text-green-800 border-green-200 dark:bg-green-950/20 dark:text-green-300 dark:border-green-800">Active</Badge>
      case "inactive":
        return <Badge className="bg-gray-100 text-gray-800 border-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700">Inactive</Badge>
      case "error":
        return <Badge className="bg-red-100 text-red-800 border-red-200 dark:bg-red-950/20 dark:text-red-300 dark:border-red-800">High failure rate</Badge>
      default:
        return <Badge>Unknown</Badge>
    }
  }

  const chartAxisColor = 'hsl(var(--muted-foreground))'
  const chartGridColor = 'hsl(var(--border))'
  const chartTextColor = 'hsl(var(--foreground))'
  const hasActionFilters = actionLogStatus !== 'all' || actionLogToolId !== 'all' || actionLogType !== 'all'

  const clearActionFilters = () => {
    setActionLogToolId('all')
    setActionLogStatus('all')
    setActionLogType('all')
    setActionLogsPage(1)
  }

  return (
    <div className="space-y-6">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link href="/dashboard">Dashboard</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link href="/dashboard/usage">Usage Overview</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>Tool Usage</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <div className="space-y-0">
        <h1 className="text-[30px] font-bold">Tool Usage</h1>
        <p className="text-base text-muted-foreground">See which tools are busiest, failing, or slowing down.</p>
      </div>
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
        <Select value={String(timeRange)} onValueChange={(value) => { setTimeRange(Number(value)); setActionLogToolId('all'); setActionLogStatus('all'); setActionLogType('all'); setActionLogsPage(1) }}>
          <SelectTrigger className="w-full sm:w-[180px]" aria-label="Time range"><SelectValue placeholder="Time range" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="1">Last 24 hours</SelectItem>
            <SelectItem value="7">Last 7 days</SelectItem>
            <SelectItem value="30">Last 30 days</SelectItem>
          </SelectContent>
        </Select>
        <Button className="w-full sm:w-auto" variant="outline" onClick={handleRefresh} disabled={loading || refreshing}><RefreshCw className={`mr-2 h-4 w-4 ${loading || refreshing ? 'animate-spin' : ''}`} />{refreshing ? 'Refreshing...' : loading ? 'Loading overview...' : 'Refresh'}</Button>
      </div>

      {!loading && loadError ? (
        <div role="alert" className="flex flex-col items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300 sm:flex-row sm:items-center">
          <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
          <span>{loadError}</span>
          <Button variant="outline" size="sm" className="w-full sm:ml-auto sm:w-auto" onClick={handleRefresh}>Retry</Button>
        </div>
      ) : null}

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="w-full justify-start overflow-x-auto">
          <TabsTrigger className="shrink-0" value="overview">Overview</TabsTrigger>
          <TabsTrigger className="shrink-0" value="performance">Performance</TabsTrigger>
          <TabsTrigger className="shrink-0" value="errors">Error Analysis</TabsTrigger>
          <TabsTrigger className="shrink-0" value="trends">Usage Trends</TabsTrigger>
          <TabsTrigger className="shrink-0" value="actions">Action Logs</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          {/* Summary Cards */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <Card className="h-full">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Tools</CardTitle>
                <Wrench className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                <div className={loading || summary?.totalTools == null ? (loadError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
                  {loading
                    ? '—'
                    : summary?.totalTools == null
                    ? (loadError ? 'Unavailable' : '—')
                    : formatDisplayNumber(summary.totalTools, { compact: true })}
                </div>
                <p className="text-xs text-muted-foreground">
                  {loading
                    ? ''
                    : (summary?.activeTools == null && loadError)
                    ? 'Unavailable'
                    : summary?.activeTools == null
                    ? '—'
                    : `${formatDisplayNumber(summary.activeTools)} active`}
                </p>
              </CardContent>
            </Card>
            <Card className="h-full">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Requests</CardTitle>
                <Zap className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                <div className={loading || summary?.totalRequests == null ? (loadError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
                  {loading
                    ? '—'
                    : summary?.totalRequests == null
                    ? (loadError ? 'Unavailable' : '—')
                    : formatDisplayNumber(summary.totalRequests, { compact: true })}
                </div>
                <p className="text-xs text-muted-foreground">Last {timeRangeLabel}</p>
              </CardContent>
            </Card>
            <Card className="h-full">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
                <TrendingUp className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                 <div className={
                  loading || summary?.avgSuccessRate == null
                    ? (loadError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground")
                    : summary.avgSuccessRate >= HEALTHY_SUCCESS_RATE_THRESHOLD
                    ? "text-2xl font-bold text-green-600 dark:text-green-400"
                    : summary.avgSuccessRate < 80
                    ? "text-2xl font-bold text-red-600 dark:text-red-400"
                    : "text-2xl font-bold text-amber-600 dark:text-amber-400"
                }>
                  {loading
                    ? '—'
                    : summary?.avgSuccessRate == null
                    ? (loadError ? 'Unavailable' : '—')
                    : formatPercentage(summary.avgSuccessRate)}
                </div>
                <p className="text-xs text-muted-foreground">Average across all tools</p>
              </CardContent>
            </Card>
            <Card className="h-full">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Average Response Time</CardTitle>
                <Clock className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                <div className={loading || summary?.avgResponseTime == null ? (loadError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
                  {loading
                    ? '—'
                    : summary?.avgResponseTime == null
                    ? (loadError ? 'Unavailable' : '—')
                    : `${formatDisplayNumber(Math.round(summary.avgResponseTime))}ms`}
                </div>
                <p className="text-xs text-muted-foreground">Across all tools</p>
              </CardContent>
            </Card>
          </div>

          {/* Tool Usage Distribution */}
          <Card>
            <CardHeader>
              <CardTitle>Tool Usage Distribution</CardTitle>
            </CardHeader>
            <CardContent>
              {loading || !pieData || pieData.length === 0 ? (
                <div className="flex items-center justify-center h-[300px]">
                  {loading ? (
                    <div className="text-center">
                      <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                      <p className="text-sm text-muted-foreground">Loading tool distribution chart...</p>
                    </div>
                  ) : toolDataError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{toolDataError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No tool usage data in the selected range.</p>
                  )}
                </div>
              ) : (
                <ResponsiveContainer width="100%" height={300}>
                  <PieChart>
                    <Pie
                      data={pieData}
                      cx="50%"
                      cy="50%"
                      labelLine={false}
                      label={false}
                      outerRadius={80}
                      fill="#8884d8"
                      dataKey="value"
                    >
                      {pieData.map((entry, index) => (
                        <Cell key={entry.name} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip
                      contentStyle={{ backgroundColor: 'hsl(var(--background))', borderColor: chartGridColor, color: chartTextColor }}
                      labelStyle={{ color: chartTextColor }}
                      itemStyle={{ color: chartTextColor }}
                    />
                    <Legend wrapperStyle={{ color: chartTextColor }} />
                  </PieChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>

          {/* Detailed Tool List */}
          <Card>
            <CardHeader>
              <CardTitle>Tool Details</CardTitle>
            </CardHeader>
            <CardContent>
              {loading || !toolUsageData || toolUsageData.length === 0 ? (
                <div className="flex items-center justify-center py-8">
                  {loading ? (
                    <div className="text-center">
                      <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                      <p className="text-sm text-muted-foreground">Loading per-tool request details...</p>
                    </div>
                  ) : toolDataError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{toolDataError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No tool usage data in the selected range.</p>
                  )}
                </div>
              ) : (
                <div className="space-y-4">
                  {toolUsageData.map((tool) => (
                  <div key={tool.toolName} className="border rounded-lg p-4">
                    <div className="flex flex-wrap items-center justify-between gap-y-1 mb-3">
                      <div className="flex flex-wrap items-center gap-2">
                        {getStatusIcon(tool.status)}
                        <h3 className="font-semibold">{tool.toolName}</h3>
                        {getStatusBadge(tool.status)}
                      </div>
                      <div className="text-sm text-muted-foreground">Last used: {formatNullableText(tool.lastUsed)}</div>
                    </div>

                    <div className="grid grid-cols-2 md:grid-cols-5 gap-3 text-sm">
                      <div className="flex flex-col gap-1 justify-center">
                        <p className="text-muted-foreground">Total Requests</p>
                        <p className="font-semibold">{formatDisplayNumber(tool.totalRequests)}</p>
                      </div>
                      <div className="flex flex-col gap-1 justify-center">
                        <p className="text-muted-foreground">Success Rate</p>
                        <p className={`font-semibold ${tool.successRate >= HEALTHY_SUCCESS_RATE_THRESHOLD ? 'text-green-600 dark:text-green-400' : tool.successRate < 80 ? 'text-red-600 dark:text-red-400' : 'text-amber-600 dark:text-amber-400'}`}>{formatPercentage(tool.successRate)}</p>
                      </div>
                      <div className="flex flex-col gap-1 justify-center">
                        <p className="text-muted-foreground">Failed Requests</p>
                        <p className="font-semibold text-red-600 dark:text-red-400">{formatDisplayNumber(tool.failedRequests)}</p>
                      </div>
                      <div className="flex flex-col gap-1 justify-center">
                        <p className="text-muted-foreground">Average Response</p>
                        <p className="font-semibold">{formatDisplayNumber(Math.round(tool.averageResponseTime))}ms</p>
                      </div>
                      <div className="flex flex-col gap-1 justify-center">
                        <p className="text-muted-foreground">Usage Status</p>
                        <p className="font-semibold">{tool.status === 'active' ? 'Active' : tool.status === 'inactive' ? 'Inactive' : tool.status === 'error' ? 'High failure rate' : 'Unknown'}</p>
                      </div>
                    </div>

                    <div className="mt-3">
                      <div className="flex justify-between text-xs text-muted-foreground mb-1">
                        <span>Success Rate</span>
                        <span>{formatPercentage(tool.successRate)}</span>
                      </div>
                      <Progress value={tool.successRate} className="h-2" />
                    </div>
                  </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="performance" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Response Time Comparison</CardTitle>
              <CardDescription>Average response time for each tool</CardDescription>
            </CardHeader>
            <CardContent>
              {loading || toolUsageData.length === 0 ? (
                <div className="flex items-center justify-center h-[300px]">
                  {loading ? (
                    <div className="text-center">
                      <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                      <p className="text-sm text-muted-foreground">Loading tool response time chart...</p>
                    </div>
                  ) : toolDataError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{toolDataError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No response time data in the selected range.</p>
                  )}
                </div>
              ) : (
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={toolUsageData}>
                    <CartesianGrid stroke={chartGridColor} strokeDasharray="3 3" />
                    <XAxis dataKey="toolName" stroke={chartAxisColor} tick={{ fill: chartAxisColor }} tickFormatter={(value) => value.length > 12 ? value.slice(0, 12) + '…' : value} />
                    <YAxis stroke={chartAxisColor} tick={{ fill: chartAxisColor }} />
                    <Tooltip
                      contentStyle={{ backgroundColor: 'hsl(var(--background))', borderColor: chartGridColor, color: chartTextColor }}
                      labelStyle={{ color: chartTextColor }}
                      itemStyle={{ color: chartTextColor }}
                    />
                    <Bar dataKey="averageResponseTime" name="Average Response Time (ms)" fill="#8884d8" />
                  </BarChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Successful vs Failed Requests</CardTitle>
              <CardDescription>Request success and failure comparison</CardDescription>
            </CardHeader>
            <CardContent>
              {loading || toolUsageData.length === 0 ? (
                <div className="flex items-center justify-center h-[300px]">
                  {loading ? (
                    <div className="text-center">
                      <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                      <p className="text-sm text-muted-foreground">Loading request outcome chart...</p>
                    </div>
                  ) : toolDataError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{toolDataError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No request outcome data in the selected range.</p>
                  )}
                </div>
              ) : (
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={toolUsageData}>
                    <CartesianGrid stroke={chartGridColor} strokeDasharray="3 3" />
                    <XAxis dataKey="toolName" stroke={chartAxisColor} tick={{ fill: chartAxisColor }} tickFormatter={(value) => value.length > 12 ? value.slice(0, 12) + '…' : value} />
                    <YAxis stroke={chartAxisColor} tick={{ fill: chartAxisColor }} />
                    <Tooltip
                      contentStyle={{ backgroundColor: 'hsl(var(--background))', borderColor: chartGridColor, color: chartTextColor }}
                      labelStyle={{ color: chartTextColor }}
                      itemStyle={{ color: chartTextColor }}
                    />
                    <Legend wrapperStyle={{ color: chartTextColor }} />
                    <Bar dataKey="successfulRequests" fill="#00C49F" name="Successful" />
                    <Bar dataKey="failedRequests" fill="#FF8042" name="Failed" />
                  </BarChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="errors" className="space-y-4">
          {loading || errorData.length === 0 ? (
            <Card>
              <CardContent className="flex items-center justify-center py-8">
                {loading ? (
                  <div className="text-center">
                    <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                    <p className="text-sm text-muted-foreground">Loading tool error breakdown...</p>
                  </div>
                ) : errorDataError ? (
                  <div className="text-center">
                    <p className="text-sm text-red-600 dark:text-red-400">{errorDataError}</p>
                    <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No tool error analysis data in the selected range.</p>
                )}
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-3">
              {errorData.map((tool) => (
                <Card key={tool.toolId}>
                  <CardHeader>
                    <CardTitle>{tool.toolName} - Error Analysis</CardTitle>
                    <CardDescription>{formatDisplayNumber(tool.totalErrors)} total errors</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {tool.errorTypes.map((error) => (
                      <div key={`${tool.toolId}-${error.errorCode}`} className="flex flex-col items-start gap-1 sm:flex-row sm:items-center sm:justify-between">
                        <span className="text-sm">{error.errorMessage}</span>
                        <span className="text-xs text-red-600 dark:text-red-400">{formatDisplayNumber(error.count)} ({formatPercentage(error.percentage)})</span>
                      </div>
                    ))}
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>

        <TabsContent value="trends" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Usage Trends (Last {timeRangeLabel})</CardTitle>
              <CardDescription>Request trends for each tool</CardDescription>
            </CardHeader>
            <CardContent>
              {loading || trendData.length === 0 ? (
                <div className="flex items-center justify-center h-[400px]">
                  {loading ? (
                    <div className="text-center">
                      <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                      <p className="text-sm text-muted-foreground">Loading tool usage trend chart...</p>
                    </div>
                  ) : trendDataError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{trendDataError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No tool usage trends in the selected range.</p>
                  )}
                </div>
              ) : (
                <ResponsiveContainer width="100%" height={400}>
                  <LineChart data={trendData}>
                    <CartesianGrid stroke={chartGridColor} strokeDasharray="3 3" />
                    <XAxis dataKey="date" stroke={chartAxisColor} tick={{ fill: chartAxisColor }} />
                    <YAxis stroke={chartAxisColor} tick={{ fill: chartAxisColor }} />
                    <Tooltip
                      contentStyle={{ backgroundColor: 'hsl(var(--background))', borderColor: chartGridColor, color: chartTextColor }}
                      labelStyle={{ color: chartTextColor }}
                      itemStyle={{ color: chartTextColor }}
                    />
                    <Legend wrapperStyle={{ color: chartTextColor }} />
                    {(() => {
                      // Extract all tool names from trendData
                      const toolNames = new Set<string>();
                      trendData.forEach(trend => {
                        Object.keys(trend).forEach(key => {
                          if (key !== 'date') {
                            toolNames.add(key);
                          }
                        });
                      });
                    return Array.from(toolNames).map((toolName: string, index) => (
                        <Line
                          key={toolName}
                          type="monotone"
                          dataKey={toolName}
                          stroke={COLORS[index % COLORS.length]}
                          strokeWidth={2}
                        />
                      ));
                    })()}
                  </LineChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="actions" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Tool Action Logs</CardTitle>
              <CardDescription>Recent tool operations in the selected time range</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="mb-3 grid grid-cols-1 gap-2 md:grid-cols-3">
                <Select value={actionLogToolId} onValueChange={(value) => { setActionLogToolId(value); setActionLogType('all'); setActionLogsPage(1) }}>
                  <SelectTrigger aria-label="Filter by tool"><SelectValue placeholder="Filter by tool" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All tools</SelectItem>
                    {actionToolOptions.map((tool) => (
                      <SelectItem key={tool.toolId} value={tool.toolId}>{tool.toolName}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Select value={actionLogStatus} onValueChange={(value: 'all' | 'success' | 'failed') => { setActionLogStatus(value); setActionLogType('all'); setActionLogsPage(1) }}>
                  <SelectTrigger aria-label="Filter by status"><SelectValue placeholder="Filter by status" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All status</SelectItem>
                    <SelectItem value="success">Success</SelectItem>
                    <SelectItem value="failed">Failed</SelectItem>
                  </SelectContent>
                </Select>
                <Select value={actionLogType} onValueChange={(value) => { setActionLogType(value); setActionLogsPage(1) }}>
                  <SelectTrigger aria-label="Filter by action"><SelectValue placeholder="Filter by action" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All actions</SelectItem>
                    {actionTypeOptions.map((actionType) => (
                      <SelectItem key={String(actionType)} value={String(actionType)}>{getActionLabel(actionType)}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="mb-3">
                {hasActionFilters ? (
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-xs text-muted-foreground">Active filters:</span>
                    {actionLogToolId !== 'all' ? (
                      <Badge variant="outline" className="text-xs">
                        Tool: {actionToolOptions.find((tool) => tool.toolId === actionLogToolId)?.toolName || 'Selected'}
                      </Badge>
                    ) : null}
                    {actionLogStatus !== 'all' ? (
                      <Badge variant="outline" className="text-xs">
                        Status: {actionLogStatus.charAt(0).toUpperCase() + actionLogStatus.slice(1)}
                      </Badge>
                    ) : null}
                    {actionLogType !== 'all' ? (
                      <Badge variant="outline" className="text-xs">Action: {getActionLabel(Number(actionLogType))}</Badge>
                    ) : null}
                    <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={clearActionFilters}>Reset filters</Button>
                  </div>
                ) : (
                  <p className="text-xs text-muted-foreground">Showing all tools, all outcomes, and all actions.</p>
                )}
              </div>
              {actionTypeOptionsError ? (
                <p className="mb-3 text-xs text-amber-600 dark:text-amber-400">{actionTypeOptionsError}</p>
              ) : null}
              {actionLoading || actionLogs.length === 0 ? (
                <div className="flex items-center justify-center py-8">
                  {actionLoading ? (
                    <div className="text-center">
                      <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                      <p className="text-sm text-muted-foreground">Loading recent tool action logs...</p>
                    </div>
                  ) : actionLogsError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{actionLogsError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : hasActionFilters ? (
                    <div className="text-center">
                      <p className="text-sm text-muted-foreground">No action logs match the current filters.</p>
                      <Button variant="ghost" size="sm" className="mt-2" onClick={clearActionFilters}>Reset filters</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No tool action logs in the selected range.</p>
                  )}
                </div>
              ) : (
                <div className="overflow-x-auto">
                <Table className="min-w-[900px]">
                  <TableHeader>
                    <TableRow>
                      <TableHead scope="col">Time</TableHead>
                      <TableHead scope="col">Tool</TableHead>
                      <TableHead scope="col">Action</TableHead>
                      <TableHead scope="col">User</TableHead>
                      <TableHead scope="col">Status</TableHead>
                      <TableHead scope="col">Error / Details</TableHead>
                      <TableHead scope="col" className="text-right">Latency</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {actionLogs.map((log) => (
                      <TableRow key={log.logId}>
                        <TableCell>{formatDateTime(log.timestamp * 1000, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })}</TableCell>
                        <TableCell>{log.toolName}</TableCell>
                        <TableCell className="font-mono text-xs">{getActionLabel(log.actionType)}</TableCell>
                        <TableCell>{log.userName}</TableCell>
                         <TableCell>
                           <span className={log.status === 1 ? 'text-green-600 dark:text-green-400 text-sm' : 'text-red-600 dark:text-red-400 text-sm'}>
                             {log.status === 1 ? 'Success' : 'Failed'}
                           </span>
                         </TableCell>
                        <TableCell className="max-w-[260px]">
                          {log.status === 2 && log.errorMessage ? (
                            <div className="text-xs text-red-600 dark:text-red-400 whitespace-pre-wrap break-words">{log.errorMessage}</div>
                          ) : (
                            <details>
                              <summary className="cursor-pointer rounded-sm text-xs text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">View details</summary>
                              <pre className="mt-1 max-h-24 overflow-auto whitespace-pre-wrap break-words rounded bg-muted p-2 text-xs">{formatNullableText(log.details)}</pre>
                            </details>
                          )}
                        </TableCell>
                        <TableCell className="text-right">{formatDisplayNumber(log.responseTime)}ms</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                </div>
              )}
              {!actionLoading && !actionLogsError ? (
                <div className="mt-3 flex flex-col gap-3 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
                  <span>{formatDisplayNumber(actionLogsTotal)} total logs</span>
                  <div className="flex flex-wrap items-center gap-2">
                    <Button variant="outline" size="sm" onClick={() => setActionLogsPage((p) => Math.max(1, p - 1))} disabled={actionLogsPage <= 1}>Previous</Button>
                    <span>Page {formatDisplayNumber(actionLogsPage)} of {formatDisplayNumber(Math.max(1, Math.ceil(actionLogsTotal / 20)))}</span>
                    <Button variant="outline" size="sm" onClick={() => setActionLogsPage((p) => p + 1)} disabled={actionLogs.length < 20 || actionLogsPage * 20 >= actionLogsTotal}>Next</Button>
                  </div>
                </div>
              ) : null}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

export default function ToolUsagePage() {
  return (
    <Suspense
      fallback={(
        <div className="space-y-6">
          <Card>
            <CardContent className="py-10">
              <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground" role="status">
                <RefreshCw className="h-4 w-4 animate-spin" aria-hidden="true" />
                <span>Loading tool usage dashboard...</span>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    >
      <ToolUsagePageContent />
    </Suspense>
  )
}
