'use client'

import {
  Key,
  TrendingUp,
  Zap,
  Globe,
  RefreshCw,
  AlertTriangle
} from 'lucide-react'
import Link from 'next/link'
import { Suspense, useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { usePathname, useRouter, useSearchParams } from 'next/navigation'
import {
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
  AreaChart,
  Area
} from 'recharts'

import { Badge } from '@/components/ui/badge'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator
} from '@/components/ui/breadcrumb'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { api } from '@/lib/api-client'
import { formatDateTime, formatDisplayNumber, formatNullableText, formatPercentage } from '@/lib/utils'

interface TokenUsageData {
  tokenName: string
  tokenId: string
  totalRequests: number
  successfulRequests: number
  failedRequests: number
  rateLimit: number
  lastUsed: string
  status: 'active' | 'inactive' | 'expired' | 'limited'
  createdDate: string
  expiryDate: string | null
  clientCount: number
  topLocations: Array<{
    country: string
    city: string
    requests: number
  }>
  minuteUsage: Array<{
    minute: string
    requests: number
  }>
}

interface TokenTrendData {
  date: string
  [key: string]: string | number
}

interface GeoUsageItem {
  tokenId: string
  tokenName: string
  locations: Array<{
    country: string
    city: string
    requests: number
    percentage: number
  }>
}

interface TokenPatternPoint {
  timeLabel: string
  requests: number
}

interface TokenMetricsResponse {
  tokens: TokenUsageData[]
  totalCount: number
}

interface TokenUsageSummary {
  totalTokens: number
  activeTokens: number
  totalRequests: number
  avgSuccessRate: number
  totalClients: number
}

const COLORS = [
  '#0088FE',
  '#00C49F',
  '#FFBB28',
  '#FF8042',
  '#8884D8',
  '#82CA9D',
  '#FF6B6B',
  '#4ECDC4',
  '#A855F7',
  '#F97316'
]
const HEALTHY_SUCCESS_RATE_THRESHOLD = 95

