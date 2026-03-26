'use client'

import {
  Key,
  TrendingUp,
  Zap,
  Globe,
  RefreshCw,
  AlertTriangle
} from 'lucide-react'
import { Suspense, useState, useEffect, useCallback, useRef } from 'react'
import { useSearchParams } from 'next/navigation'
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

function maskIdentifier(value: string | null | undefined): string {
  const normalized = value?.trim() || ''
  if (!normalized) return '-'
  if (normalized.length <= 8) return `${normalized.slice(0, 2)}***${normalized.slice(-2)}`
  return `${normalized.slice(0, 4)}****${normalized.slice(-4)}`
}

function TokenUsagePageContent() {
  const searchParams = useSearchParams()
  const [timeRange, setTimeRange] = useState(() => {
    const param = searchParams.get('timeRange')
    const num = param ? Number(param) : NaN
    return [1, 7, 30].includes(num) ? num : 1
  })
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('overview')
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
  const patternTokens = tokenUsageData.filter((token) => token.status === 'active').slice(0, 5)
  const hasDataRef = useRef(false)
  useEffect(() => {
    if (timeRange) {
      hasDataRef.current = false
    }
  }, [timeRange])

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
        setTrendDataError('Unable to load usage trends. Check your connection and try again.')
      }

      if (geoResult.status === 'fulfilled' && geoResult.value.data?.common?.code === 0 && geoResult.value.data?.data?.geoUsage) {
        setGeoUsage(geoResult.value.data.data.geoUsage)
        setGeoUsageError(null)
      } else {
        setGeoUsage([])
        setGeoUsageError('Unable to load client location data. Check your connection and try again.')
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
        setTokenDataError('Unable to load token usage data. Check your connection and try again.')
      }

      setPatternUsage({})
      setPatternUsageErrors({})
      if (!summaryFailure) {
        setLoadError(null)
      } else {
        setLoadError('Unable to load token usage data. Check your connection and try again.')
      }
      hasDataRef.current = true
    } catch (error) {
      setSummary(null)
      setTokenUsageData([])
      setTokenDataError('Unable to load token usage data. Check your connection and try again.')
      setTrendData([])
      setTrendDataError('Unable to load usage trends. Check your connection and try again.')
      setGeoUsage([])
      setGeoUsageError('Unable to load client location data. Check your connection and try again.')
      setPatternUsage({})
      setPatternUsageErrors({})
      setLoadError('Unable to load token usage data. Check your connection and try again.')
    } finally {
      setLoading(false)
    }
  }, [timeRange])

  useEffect(() => {
    let cancelled = false
    const loadPatternUsage = async () => {
      if (activeTab !== 'patterns') {
        return
      }

      setPatternLoading(true)

      const activeTokens = tokenUsageData.filter((token) => token.status === 'active').slice(0, 5)
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
        error: 'Unable to load usage patterns. Check your connection and try again.'
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
        error: 'Unable to load usage patterns. Check your connection and try again.'
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
  }, [activeTab, tokenUsageData])

  const handleRefresh = async () => {
    setRefreshing(true)
    await fetchTokenUsageData()
    setRefreshing(false)
  }

  useEffect(() => {
    fetchTokenUsageData()
  }, [fetchTokenUsageData])

  const pieData = tokenUsageData.map((token) => ({
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
          <Badge className="bg-gray-100 text-gray-800 border-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700">
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
          <Badge className="bg-yellow-100 text-yellow-800 border-yellow-200 dark:bg-yellow-950/20 dark:text-yellow-300 dark:border-yellow-800">
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
      <div className="space-y-0">
        <h1 className="text-[30px] font-bold">Access token usage</h1>
        <p className="text-base text-muted-foreground">See which tokens are active, where they are used, and when patterns change.</p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Select value={String(timeRange)} onValueChange={(value) => setTimeRange(Number(value))}>
          <SelectTrigger className="w-[180px]" aria-label="Time range"><SelectValue placeholder="Time range" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="1">Last 24 hours</SelectItem>
            <SelectItem value="7">Last 7 days</SelectItem>
            <SelectItem value="30">Last 30 days</SelectItem>
          </SelectContent>
        </Select>
        <Button variant="outline" onClick={handleRefresh} disabled={loading || refreshing}><RefreshCw className={`mr-2 h-4 w-4 ${loading || refreshing ? 'animate-spin' : ''}`} />Refresh data</Button>
      </div>
      <p className="text-xs text-muted-foreground">Minute-level patterns are available only in the 24-hour view.</p>
      {!loading && loadError ? (
        <div className="flex items-center gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300">
          <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
          <span>{loadError}</span>
          <Button variant="outline" size="sm" className="ml-auto" onClick={handleRefresh}>Retry</Button>
        </div>
      ) : null}

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="w-full justify-start overflow-x-auto">
          <TabsTrigger className="shrink-0" value="overview">Overview</TabsTrigger>
          <TabsTrigger className="shrink-0" value="geographic">Client Locations</TabsTrigger>
          <TabsTrigger className="shrink-0" value="patterns">Usage Patterns</TabsTrigger>
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
                      ? 'text-sm text-muted-foreground'
                      : 'text-2xl font-bold'
                  }
                >
                  {loading
                    ? 'Loading...'
                    : summary?.totalTokens == null
                    ? (loadError ? 'Unavailable' : '—')
                    : summary.totalTokens.toLocaleString()}
                </div>
                <p className="text-xs text-muted-foreground">
                  {loading
                    ? '-'
                    : (summary?.activeTokens == null && loadError)
                    ? 'Unavailable'
                    : summary?.activeTokens == null
                    ? '—'
                    : `${summary.activeTokens.toLocaleString()} active`}
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
                      ? 'text-sm text-muted-foreground'
                      : 'text-2xl font-bold'
                  }
                >
                  {loading
                    ? 'Loading...'
                    : summary?.totalRequests == null
                    ? (loadError ? 'Unavailable' : '—')
                    : summary.totalRequests.toLocaleString()}
                </div>
                <p className="text-xs text-muted-foreground">
                  Last {timeRangeLabel}
                </p>
              </CardContent>
            </Card>
            <Card className="h-full">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Success Rate
                </CardTitle>
                <TrendingUp className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                {(() => {
                  if (loading) {
                    return <div className="text-sm text-muted-foreground">Loading...</div>
                  }
                  let successRate: number | null = null
                  if (summary?.avgSuccessRate != null) {
                    successRate = summary.avgSuccessRate
                  }
                  return successRate === null ? (
                    <div className="text-sm text-muted-foreground">{loadError ? 'Unavailable' : '—'}</div>
                  ) : (
                    <div className={`text-2xl font-bold ${
                      successRate >= HEALTHY_SUCCESS_RATE_THRESHOLD
                        ? 'text-green-600 dark:text-green-400'
                        : successRate < 80
                        ? 'text-red-600 dark:text-red-400'
                        : 'text-amber-600 dark:text-amber-400'
                    }`}>
                      {successRate.toFixed(1)}%
                    </div>
                  )
                })()}
                <p className="text-xs text-muted-foreground">
                  Average across all tokens
                </p>
              </CardContent>
            </Card>
            <Card className="h-full">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Clients
                </CardTitle>
                <Globe className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                <div
                  className={
                    loading ||
                    summary?.totalClients == null
                      ? 'text-sm text-muted-foreground'
                      : 'text-2xl font-bold'
                  }
                >
                  {loading
                    ? 'Loading...'
                    : summary?.totalClients == null
                    ? (loadError ? 'Unavailable' : '—')
                    : summary.totalClients.toLocaleString()}
                </div>
                <p className="text-xs text-muted-foreground">
                  Clients seen in the last {timeRangeLabel}
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Token Usage Distribution */}
          <Card>
            <CardHeader>
              <CardTitle>Token Usage Distribution</CardTitle>
            </CardHeader>
            <CardContent>
              {loading || !pieData || pieData.length === 0 ? (
                <div className="flex items-center justify-center h-[300px]">
                  {loading ? (
                    <p className="text-sm text-muted-foreground">Loading token distribution...</p>
                  ) : tokenDataError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{tokenDataError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No token usage data in this period.</p>
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
              <CardTitle>Token Details</CardTitle>
            </CardHeader>
            <CardContent>
              {loading || !tokenUsageData || tokenUsageData.length === 0 ? (
                <div className="flex items-center justify-center py-8">
                  {loading ? (
                    <p className="text-sm text-muted-foreground">Loading...</p>
                  ) : tokenDataError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{tokenDataError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No token usage data in this period.</p>
                  )}
                </div>
              ) : (
                <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Token Name</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Token ID</TableHead>
                      <TableHead>Total Requests</TableHead>
                      <TableHead>Success Rate</TableHead>
                      <TableHead>Clients</TableHead>
                      <TableHead className="text-right">Last Used</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {tokenUsageData.map((token) => (
                      <TableRow key={token.tokenId}>
                        <TableCell>
                          <span className="font-semibold">
                            {token.tokenName}
                          </span>
                        </TableCell>
                        <TableCell>{getStatusBadge(token.status)}</TableCell>
                        <TableCell>
                          <p className="font-mono text-xs truncate max-w-[200px]" title={maskIdentifier(token.tokenId)}>
                            {maskIdentifier(token.tokenId)}
                          </p>
                        </TableCell>
                        <TableCell>
                          <span className="font-semibold">
                            {token.totalRequests.toLocaleString()}
                          </span>
                        </TableCell>
                        <TableCell>
                          {token.totalRequests === 0 ||
                          isNaN(
                            (token.successfulRequests / token.totalRequests) *
                              100
                          ) ||
                          !isFinite(
                            (token.successfulRequests / token.totalRequests) *
                              100
                          ) ? (
                            <span className="font-semibold text-muted-foreground">
                              -
                            </span>
                          ) : (
                            <span className={`font-semibold ${
                              ((token.successfulRequests / token.totalRequests) * 100) >= HEALTHY_SUCCESS_RATE_THRESHOLD
                                ? 'text-green-600 dark:text-green-400'
                                : ((token.successfulRequests / token.totalRequests) * 100) < 80
                                ? 'text-red-600 dark:text-red-400'
                                : 'text-amber-600 dark:text-amber-400'
                            }`}>
                              {((token.successfulRequests / token.totalRequests) * 100).toFixed(1)}%
                            </span>
                          )}
                        </TableCell>
                        <TableCell>
                          <span className="font-semibold">
                            {token.clientCount.toLocaleString()}
                          </span>
                        </TableCell>
                        <TableCell className="text-right">
                          <span className="text-sm text-muted-foreground">
                            {token.lastUsed}
                          </span>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="geographic" className="space-y-4">
          {loading || geoUsage.length === 0 ? (
            <div className="flex items-center justify-center py-8">
              {loading ? (
                <p className="text-sm text-muted-foreground">Loading...</p>
              ) : geoUsageError ? (
                <div className="text-center">
                  <p className="text-sm text-red-600 dark:text-red-400">{geoUsageError}</p>
                  <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No client location data in this period.</p>
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
                        <p className="text-sm text-muted-foreground text-center py-4">No client location data in this period.</p>
                    ) : (
                      token.locations.map((location) => (
                        <div
                          key={`${location.country}-${location.city}`}
                          className="flex items-center justify-between"
                        >
                          <div className="flex items-center gap-2">
                            <Globe className="h-4 w-4 text-muted-foreground" />
                            <span className="text-sm">
                              {location.city}, {location.country}
                            </span>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-semibold">
                              {location.requests.toLocaleString()}
                            </span>
                            <div className="w-20">
                              <Progress
                                value={
                                  location.percentage
                                }
                                className="h-2"
                              />
                            </div>
                            <span className="text-xs text-muted-foreground w-12">
                              {location.percentage.toFixed(1)}%
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
                    <p className="text-sm text-muted-foreground">Loading...</p>
                  ) : trendDataError ? (
                    <div className="text-center">
                      <p className="text-sm text-red-600 dark:text-red-400">{trendDataError}</p>
                      <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">No token usage trends in this period.</p>
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

          {loading || patternLoading || patternPending || patternTokens.length === 0 ? (
            <div className="flex items-center justify-center py-8">
              {loading || patternLoading || patternPending ? (
                <p className="text-sm text-muted-foreground">Loading...</p>
              ) : tokenDataError ? (
                <div className="text-center">
                  <p className="text-sm text-red-600 dark:text-red-400">{tokenDataError}</p>
                  <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No tokens used in the last 24 hours for minute-level patterns.</p>
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
                            <XAxis dataKey="timeLabel" stroke={chartAxisColor} tick={{ fill: chartAxisColor }} />
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
                              stroke={COLORS[tokenUsageData.findIndex((t) => t.tokenId === token.tokenId) % COLORS.length]}
                              fill={COLORS[tokenUsageData.findIndex((t) => t.tokenId === token.tokenId) % COLORS.length]}
                              fillOpacity={0.3}
                            />
                          </AreaChart>
                        </ResponsiveContainer>
                      ) : (
                        <div className="flex items-center justify-center py-8">
                          {patternUsageErrors[token.tokenId] ? (
                            <div className="text-center">
                              <p className="text-sm text-red-600 dark:text-red-400">{patternUsageErrors[token.tokenId]}</p>
                              <Button variant="outline" size="sm" className="mt-2" onClick={handleRefresh}>Retry</Button>
                            </div>
                          ) : (
                            <p className="text-sm text-muted-foreground">No minute-level usage data available.</p>
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
                <span>Loading token usage...</span>
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