function formatUsageTimestamp(value: string | null): string {
  if (!value?.trim()) return '—'

  const parsed = Date.parse(value)
  if (Number.isNaN(parsed)) return formatNullableText(value)

  return formatDateTime(parsed, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

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

function maskIdentifier(value: string | null | undefined): string {
  const normalized = value?.trim() || ''
  if (!normalized) return '—'
  if (normalized.length <= 8) return `${normalized.slice(0, 2)}***${normalized.slice(-2)}`
  return `${normalized.slice(0, 4)}****${normalized.slice(-4)}`
}

function getTokenSuccessRate(token: TokenUsageData): number | null {
  if (token.totalRequests === 0) return null
  const successRate = (token.successfulRequests / token.totalRequests) * 100
  return Number.isFinite(successRate) ? successRate : null
}

function TokenUsagePageContent() {
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
    return ['overview', 'geographic', 'patterns'].includes(requestedTab || '') ? requestedTab || 'overview' : 'overview'
  })
  const [refreshing, setRefreshing] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [tokenDataError, setTokenDataError] = useState<string | null>(null)
  const [trendDataError, setTrendDataError] = useState<string | null>(null)
  const [geoUsageError, setGeoUsageError] = useState<string | null>(null)
  const [summary, setSummary] = useState<TokenUsageSummary | null>(null)
  const [tokenUsageData, setTokenUsageData] = useState<TokenUsageData[]>([])
  const [trendData, setTrendData] = useState<TokenTrendData[]>([])
  const [geoUsage, setGeoUsage] = useState<GeoUsageItem[]>([])
  const [patternUsage, setPatternUsage] = useState<Record<string, TokenPatternPoint[]>>({})
  const [patternUsageErrors, setPatternUsageErrors] = useState<Record<string, string>>({})
  const [patternLoading, setPatternLoading] = useState(false)
  const timeRangeLabel = timeRange === 1 ? '24 hours' : `${timeRange} days`
  const hasDataRef = useRef(false)
  const displayTokenUsageData = useMemo(() => {
    const statusRank: Record<TokenUsageData['status'], number> = {
      active: 0,
      limited: 1,
      inactive: 2,
      expired: 3,
    }
    const getTimestamp = (value: string | null) => {
      const parsed = Date.parse(value || '')
      return Number.isNaN(parsed) ? 0 : parsed
    }

    return [...tokenUsageData].sort((a, b) => {
      const statusDelta = (statusRank[a.status] ?? 9) - (statusRank[b.status] ?? 9)
      if (statusDelta !== 0) {
        return statusDelta
      }
      const lastUsedDelta = getTimestamp(b.lastUsed) - getTimestamp(a.lastUsed)
      if (lastUsedDelta !== 0) {
        return lastUsedDelta
      }
      return b.totalRequests - a.totalRequests
    })
  }, [tokenUsageData])
  const patternTokens = displayTokenUsageData.filter((token) => token.status === 'active').slice(0, 5)

  useEffect(() => {
    const tabTitles: Record<string, string> = {
      overview: 'Access Token Usage',
      geographic: 'Access Token Client Locations',
      patterns: 'Access Token Usage Patterns'
    }
    const tabTitle = tabTitles[activeTab] || 'Access Token Usage'

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

  const fetchTokenUsageData = useCallback(async () => {
    try {
      if (!hasDataRef.current) setLoading(true)

      const [summaryResult, trendsResult, geoResult, metricsCountResult] = await Promise.allSettled([
        api.usage.getTokenUsageSummary({ timeRange }),
        api.usage.getTokenTrends({ timeRange, granularity: 2 }),
        api.usage.getTokenGeoDistribution({ timeRange }),
        api.usage.getTokenMetrics({ timeRange, page: 1, pageSize: 1 })
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
        setTrendDataError('Could not load token usage trends for this time range. Retry to refresh the Usage Patterns tab.')
      }

      if (geoResult.status === 'fulfilled' && geoResult.value.data?.common?.code === 0 && geoResult.value.data?.data?.geoUsage) {
        setGeoUsage(geoResult.value.data.data.geoUsage)
        setGeoUsageError(null)
      } else {
        setGeoUsage([])
        setGeoUsageError('Could not load client location data for this time range. Retry to refresh the Client Locations tab.')
      }

      let metricsFetchSucceeded = false
      let allTokens: TokenUsageData[] = []
      if (metricsCountResult.status === 'fulfilled' && metricsCountResult.value.data?.common?.code === 0) {
        const metricsCountData = metricsCountResult.value.data?.data as TokenMetricsResponse | undefined
        const metricsTotalCount = metricsCountData?.totalCount ?? 0
        if (metricsTotalCount === 0) {
          metricsFetchSucceeded = true
        } else {
          try {
            const metricsRes = await api.usage.getTokenMetrics({ timeRange, page: 1, pageSize: metricsTotalCount })
            if (metricsRes.data?.common?.code === 0 && metricsRes.data?.data?.tokens) {
              allTokens = metricsRes.data.data.tokens as TokenUsageData[]
              metricsFetchSucceeded = true
            }
          } catch {
            metricsFetchSucceeded = false
          }
        }
      }

      if (metricsFetchSucceeded) {
        setTokenUsageData(allTokens)
        setTokenDataError(null)
      } else {
        setTokenUsageData([])
        setTokenDataError('Could not load per-token usage data for this time range. Retry to refresh the Overview tab.')
      }

      setPatternUsage({})
      setPatternUsageErrors({})
      if (!summaryFailure) {
        setLoadError(null)
      } else {
        setLoadError('Could not load the token usage summary cards. Retry to refresh totals, success rate, and client counts.')
      }
      hasDataRef.current = true
    } catch (error) {
      setSummary(null)
      setTokenUsageData([])
      setTokenDataError('Could not load per-token usage data for this time range. Retry to refresh the Overview tab.')
      setTrendData([])
      setTrendDataError('Could not load token usage trends for this time range. Retry to refresh the Usage Patterns tab.')
      setGeoUsage([])
      setGeoUsageError('Could not load client location data for this time range. Retry to refresh the Client Locations tab.')
      setPatternUsage({})
      setPatternUsageErrors({})
      setLoadError(
        getRequestErrorMessage(error, {
          auth: 'Session expired or access revoked. Sign in again.',
          network: 'Could not load token usage. Check your connection and retry.',
          fallback: 'Could not load the token usage summary cards. Retry to refresh totals, success rate, and client counts.'
        })
      )
    } finally {
      setLoading(false)
    }
  }, [timeRange])

  useEffect(() => {
    let cancelled = false
    const loadPatternUsage = async () => {
      if (activeTab !== 'patterns' || timeRange !== 1) {
        if (!cancelled) {
          setPatternUsage({})
          setPatternUsageErrors({})
          setPatternLoading(false)
        }
        return
      }

      setPatternLoading(true)

      const activeTokens = displayTokenUsageData.filter((token) => token.status === 'active').slice(0, 5)
      if (activeTokens.length === 0) {
        if (!cancelled) {
          setPatternUsage({})
          setPatternUsageErrors({})
          setPatternLoading(false)
        }
        return
      }

      const patternResults = await Promise.all(
        activeTokens.map(async (token) => {
          try {
            const patternRes = await api.usage.getTokenUsagePatterns({ tokenId: token.tokenId, patternType: 1 })
            if (patternRes.data?.common?.code !== 0) {
              return {
                tokenId: token.tokenId,
                points: [] as TokenPatternPoint[],
        error: 'Could not load minute-level usage patterns for this token. Retry to refresh the Usage Patterns tab.'
              }
            }
            return {
              tokenId: token.tokenId,
              points: (patternRes.data?.data?.patterns || []) as TokenPatternPoint[],
              error: null as string | null
            }
          } catch {
            return {
              tokenId: token.tokenId,
              points: [] as TokenPatternPoint[],
        error: 'Could not load minute-level usage patterns for this token. Retry to refresh the Usage Patterns tab.'
            }
          }
        })
      )

      if (!cancelled) {
        const mappedPatterns: Record<string, TokenPatternPoint[]> = {}
        const mappedPatternErrors: Record<string, string> = {}
        patternResults.forEach((result) => {
          mappedPatterns[result.tokenId] = result.points
          if (result.error) {
            mappedPatternErrors[result.tokenId] = result.error
          }
        })
        setPatternUsage(mappedPatterns)
        setPatternUsageErrors(mappedPatternErrors)
        setPatternLoading(false)
      }
    }

    void loadPatternUsage()

    return () => {
      cancelled = true
    }
  }, [activeTab, timeRange, displayTokenUsageData])

  const handleRefresh = async () => {
    setRefreshing(true)
    await fetchTokenUsageData()
    setRefreshing(false)
  }

  useEffect(() => {
    fetchTokenUsageData()
  }, [fetchTokenUsageData])

  const pieData = displayTokenUsageData.map((token) => ({
    name: token.tokenName,
    value: token.totalRequests
  }))
  const patternPending =
    activeTab === 'patterns' &&
    patternTokens.some((token) => patternUsage[token.tokenId] === undefined)


  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'active':
        return (
          <Badge className="bg-green-100 text-green-800 border-green-200 dark:bg-green-950/20 dark:text-green-300 dark:border-green-800">
            Active
          </Badge>
        )
      case 'inactive':
        return (
          <Badge className="border-slate-200 bg-slate-100 text-slate-800 dark:border-slate-700 dark:bg-slate-900/60 dark:text-slate-200">
            Inactive
          </Badge>
        )
      case 'expired':
        return (
          <Badge className="bg-red-100 text-red-800 border-red-200 dark:bg-red-950/20 dark:text-red-300 dark:border-red-800">
            Expired
          </Badge>
        )
      case 'limited':
        return (
          <Badge className="bg-amber-100 text-amber-800 border-amber-200 dark:bg-amber-950/20 dark:text-amber-300 dark:border-amber-800">
            Rate Limited
          </Badge>
        )
      default:
        return <Badge>Unknown</Badge>
    }
  }

  const chartAxisColor = 'hsl(var(--muted-foreground))'
  const chartGridColor = 'hsl(var(--border))'
  const chartTextColor = 'hsl(var(--foreground))'

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
            <BreadcrumbPage>Access Token Usage</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <div className="space-y-0">
        <h1 className="text-[30px] font-bold tracking-tight">Access Token Usage</h1>
        <p className="text-sm leading-6 text-muted-foreground">See which tokens are active, where they are used, and when patterns change.</p>
      </div>
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
        <Select value={String(timeRange)} onValueChange={(value) => setTimeRange(Number(value))}>
          <SelectTrigger className="min-h-11 w-full sm:w-[180px]" aria-label="Time range"><SelectValue placeholder="Time range" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="1">Last 24 hours</SelectItem>
            <SelectItem value="7">Last 7 days</SelectItem>
            <SelectItem value="30">Last 30 days</SelectItem>
          </SelectContent>
        </Select>
        <Button className="min-h-11 w-full sm:w-auto" variant="outline" onClick={handleRefresh} disabled={loading || refreshing}><RefreshCw className={`mr-2 h-4 w-4 ${loading || refreshing ? 'animate-spin' : ''}`} />Refresh</Button>
      </div>
      {activeTab === 'patterns' && timeRange !== 1 ? (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between rounded-md border border-border/60 bg-muted/20 px-3 py-2">
          <p className="text-sm text-muted-foreground">Minute-level patterns are available only in the last 24 hours view.</p>
          <Button type="button" variant="outline" size="sm" className="min-h-11 w-full sm:w-auto" onClick={() => setTimeRange(1)}>
            Switch to last 24 hours
          </Button>
        </div>
      ) : null}
      {!loading && loadError ? (
        <div role="alert" className="flex flex-col items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300 sm:flex-row sm:items-center">
          <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
          <span>{loadError}</span>
          <Button variant="outline" size="sm" className="w-full sm:ml-auto sm:w-auto" onClick={handleRefresh}>Retry</Button>
        </div>
      ) : null}

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="h-11 w-full justify-start overflow-x-auto">
          <TabsTrigger className="min-h-11 shrink-0 px-4" value="overview">Overview</TabsTrigger>
          <TabsTrigger className="min-h-11 shrink-0 px-4" value="geographic">Client Locations</TabsTrigger>
          <TabsTrigger className="min-h-11 shrink-0 px-4" value="patterns">Usage Patterns</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          {/* Summary Cards */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <Card className="h-full">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Total Tokens
                </CardTitle>
                <Key className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                 <div
                   className={
                     loading ||
                     summary?.totalTokens == null
                       ? (loadError ? 'text-sm text-red-600 dark:text-red-400' : 'text-sm text-muted-foreground')
                       : 'text-2xl font-bold'
                   }
                 >
                  {loading
                    ? '—'
                    : summary?.totalTokens == null
                    ? (loadError ? 'Unavailable' : '—')
                    : formatDisplayNumber(summary.totalTokens, { compact: true })}
                </div>
                <p className="text-xs text-muted-foreground">
                  {loading
                    ? ''
                    : (summary?.activeTokens == null && loadError)
                    ? 'Unavailable'
                    : summary?.activeTokens == null
                    ? '—'
                    : `${formatDisplayNumber(summary.activeTokens)} active`}
                </p>
              </CardContent>
            </Card>
            <Card className="h-full">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Total Requests
                </CardTitle>
                <Zap className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                 <div
                   className={
                     loading ||
                     summary?.totalRequests == null
                       ? (loadError ? 'text-sm text-red-600 dark:text-red-400' : 'text-sm text-muted-foreground')
                       : 'text-2xl font-bold'
                   }
                 >
                  {loading
                    ? '—'
                    : summary?.totalRequests == null
                    ? (loadError ? 'Unavailable' : '—')
                    : formatDisplayNumber(summary.totalRequests, { compact: true })}
                </div>
                <p className="text-xs text-muted-foreground">
                  Last {timeRangeLabel}
                </p>
              </CardContent>
            </Card>
            <Card className="h-full">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Request Success Rate
                </CardTitle>
                <TrendingUp className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                {(() => {
                  if (loading) {
                    return <div className="text-sm text-muted-foreground">—</div>
                  }
                  let successRate: number | null = null
                  if (summary?.avgSuccessRate != null) {
                    successRate = summary.avgSuccessRate
                  }
                   return successRate === null ? (
                    <div className={loadError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground"}>{loadError ? 'Unavailable' : '—'}</div>
                  ) : (
                    <div className={`text-2xl font-bold ${
                      successRate >= HEALTHY_SUCCESS_RATE_THRESHOLD
                        ? 'text-green-600 dark:text-green-400'
                        : successRate < 80
                        ? 'text-red-600 dark:text-red-400'
                        : 'text-amber-600 dark:text-amber-400'
                    }`}>
                      {formatPercentage(successRate)}
                    </div>
                  )
                })()}
                <p className="text-xs text-muted-foreground">
                  Successful requests across all tokens in the last {timeRangeLabel}
                </p>
              </CardContent>
            </Card>
            <Card className="h-full">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Unique Clients
                </CardTitle>
                <Globe className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                 <div
                   className={
                     loading ||
                     summary?.totalClients == null
                       ? (loadError ? 'text-sm text-red-600 dark:text-red-400' : 'text-sm text-muted-foreground')
                       : 'text-2xl font-bold'
                   }
                 >
                  {loading
                    ? '—'
                    : summary?.totalClients == null
                    ? (loadError ? 'Unavailable' : '—')
                    : formatDisplayNumber(summary.totalClients, { compact: true })}
                </div>
                <p className="text-xs text-muted-foreground">
                  Distinct clients seen in the last {timeRangeLabel}
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Token Usage Distribution */}
          <Card>
            <CardHeader>
              <CardTitle>Requests by Token</CardTitle>
              <CardDescription>Share of total requests for the selected time range.</CardDescription>
            </CardHeader>
            <CardContent>
              {loading || !pieData || pieData.length === 0 ? (
                <div className="flex items-center justify-center h-[300px]">
                  {loading ? (
                    <div className="text-center">
                      <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                      <p className="text-sm text-muted-foreground">Loading token distribution chart...</p>
                    </div>
                  ) : tokenDataError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{tokenDataError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No token usage data is available for this period. Choose a wider time range or use a token and check back.</p>
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
                      stroke="hsl(var(--background))"
                      strokeWidth={1}
                    >
                      {pieData.map((entry, index) => (
                        <Cell
                          key={entry.name}
                          fill={COLORS[index % COLORS.length]}
                        />
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

          {/* Detailed Token List */}
          <Card>
            <CardHeader>
              <CardTitle>Tokens by Usage</CardTitle>
              <CardDescription>Sorted with active and most recently used tokens first.</CardDescription>
            </CardHeader>
            <CardContent>
              {loading || displayTokenUsageData.length === 0 ? (
                <div className="flex items-center justify-center py-8">
                  {loading ? (
                    <div className="text-center">
                      <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                      <p className="text-sm text-muted-foreground">Loading per-token request details...</p>
                    </div>
                  ) : tokenDataError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{tokenDataError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No token usage data is available for this period. Choose a wider time range or use a token and check back.</p>
                  )}
                </div>
              ) : (
                <>
                  <div className="space-y-3 md:hidden">
                    {displayTokenUsageData.map((token) => {
                      const successRate = getTokenSuccessRate(token)

                      return (
                        <Card key={token.tokenId} className="border border-border/60 shadow-sm">
                          <CardContent className="space-y-4 p-4">
                            <div className="flex items-start justify-between gap-3">
                              <div className="min-w-0 space-y-1">
                                <p className="text-sm font-semibold">{token.tokenName}</p>
                                <p className="break-all font-mono text-xs text-muted-foreground">
                                  {maskIdentifier(token.tokenId)}
                                </p>
                              </div>
                              <div className="shrink-0">{getStatusBadge(token.status)}</div>
                            </div>

                            <div className="grid grid-cols-2 gap-3 text-sm">
                              <div className="rounded-lg border border-border/60 bg-muted/20 px-3 py-2">
                                <p className="text-xs text-muted-foreground">Total Requests</p>
                                <p className="mt-1 font-semibold">{formatDisplayNumber(token.totalRequests)}</p>
                              </div>
                              <div className="rounded-lg border border-border/60 bg-muted/20 px-3 py-2">
                                <p className="text-xs text-muted-foreground">Unique Clients</p>
                                <p className="mt-1 font-semibold">{formatDisplayNumber(token.clientCount)}</p>
                              </div>
                              <div className="rounded-lg border border-border/60 bg-muted/20 px-3 py-2">
                                <p className="text-xs text-muted-foreground">Success Rate</p>
                                {successRate == null ? (
                                  <p className="mt-1 font-semibold text-muted-foreground">—</p>
                                ) : (
                                  <p className={`mt-1 font-semibold ${
                                    successRate >= HEALTHY_SUCCESS_RATE_THRESHOLD
                                      ? 'text-green-600 dark:text-green-400'
                                      : successRate < 80
                                      ? 'text-red-600 dark:text-red-400'
                                      : 'text-amber-600 dark:text-amber-400'
                                  }`}>
                                    {formatPercentage(successRate)}
                                  </p>
                                )}
                              </div>
                              <div className="rounded-lg border border-border/60 bg-muted/20 px-3 py-2">
                                <p className="text-xs text-muted-foreground">Last Used</p>
                                <p className="mt-1 text-muted-foreground">{formatUsageTimestamp(token.lastUsed)}</p>
                              </div>
                            </div>

                            <div className="space-y-2">
                              <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
                                <span>Success rate</span>
                                <span>{successRate == null ? '—' : formatPercentage(successRate)}</span>
                              </div>
                              <Progress
                                value={successRate ?? 0}
                                className="h-2"
                                aria-label={successRate == null
                                  ? `${token.tokenName} success rate unavailable`
                                  : `${token.tokenName} success rate ${formatPercentage(successRate)}`}
                              />
                            </div>
                          </CardContent>
                        </Card>
                      )
                    })}
                  </div>

                  <div className="hidden overflow-x-auto md:block">
                    <Table className="min-w-[920px]">
                      <TableHeader>
                        <TableRow>
                          <TableHead scope="col">Token Name</TableHead>
                          <TableHead scope="col">Status</TableHead>
                          <TableHead scope="col">Token ID</TableHead>
                          <TableHead scope="col">Total Requests</TableHead>
                          <TableHead scope="col">Success Rate</TableHead>
                          <TableHead scope="col">Unique Clients</TableHead>
                          <TableHead scope="col" className="text-right">Last Used</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {displayTokenUsageData.map((token) => {
                          const successRate = getTokenSuccessRate(token)

                          return (
                            <TableRow key={token.tokenId}>
                              <TableCell>
                                <span className="font-semibold">
                                  {token.tokenName}
                                </span>
                              </TableCell>
                              <TableCell>{getStatusBadge(token.status)}</TableCell>
                              <TableCell>
                                <p className="max-w-[200px] truncate font-mono text-xs" title={maskIdentifier(token.tokenId)}>
                                  {maskIdentifier(token.tokenId)}
                                </p>
                              </TableCell>
                              <TableCell>
                                <span className="font-semibold">
                                  {formatDisplayNumber(token.totalRequests)}
                                </span>
                              </TableCell>
                              <TableCell>
                                {successRate == null ? (
                                  <span className="font-semibold text-muted-foreground">—</span>
                                ) : (
                                  <span className={`font-semibold ${
                                    successRate >= HEALTHY_SUCCESS_RATE_THRESHOLD
                                      ? 'text-green-600 dark:text-green-400'
                                      : successRate < 80
                                      ? 'text-red-600 dark:text-red-400'
                                      : 'text-amber-600 dark:text-amber-400'
                                  }`}>
                                    {formatPercentage(successRate)}
                                  </span>
                                )}
                              </TableCell>
                              <TableCell>
                                <span className="font-semibold">
                                  {formatDisplayNumber(token.clientCount)}
                                </span>
                              </TableCell>
                              <TableCell className="text-right">
                                <span className="text-sm text-muted-foreground">
                                  {formatUsageTimestamp(token.lastUsed)}
                                </span>
                              </TableCell>
                            </TableRow>
                          )
                        })}
                      </TableBody>
                    </Table>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="geographic" className="space-y-4">
          {loading || geoUsage.length === 0 ? (
            <div className="flex items-center justify-center py-8">
              {loading ? (
                <div className="text-center">
                  <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                  <p className="text-sm text-muted-foreground">Loading client location data...</p>
                </div>
              ) : geoUsageError ? (
                <div className="text-center">
                  <p className="text-sm text-red-600 dark:text-red-400">{geoUsageError}</p>
                  <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No client location data is available for this period. Use a token from a client and check back.</p>
              )}
            </div>
          ) : (
            <>
              {geoUsage.map((token) => (
                <Card key={`${token.tokenId}-geo`}>
                  <CardHeader>
                    <CardTitle>
                        {token.tokenName} - Client Locations
                    </CardTitle>
                    <CardDescription>
                      Requests by client location for this token
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-3">
                      {token.locations.length === 0 ? (
                        <p className="text-sm text-muted-foreground text-center py-4">No client location data is available for this period. Use a token from a client and check back.</p>
                    ) : (
                      token.locations.map((location) => (
                        <div
                          key={`${location.country}-${location.city}`}
                          className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
                        >
                          <div className="flex min-w-0 items-center gap-2">
                            <Globe className="h-4 w-4 text-muted-foreground" />
                            <span className="text-sm">
                              {location.city && location.country
                                ? `${location.city}, ${location.country}`
                                : formatNullableText(location.city || location.country)}
                            </span>
                          </div>
                          <div className="flex w-full items-center justify-between gap-2 sm:w-auto">
                            <span className="text-sm font-semibold">
                              {formatDisplayNumber(location.requests)}
                            </span>
                            <div className="w-20">
                              <Progress
                                value={location.percentage}
                                className="h-2"
                                aria-label={`Share from ${location.city && location.country ? `${location.city}, ${location.country}` : formatNullableText(location.city || location.country)}: ${formatPercentage(location.percentage)}`}
                              />
                            </div>
                            <span className="text-xs text-muted-foreground w-12">
                              {formatPercentage(location.percentage)}
                            </span>
                          </div>
                        </div>
                      ))
                    )}
                    </div>
                  </CardContent>
                </Card>
              ))}
            </>
          )}
        </TabsContent>

        <TabsContent value="patterns" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Usage Trends (Last {timeRangeLabel})</CardTitle>
              <CardDescription>Request trends for each token</CardDescription>
            </CardHeader>
            <CardContent>
              {loading || !trendData || trendData.length === 0 ? (
                <div className="flex items-center justify-center h-[400px]">
                  {loading ? (
                    <div className="text-center">
                      <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                      <p className="text-sm text-muted-foreground">Loading token usage trend chart...</p>
                    </div>
                  ) : trendDataError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{trendDataError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No token usage trends are available for this period. Choose a wider time range or check back after more traffic arrives.</p>
                  )}
                </div>
              ) : (
                <ResponsiveContainer width="100%" height={400}>
                  <LineChart data={trendData}>
                    <CartesianGrid stroke={chartGridColor} strokeDasharray="3 3" />
                    <XAxis dataKey="date" stroke={chartAxisColor} tick={{ fill: chartAxisColor }} tickMargin={8} minTickGap={24} />
                    <YAxis stroke={chartAxisColor} tick={{ fill: chartAxisColor }} />
                    <Tooltip
                      contentStyle={{ backgroundColor: 'hsl(var(--background))', borderColor: chartGridColor, color: chartTextColor }}
                      labelStyle={{ color: chartTextColor }}
                      itemStyle={{ color: chartTextColor }}
                    />
                    <Legend wrapperStyle={{ color: chartTextColor }} />
                    {(() => {
                      const tokenNames = new Set<string>()
                      trendData.forEach((trend) => {
                        Object.keys(trend).forEach((key) => {
                          if (key !== 'date') {
                            tokenNames.add(key)
                          }
                        })
                      })
                      return Array.from(tokenNames).map((tokenName, index) => (
                        <Line
                          key={tokenName}
                          type="monotone"
                          dataKey={tokenName}
                          stroke={COLORS[index % COLORS.length]}
                          strokeWidth={2}
                        />
                      ))
                    })()}
                  </LineChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>

          {timeRange !== 1 ? (
            <Card>
              <CardContent className="flex items-center justify-center py-8">
                <div className="text-center">
                  <p className="text-sm text-muted-foreground">Switch to Last 24 hours to review minute-level token patterns.</p>
                </div>
              </CardContent>
            </Card>
          ) : loading || patternLoading || patternPending || patternTokens.length === 0 ? (
            <div className="flex items-center justify-center py-8">
              {loading || patternLoading || patternPending ? (
                <div className="text-center">
                  <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                  <p className="text-sm text-muted-foreground">Loading minute-by-minute token patterns...</p>
                </div>
              ) : tokenDataError ? (
                <div className="text-center">
                  <p className="text-sm text-red-600 dark:text-red-400">{tokenDataError}</p>
                  <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No tokens were used in the last 24 hours, so minute-level patterns are unavailable. Use a token and refresh this view.</p>
              )}
            </div>
          ) : (
            <>
              {patternTokens.map((token) => (
                  <Card key={`${token.tokenId}-pattern`}>
                    <CardHeader>
                      <CardTitle>
                        {token.tokenName} - Minute Usage Pattern
                      </CardTitle>
                      <CardDescription>
                        Last 60 minutes usage pattern for this token
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      {patternUsage[token.tokenId] && patternUsage[token.tokenId].length > 0 ? (
                        <ResponsiveContainer width="100%" height={200}>
                          <AreaChart data={patternUsage[token.tokenId]}>
                            <CartesianGrid stroke={chartGridColor} strokeDasharray="3 3" />
                            <XAxis dataKey="timeLabel" stroke={chartAxisColor} tick={{ fill: chartAxisColor }} tickMargin={8} minTickGap={20} />
                            <YAxis stroke={chartAxisColor} tick={{ fill: chartAxisColor }} />
                            <Tooltip
                              contentStyle={{ backgroundColor: 'hsl(var(--background))', borderColor: chartGridColor, color: chartTextColor }}
                              labelStyle={{ color: chartTextColor }}
                              itemStyle={{ color: chartTextColor }}
                            />
                            <Area
                              type="monotone"
                              dataKey="requests"
                              name="Requests"
                              stroke={COLORS[displayTokenUsageData.findIndex((t) => t.tokenId === token.tokenId) % COLORS.length]}
                              fill={COLORS[displayTokenUsageData.findIndex((t) => t.tokenId === token.tokenId) % COLORS.length]}
                              fillOpacity={0.3}
                            />
                          </AreaChart>
                        </ResponsiveContainer>
                      ) : (
                        <div className="flex items-center justify-center py-8">
                          {patternUsageErrors[token.tokenId] ? (
                            <div className="text-center" role="alert">
                              <p className="text-sm text-red-600 dark:text-red-400">{patternUsageErrors[token.tokenId]}</p>
                              <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                            </div>
                          ) : (
                            <p className="text-sm text-muted-foreground">No minute-level usage data is available yet. Run more requests with this token and refresh the chart.</p>
                          )}
                        </div>
                      )}
                    </CardContent>
                  </Card>
              ))}

            </>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}

export default function TokenUsagePage() {
  return (
    <Suspense
      fallback={(
        <div className="space-y-6">
          <Card>
            <CardContent className="py-10">
              <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground" role="status">
                <RefreshCw className="h-4 w-4 animate-spin" aria-hidden="true" />
                <span>Loading token usage dashboard...</span>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    >
      <TokenUsagePageContent />
    </Suspense>
  )
}
