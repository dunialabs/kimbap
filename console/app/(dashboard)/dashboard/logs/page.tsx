'use client'

import {
  Download,
  Search,
  Eye,
  Clock,
  AlertTriangle,
  Info,
  XCircle,
  Activity,
  RefreshCw,
  Loader2,
  ChevronDown,
  ChevronUp
} from 'lucide-react'
import { Suspense, useState, useEffect, useCallback, useRef } from 'react'
import { usePathname, useRouter, useSearchParams } from 'next/navigation'
import { toast } from 'sonner'

import { api } from '@/lib/api-client'
import { useDebounce } from '@/hooks/use-debounce'
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
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  ScrollableDialogContent,
  DialogTrigger
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Textarea } from '@/components/ui/textarea'
import { getDomainLabel } from '@/lib/log-utils'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, ResponsiveContainer, Legend } from 'recharts'

interface LogEntry {
  id: string
  timestamp: string
  level: 'INFO' | 'WARN' | 'ERROR' | 'DEBUG'
  message: string
  source: string
  requestId?: string
  userId?: string
  rawData: string
  details: {
    method?: string
    url?: string
    statusCode?: number
    responseTime?: number
    userAgent?: string
    ip?: string
  }
}

// API response interface
interface LogsApiResponse {
  logs: LogEntry[]
  totalCount: number
  totalPages: number
  availableSources: string[]
}

interface RealtimeLogsApiResponse {
  newLogs: LogEntry[]
  latestLogId: number
  hasMore: boolean
}

const STATISTICS_SUPPORTED_TIME_RANGES = new Set(['1h', '6h', '24h', '7d'])
const DEFAULT_TIME_RANGE = '24h'

const getStatisticsTimeRange = (range: string) => (
  STATISTICS_SUPPORTED_TIME_RANGES.has(range) ? range : DEFAULT_TIME_RANGE
)

interface LogStatistics {
  totalLogs: number
  errorLogs: number
  warnLogs: number
  infoLogs: number
  debugLogs: number
  errorRate: number
  domainStats: { domain: string; label: string; logCount: number; errorCount: number; percentage: number }[]
  hourlyStats: { hour: string; totalCount: number; errorCount: number; timestamp: number }[]
}

function LogsPageContent() {
  const searchParams = useSearchParams()
  const pathname = usePathname()
  const router = useRouter()
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [totalCount, setTotalCount] = useState<number>(0)
  const [totalPages, setTotalPages] = useState<number>(0)
  const [availableSources, setAvailableSources] = useState<string[]>([])
  const [levelFilter, setLevelFilter] = useState<string>('all')
  const [sourceFilter, setSourceFilter] = useState<string>('all')
  const [timeFilter, setTimeFilter] = useState<string>(() => {
    const requested = searchParams.get('timeRange')
    const supportedRanges = new Set(['all', '1h', '6h', '24h', '7d', '30d'])
    return requested && supportedRanges.has(requested) ? requested : DEFAULT_TIME_RANGE
  })
  const [searchTerm, setSearchTerm] = useState('')
  const debouncedSearchTerm = useDebounce(searchTerm, 400)
  const [currentPage, setCurrentPage] = useState<number>(1)
  const [pageSize] = useState<number>(50)
  const [loading, setLoading] = useState<boolean>(false)
  const [exportLoading, setExportLoading] = useState<boolean>(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [statistics, setStatistics] = useState<LogStatistics | null>(null)
  const [statsLoading, setStatsLoading] = useState<boolean>(false)
  const [statsError, setStatsError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<string>(() => {
    const requestedTab = searchParams.get('tab')
    return ['table', 'raw', 'statistics'].includes(requestedTab || '') ? requestedTab || 'table' : 'table'
  })
  const [statisticsTimeFilter, setStatisticsTimeFilter] = useState<string>(() => getStatisticsTimeRange(timeFilter))

  useEffect(() => {
    const currentParam = searchParams.get('timeRange')
    const syncedTimeRange = activeTab === 'statistics' ? statisticsTimeFilter : timeFilter
  
    if (currentParam === syncedTimeRange) {
      return
    }
  
    const params = new URLSearchParams(searchParams.toString())
    params.set('timeRange', syncedTimeRange)
    router.replace(`${pathname}?${params.toString()}`, { scroll: false })
  }, [activeTab, pathname, router, searchParams, statisticsTimeFilter, timeFilter])

  useEffect(() => {
    const currentTab = searchParams.get('tab')
    if (currentTab === activeTab) {
      return
    }

    const params = new URLSearchParams(searchParams.toString())
    params.set('tab', activeTab)
    router.replace(`${pathname}?${params.toString()}`, { scroll: false })
  }, [activeTab, pathname, router, searchParams])
  const tableScopedFiltersEnabled = activeTab !== 'statistics'
  const [latestLogId, setLatestLogId] = useState<number>(0)
  const [realtimeHealthy, setRealtimeHealthy] = useState<boolean>(true)
  const [filtersOpen, setFiltersOpen] = useState<boolean>(true)
  const logsRequestSeqRef = useRef(0)
  const statsRequestSeqRef = useRef(0)

  // Function to load log data
  const loadLogs = useCallback(async (options?: { silent?: boolean }) => {
    const silent = options?.silent === true
    const requestSeq = ++logsRequestSeqRef.current
    if (!silent) {
      setLoading(true)
    }

    try {
      const response = await api.logs.getLogs({
        page: currentPage,
        pageSize,
        timeRange: timeFilter,
        level: levelFilter === 'all' ? undefined : levelFilter,
        source: sourceFilter === 'all' ? undefined : sourceFilter,
        search: debouncedSearchTerm || undefined
      })

      if (requestSeq !== logsRequestSeqRef.current) {
        return false
      }

      if (response.data?.common?.code === 0) {
        const data: LogsApiResponse = response.data.data
        const safeTotalPages = Math.max(data.totalPages || 0, 1)
        const clampedPage = Math.min(Math.max(currentPage, 1), safeTotalPages)
        if (clampedPage !== currentPage) {
          setCurrentPage(clampedPage)
          return true
        }
        setLogs(data.logs)
        setTotalCount(data.totalCount)
        setTotalPages(data.totalPages)
        setAvailableSources(data.availableSources)
        if (data.logs.length > 0) {
          const maxLogId = data.logs.reduce((max: number, log) => Math.max(max, Number(log.id) || 0), 0)
          setLatestLogId(maxLogId)
        } else {
          setLatestLogId(0)
        }
        setLoadError(null)
        setRealtimeHealthy(true)
        return true
      } else {
        // API returned non-zero code, toast shown below
        if (!silent) {
          toast.error(
            'Unable to load logs: ' +
              (response.data?.common?.message || 'Unknown error')
          )
          setLogs([])
          setTotalCount(0)
          setTotalPages(0)
          setAvailableSources([])
          setLatestLogId(0)
          setLoadError('Unable to load logs. Check your connection and try again.')
        }
        setRealtimeHealthy(false)
        return false
      }
    } catch (error) {
      if (requestSeq !== logsRequestSeqRef.current) {
        return false
      }
      // Network/fetch error, toast shown below
      if (!silent) {
        toast.error('Error loading logs')
        setLogs([])
        setTotalCount(0)
        setTotalPages(0)
        setAvailableSources([])
        setLatestLogId(0)
        setLoadError('Unable to load logs. Check your connection and try again.')
      }
      setRealtimeHealthy(false)
      return false
    } finally {
      if (!silent && requestSeq === logsRequestSeqRef.current) {
        setLoading(false)
      }
    }
  }, [currentPage, pageSize, timeFilter, levelFilter, sourceFilter, debouncedSearchTerm])

  const loadRealtimeLogs = useCallback(async () => {
    const realtimeSeq = logsRequestSeqRef.current
    if (currentPage !== 1 || activeTab !== 'table' || debouncedSearchTerm) {
      return
    }

    if (timeFilter !== 'all') {
      const reloaded = await loadLogs({ silent: true })
      if (realtimeSeq !== logsRequestSeqRef.current) {
        return
      }
      setRealtimeHealthy(reloaded)
      return
    }

    try {
      const response = await api.logs.getRealtimeLogs({
        lastLogId: latestLogId,
        level: levelFilter,
        source: sourceFilter,
        limit: pageSize
      })

      if (response.data?.common?.code !== 0 || !response.data?.data) {
        if (realtimeSeq !== logsRequestSeqRef.current) {
          return
        }
        setRealtimeHealthy(false)
        return
      }

      const data: RealtimeLogsApiResponse = response.data.data
      if (data.newLogs.length === 0) {
        if (realtimeSeq !== logsRequestSeqRef.current) {
          return
        }
        setRealtimeHealthy(true)
        return
      }

      if (data.hasMore) {
        const reloaded = await loadLogs({ silent: true })
        if (realtimeSeq !== logsRequestSeqRef.current) {
          return
        }
        setRealtimeHealthy(reloaded)
        return
      }

      if (realtimeSeq !== logsRequestSeqRef.current) {
        return
      }

      setLogs((prevLogs) => {
        const existingIds = new Set(prevLogs.map((log) => log.id))
        const appended = data.newLogs.filter((log) => !existingIds.has(log.id))
        if (appended.length === 0) {
          return prevLogs
        }
        const count = appended.length
        setTotalCount((prevCount) => {
          const nextCount = prevCount + count
          setTotalPages(Math.max(1, Math.ceil(nextCount / pageSize)))
          return nextCount
        })
        return [...appended.reverse(), ...prevLogs].slice(0, pageSize)
      })

      if (realtimeSeq !== logsRequestSeqRef.current) {
        return
      }
      setLatestLogId((prevLatestLogId) => Math.max(prevLatestLogId, data.latestLogId))
      setRealtimeHealthy(true)
    } catch {
      if (realtimeSeq !== logsRequestSeqRef.current) {
        return
      }
      setRealtimeHealthy(false)
    }
  }, [activeTab, currentPage, debouncedSearchTerm, latestLogId, levelFilter, loadLogs, pageSize, sourceFilter, timeFilter])

  // Reload on initial load and filter changes
  useEffect(() => {
    loadLogs()
  }, [loadLogs])

  const loadStatistics = useCallback(async () => {
    if (activeTab !== 'statistics') return true
    const statsSeq = ++statsRequestSeqRef.current
    setStatsLoading(true)

    try {
      const response = await api.logs.getStatistics({
        timeRange: statisticsTimeFilter,
      })

      if (statsSeq !== statsRequestSeqRef.current) return true

      if (response.data?.common?.code === 0) {
        setStatistics(response.data.data.statistics)
        setStatsError(null)
        return true
      } else {
        setStatistics(null)
        setStatsError('Unable to load statistics. Check your connection and try again.')
        return false
      }
    } catch (error) {
      if (statsSeq !== statsRequestSeqRef.current) return false
      setStatistics(null)
      setStatsError('Unable to load statistics. Check your connection and try again.')
      return false
    } finally {
      if (statsSeq === statsRequestSeqRef.current) {
        setStatsLoading(false)
      }
    }
  }, [activeTab, statisticsTimeFilter])

  useEffect(() => {
    loadStatistics()
  }, [loadStatistics])

  useEffect(() => {
    const interval = setInterval(() => {
      loadRealtimeLogs()
    }, 10000)

    return () => clearInterval(interval)
  }, [loadRealtimeLogs])

  // Manual refresh function
  const handleRefresh = async () => {
    const [logsOk, statsOk] = await Promise.all([
      loadLogs(),
      activeTab === 'statistics' ? loadStatistics() : Promise.resolve(true)
    ])
    if (logsOk && statsOk) {
      toast.success('Logs refreshed')
    }
  }

  const handleDownloadLogs = async () => {
    const exportLoadFailed = activeTab === 'statistics'
      ? !!statsError
      : !!loadError
    const noExportableLogs = activeTab === 'statistics'
      ? ((statistics?.totalLogs ?? 0) === 0)
      : logs.length === 0

    if (exportLoadFailed) {
      toast.error('Unable to load logs. Retry loading before exporting.')
      return
    }

    if (noExportableLogs) {
      toast.error('No logs available to download')
      return
    }

    try {
      setExportLoading(true)
      const scopedLevel = activeTab === 'statistics' ? 'all' : levelFilter
      const scopedSource = activeTab === 'statistics' ? 'all' : sourceFilter
      const scopedSearch = activeTab === 'statistics' ? '' : debouncedSearchTerm
      const scopedTimeRange = activeTab === 'statistics' ? statisticsTimeFilter : timeFilter
      const response = await api.logs.exportLogs({
        timeRange: scopedTimeRange,
        level: scopedLevel,
        source: scopedSource,
        search: scopedSearch,
        format: 1,
        maxRecords: 10000
      })

      if (response.data?.common?.code !== 0 || !response.data?.data) {
        toast.error(
                'Could not export logs: ' +
            (response.data?.common?.message || 'Unknown error')
        )
        return
      }

      const { downloadUrl, fileName, recordCount } = response.data.data
      const baseUrl = response.config?.baseURL || ''
      const resolvedDownloadUrl =
        downloadUrl.startsWith('http://') || downloadUrl.startsWith('https://')
          ? downloadUrl
          : `${baseUrl.replace(/\/$/, '')}${downloadUrl.startsWith('/') ? '' : '/'}${downloadUrl}`

      const a = document.createElement('a')
      a.href = resolvedDownloadUrl
      a.download = fileName
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)

      toast.success(`Exported ${recordCount} log records`)
    } catch {
      toast.error('Could not export logs')
    } finally {
      setExportLoading(false)
    }
  }

  const getLevelIcon = (level: string) => {
    switch (level) {
      case 'ERROR':
        return <XCircle className="h-4 w-4" aria-hidden="true" focusable="false" />
      case 'WARN':
        return <AlertTriangle className="h-4 w-4" aria-hidden="true" focusable="false" />
      case 'INFO':
        return <Info className="h-4 w-4" aria-hidden="true" focusable="false" />
      case 'DEBUG':
        return <Activity className="h-4 w-4" aria-hidden="true" focusable="false" />
      default:
        return <Info className="h-4 w-4" aria-hidden="true" focusable="false" />
    }
  }

  const getLevelColor = (level: string): string => {
    switch (level) {
      case 'ERROR':
        return 'destructive'
      case 'WARN':
        return 'warn'
      case 'INFO':
        return 'info'
      case 'DEBUG':
        return 'debug'
      default:
        return 'outline'
    }
  }

  // Clear filters function
  const clearFilters = () => {
    setLevelFilter('all')
    setSourceFilter('all')
    setTimeFilter(DEFAULT_TIME_RANGE)
    setStatisticsTimeFilter(getStatisticsTimeRange(DEFAULT_TIME_RANGE))
    setSearchTerm('')
    setCurrentPage(1)
  }

  const getTimeRangeLabel = (range: string) => {
    switch (range) {
      case '1h': return 'Last hour'
      case '6h': return 'Last 6 hours'
      case '24h': return 'Last 24 hours'
      case '7d': return 'Last 7 days'
      case '30d': return 'Last 30 days'
      case 'all': return 'All time'
      default: return range
    }
  }

  const chartAxisColor = 'hsl(var(--muted-foreground))'
  const chartGridColor = 'hsl(var(--border))'
  const chartTextColor = 'hsl(var(--foreground))'
  const exportLoadFailed = activeTab === 'statistics'
    ? !!statsError
    : !!loadError
  const noExportableLogs = activeTab === 'statistics'
    ? ((statistics?.totalLogs ?? 0) === 0)
    : logs.length === 0
  const isRealtimePaused = currentPage !== 1 || activeTab !== 'table' || !!debouncedSearchTerm
  const liveStatusText = loading ? 'Syncing' : isRealtimePaused ? 'Paused' : realtimeHealthy ? 'Live' : 'Refresh required'
  const selectedTimeRange = activeTab === 'statistics' ? statisticsTimeFilter : timeFilter
  const levelDisplayLabel: Record<string, string> = {
    ERROR: 'Error',
    WARN: 'Warning',
    INFO: 'Info',
    DEBUG: 'Debug',
  }

  const activeFilterBadges: string[] = []

  if (selectedTimeRange !== DEFAULT_TIME_RANGE) {
    activeFilterBadges.push(`Time: ${getTimeRangeLabel(selectedTimeRange)}`)
  }

  if (tableScopedFiltersEnabled) {
    if (debouncedSearchTerm.trim()) {
      activeFilterBadges.push(`Search: ${debouncedSearchTerm.trim()}`)
    }

    if (levelFilter !== 'all') {
      activeFilterBadges.push(`Level: ${levelDisplayLabel[levelFilter] || levelFilter}`)
    }

    if (sourceFilter !== 'all') {
      activeFilterBadges.push(`Source: ${getDomainLabel(sourceFilter)}`)
    }
  }

  const hasActiveFilters = activeFilterBadges.length > 0

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="space-y-0">
          <h1 className="text-[30px] font-bold">Logs & Monitoring</h1>
          <p className="text-base text-muted-foreground">
            Investigate requests, errors, and live activity.
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={loading}
          >
            <RefreshCw
              className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`}
              aria-hidden="true"
            />
            Refresh data
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleDownloadLogs}
            disabled={loading || exportLoading || exportLoadFailed || noExportableLogs}
            title={exportLoadFailed ? 'Retry loading logs before exporting' : noExportableLogs ? 'No logs available to download' : 'Download logs'}
            aria-label={exportLoadFailed ? 'Download logs disabled: retry loading logs before exporting' : noExportableLogs ? 'Download logs disabled: no logs available' : 'Download logs'}
          >
            <Download className="mr-2 h-4 w-4" />
            {exportLoading ? 'Exporting...' : 'Download Logs'}
          </Button>
        </div>
      </div>

      {/* Filters */}
      <Collapsible open={filtersOpen} onOpenChange={setFiltersOpen}>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between py-3">
            <div className="flex items-center gap-2 flex-wrap">
              <CardTitle>Filters</CardTitle>
              {!filtersOpen && (
                hasActiveFilters
                  ? activeFilterBadges.map((badge) => (
                      <Badge key={badge} variant="outline" className="text-xs max-w-[160px] truncate" title={badge}>
                        {badge}
                      </Badge>
                    ))
                  : <span className="text-xs text-muted-foreground">{getTimeRangeLabel(selectedTimeRange)}</span>
              )}
            </div>
            <CollapsibleTrigger asChild>
              <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                {filtersOpen ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                <span className="sr-only">{filtersOpen ? 'Collapse filters' : 'Expand filters'}</span>
              </Button>
            </CollapsibleTrigger>
          </CardHeader>
          <CollapsibleContent>
            <CardContent>
              <div className={`grid grid-cols-1 ${tableScopedFiltersEnabled ? 'md:grid-cols-5' : 'md:grid-cols-2'} gap-3`}>
                {tableScopedFiltersEnabled ? (
                <div className="space-y-2">
                  <Label htmlFor="search">Search</Label>
                  <div className="relative">
                    <Search
                      className="absolute left-3 top-3 h-4 w-4 text-muted-foreground pointer-events-none"
                      aria-hidden="true"
                      focusable="false"
                    />
                    <Input
                      id="search"
                      placeholder="Search request ID, user ID, error text, or user agent"
                      value={searchTerm}
                      onChange={(e) => { setSearchTerm(e.target.value); setCurrentPage(1) }}
                      className="pl-10"
                      autoCapitalize="none"
                      autoCorrect="off"
                      spellCheck={false}
                    />
                  </div>
                </div>
                ) : null}

                <div className="space-y-2">
                  <Label htmlFor="time-range-filter">Time Range</Label>
                  <Select
                    value={activeTab === 'statistics' ? statisticsTimeFilter : timeFilter}
                    onValueChange={(value) => {
                      if (activeTab === 'statistics') {
                        setStatisticsTimeFilter(getStatisticsTimeRange(value))
                      } else {
                        setTimeFilter(value)
                      }
                      setCurrentPage(1)
                    }}
                  >
                    <SelectTrigger id="time-range-filter">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {activeTab === 'statistics' ? null : <SelectItem value="all">All Time</SelectItem>}
                      <SelectItem value="1h">Last Hour</SelectItem>
                      <SelectItem value="6h">Last 6 Hours</SelectItem>
                      <SelectItem value="24h">Last 24 Hours</SelectItem>
                      <SelectItem value="7d">Last 7 Days</SelectItem>
                      {activeTab === 'statistics' ? null : <SelectItem value="30d">Last 30 Days</SelectItem>}
                    </SelectContent>
                  </Select>
                </div>

                {tableScopedFiltersEnabled ? (
                <div className="space-y-2">
                  <Label htmlFor="level-filter">Level</Label>
                  <Select value={levelFilter} onValueChange={(value) => { setLevelFilter(value); setCurrentPage(1) }}>
                    <SelectTrigger id="level-filter">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Levels</SelectItem>
                      <SelectItem value="ERROR">Error</SelectItem>
                      <SelectItem value="WARN">Warning</SelectItem>
                      <SelectItem value="INFO">Info</SelectItem>
                      <SelectItem value="DEBUG">Debug</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                ) : null}

                {tableScopedFiltersEnabled ? (
                <div className="space-y-2">
                  <Label htmlFor="source-filter">Source</Label>
                  <Select value={sourceFilter} onValueChange={(value) => { setSourceFilter(value); setCurrentPage(1) }}>
                    <SelectTrigger id="source-filter">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Sources</SelectItem>
                      {availableSources.map((source) => (
                        <SelectItem key={source} value={source}>
                          {getDomainLabel(source)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                ) : null}

                {tableScopedFiltersEnabled ? (
                <div className="flex items-end">
                  <Button variant="outline" onClick={clearFilters}>
                    Reset filters
                  </Button>
                </div>
                ) : null}
              </div>

              <div className="mt-3">
                {hasActiveFilters ? (
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-xs text-muted-foreground">Active filters:</span>
                    {activeFilterBadges.map((badge) => (
                      <Badge key={badge} variant="outline" className="text-xs">
                        {badge}
                      </Badge>
                    ))}
                    <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={clearFilters}>
                      Reset filters
                    </Button>
                  </div>
                ) : (
                  <p className="text-xs text-muted-foreground">
                    {tableScopedFiltersEnabled
                      ? 'Showing the last 24 hours with all levels and all sources.'
                      : 'Statistics view uses only the time range filter. Search, level, and source resume in Table View.'}
                  </p>
                )}
              </div>
            </CardContent>
          </CollapsibleContent>
        </Card>
      </Collapsible>

      {/* Logs Display */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="w-full justify-start overflow-x-auto">
          <TabsTrigger value="table">Table View</TabsTrigger>
          <TabsTrigger value="raw">Raw View</TabsTrigger>
          <TabsTrigger value="statistics">Statistics</TabsTrigger>
        </TabsList>

        <TabsContent value="table">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <span>
                  {loading ? 'Server Logs' : `Server Logs (${totalCount.toLocaleString()} total, showing ${logs.length.toLocaleString()})`}
                </span>
                <Badge variant="outline" className={currentPage !== 1 || activeTab !== 'table' || debouncedSearchTerm ? 'border-yellow-500 text-yellow-600 dark:text-yellow-400' : realtimeHealthy ? 'border-green-500 text-green-600 dark:text-green-400' : 'border-red-500 text-red-600 dark:text-red-400'}>
                  <Clock className="mr-1 h-3 w-3" />
                  {liveStatusText}
                </Badge>
              </CardTitle>
              <CardDescription>
                {!loading && totalPages > 1 && (
                  <span>
                    Page {currentPage} of {totalPages}
                  </span>
                )}
                <span className="block text-xs mt-1">
                  {isRealtimePaused
                    ? activeTab === 'statistics'
                      ? 'Live updates paused in Statistics view.'
                      : 'Live updates pause while searching or browsing older pages.'
                    : realtimeHealthy
                    ? 'Live updates run every 10 seconds.'
                    : 'Live updates stopped. Select Refresh data to load the latest logs.'}
                </span>
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="overflow-x-auto">
              <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[180px]">Timestamp</TableHead>
                      <TableHead className="w-[80px]">Level</TableHead>
                      <TableHead className="w-[200px]">Source</TableHead>
                      <TableHead className="w-[300px]">Message</TableHead>
                      <TableHead className="w-[100px]">Request ID</TableHead>
                      <TableHead className="w-[80px]">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {loading ? (
                      <TableRow>
                        <TableCell colSpan={6} className="text-center py-8">
                          <p className="text-sm text-muted-foreground">Loading filtered logs...</p>
                        </TableCell>
                      </TableRow>
                    ) : logs.map((log) => (
                      <Dialog key={log.id}>
                      <TableRow>
                        <TableCell className="font-mono text-xs">
                          {log.timestamp}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={getLevelColor(log.level) === 'destructive' ? 'destructive' : 'outline'}
                            className={
                              getLevelColor(log.level) === 'warn'
                                ? 'border-yellow-500 bg-yellow-50 text-yellow-700 dark:bg-yellow-950/20 dark:text-yellow-400 dark:border-yellow-700 text-xs'
                                : getLevelColor(log.level) === 'info'
                                ? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-950/20 dark:text-blue-400 dark:border-blue-700 text-xs'
                                : getLevelColor(log.level) === 'debug'
                                ? 'border-gray-400 bg-gray-50 text-gray-600 dark:bg-gray-800/20 dark:text-gray-400 dark:border-gray-600 text-xs'
                                : 'text-xs'
                            }
                          >
                            {getLevelIcon(log.level)}
                            <span className="ml-1">{log.level}</span>
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant="outline"
                            className="text-xs whitespace-nowrap"
                          >
                            {getDomainLabel(log.source)}
                          </Badge>
                        </TableCell>
                        <TableCell className="max-w-[300px]">
                          <DialogTrigger asChild>
                            <button
                              type="button"
                              className="truncate text-left w-full rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 hover:text-foreground/80"
                              aria-label="Open log details"
                              title={log.message}
                            >
                              {log.message}
                            </button>
                          </DialogTrigger>
                        </TableCell>
                        <TableCell className="font-mono text-xs">
                          {log.requestId ? (
                            <Badge variant="secondary" className="text-xs" title={log.requestId}>
                              {log.requestId.slice(-6)}
                            </Badge>
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </TableCell>
                        <TableCell>
                          <DialogTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              aria-label="View log details"
                            >
                              <Eye className="h-4 w-4" />
                            </Button>
                          </DialogTrigger>
                            <ScrollableDialogContent className="max-w-4xl">
                              <DialogHeader>
                                <DialogTitle className="flex items-center gap-2">
                                  {getLevelIcon(log.level)}
                                  Log Details - {log.timestamp}
                                </DialogTitle>
                                <DialogDescription>
                                  {getDomainLabel(log.source)} • {log.level} •{' '}
                                  {log.requestId || 'No Request ID'}
                                </DialogDescription>
                              </DialogHeader>

                              <div>
                              <div className="space-y-4">
                                <div>
                                  <Label className="text-sm font-medium">
                                    Message
                                  </Label>
                                  <p className="mt-1 p-3 bg-muted rounded text-sm">
                                    {log.message}
                                  </p>
                                </div>

                                {Object.keys(log.details).length > 0 && (
                                  <div>
                                    <Label className="text-sm font-medium">
                                      Request Details
                                    </Label>
                                    <div className="mt-1 p-3 bg-muted rounded text-sm space-y-1">
                                      {log.details.method && (
                                        <div>
                                          <strong>Method:</strong>{' '}
                                          {log.details.method}
                                        </div>
                                      )}
                                      {log.details.url && (
                                        <div>
                                          <strong>URL:</strong>{' '}
                                          {log.details.url}
                                        </div>
                                      )}
                                      {log.details.statusCode && (
                                        <div>
                                          <strong>Status:</strong>{' '}
                                          {log.details.statusCode}
                                        </div>
                                      )}
                                      {log.details.responseTime && (
                                        <div>
                                          <strong>Response Time:</strong>{' '}
                                          {log.details.responseTime}ms
                                        </div>
                                      )}
                                      {log.details.ip && (
                                        <div>
                                          <strong>IP:</strong> {log.details.ip}
                                        </div>
                                      )}
                                      {log.details.userAgent && (
                                        <div>
                                          <strong>User Agent:</strong>{' '}
                                          {log.details.userAgent}
                                        </div>
                                      )}
                                    </div>
                                  </div>
                                )}

                                <div>
                                  <Label
                                    htmlFor={`raw-log-data-${log.id}`}
                                    className="text-sm font-medium"
                                  >
                                    Raw Log Data
                                  </Label>
                                  <Textarea
                                    id={`raw-log-data-${log.id}`}
                                    value={log.rawData}
                                    readOnly
                                    className="mt-1 font-mono text-xs min-h-[200px]"
                                  />
                                </div>
                              </div>

                              </div>
                              <DialogFooter className="border-t pt-4">
                                <Button
                                  variant="outline"
                                  onClick={async () => {
                                    try {
                                      if (!navigator?.clipboard?.writeText) {
                                        toast.error('Clipboard not available')
                                        return
                                      }
                                      await navigator.clipboard.writeText(log.rawData)
                                      toast.success('Raw log data copied to clipboard')
                                    } catch {
                                      toast.error('Could not copy raw log data')
                                    }
                                  }}
                                >
                                  Copy Raw Data
                                </Button>
                              </DialogFooter>
                            </ScrollableDialogContent>
                        </TableCell>
                      </TableRow>
                      </Dialog>
                    ))}
                    {logs.length === 0 && !loading && (
                      <TableRow>
                        <TableCell colSpan={6} className="text-center py-8">
                          {loadError ? (
                            <div className="flex flex-col items-center gap-2">
                              <AlertTriangle className="h-5 w-5 text-red-500 dark:text-red-400" aria-hidden="true" />
                              <p className="text-sm text-red-600 dark:text-red-400">{loadError}</p>
                              <Button variant="outline" size="sm" onClick={handleRefresh}>
                                Retry
                              </Button>
                            </div>
                          ) : (
                            <div className="flex flex-col items-center gap-2">
                              <p className="text-sm text-muted-foreground">No logs found for the current filters.</p>
                              <p className="text-xs text-muted-foreground">Try broadening the time range or resetting filters.</p>
                              {hasActiveFilters ? (
                                <Button variant="ghost" size="sm" onClick={clearFilters}>
                                  Reset filters
                                </Button>
                              ) : timeFilter === DEFAULT_TIME_RANGE ? (
                                <Button variant="outline" size="sm" onClick={() => { setTimeFilter('7d'); setCurrentPage(1) }}>
                                  Show last 7 days
                                </Button>
                              ) : null}
                            </div>
                          )}
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>

              {/* Pagination controls */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-4">
                  <div className="text-sm text-muted-foreground">
                    Showing {(currentPage - 1) * pageSize + 1} to{' '}
                    {Math.min(currentPage * pageSize, totalCount)} of{' '}
                    {totalCount} logs
                  </div>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() =>
                        setCurrentPage((prev) => Math.max(1, prev - 1))
                      }
                      disabled={currentPage === 1 || loading}
                    >
                      Previous
                    </Button>
                    <span className="flex items-center px-3 py-1 text-sm">
                      Page {currentPage} of {totalPages}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() =>
                        setCurrentPage((prev) => Math.min(totalPages, prev + 1))
                      }
                      disabled={currentPage === totalPages || loading}
                    >
                      Next
                    </Button>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="raw">
            <Card>
            <CardHeader className="flex flex-row items-start justify-between gap-2">
              <div>
                <CardTitle>Raw Log View</CardTitle>
                <CardDescription>
                  {loading
                    ? 'Preparing raw logs for the current filtered page...'
                    : `Raw server logs for this filtered page (${logs.length.toLocaleString()} rows on page ${currentPage} of ${Math.max(totalPages, 1)})`}
                </CardDescription>
              </div>
              {!loading && logs.length > 0 && (
                <Button
                  variant="outline"
                  size="sm"
                  className="shrink-0"
                  onClick={async () => {
                    const rawText = logs.map((log) => log.rawData).join('\n\n')
                    try {
                      if (!navigator?.clipboard?.writeText) {
                        toast.error('Clipboard not available')
                        return
                      }
                      await navigator.clipboard.writeText(rawText)
                      toast.success('Raw logs copied to clipboard')
                    } catch {
                      toast.error('Could not copy raw logs')
                    }
                  }}
                >
                  Copy All
                </Button>
              )}
            </CardHeader>
            <CardContent>
              {loading ? (
                <div className="flex items-center justify-center py-16 text-muted-foreground">
                  <p className="text-sm">Preparing raw log output...</p>
                </div>
              ) : logs.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
                  <Activity className="h-12 w-12 mb-3 opacity-40" />
                  <p className="text-sm">{loadError || 'No logs available'}</p>
                  <p className="text-xs mt-1">{loadError ? 'Try refresh to load data again' : 'Try adjusting your filters or check back later'}</p>
                  {loadError ? (
                    <Button variant="outline" size="sm" className="mt-3" onClick={handleRefresh}>
                      Retry
                    </Button>
                  ) : hasActiveFilters ? (
                    <Button variant="ghost" size="sm" className="mt-3" onClick={clearFilters}>
                      Reset filters
                    </Button>
                  ) : null}
                </div>
              ) : (
                <Textarea
                  value={logs.map((log) => log.rawData).join('\n\n')}
                  readOnly
                  aria-label="Raw log output"
                  className="min-h-[600px] font-mono text-xs"
                  placeholder="No logs available..."
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="statistics" className="space-y-4">
          {statsError ? (
            <div className="flex items-center gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300">
              <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{statsError}</span>
              <Button variant="outline" size="sm" className="ml-auto" onClick={() => void loadStatistics()}>Retry</Button>
            </div>
          ) : null}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <Card className="h-full">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Total Logs</CardTitle>
                <CardDescription className="text-xs">{getTimeRangeLabel(statisticsTimeFilter)}</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                <div className={statsLoading || !statistics ? (statsError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
                  {statsLoading
                    ? 'Syncing total log count...'
                    : !statistics
                    ? (statsError ? 'Unavailable' : '—')
                    : statistics.totalLogs.toLocaleString()}
                </div>
              </CardContent>
            </Card>

            <Card className="h-full">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Errors</CardTitle>
                <CardDescription className="text-xs">{getTimeRangeLabel(statisticsTimeFilter)}</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                <div className={statsLoading || !statistics ? (statsError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
                  {statsLoading
                    ? 'Syncing error count...'
                    : !statistics
                    ? (statsError ? 'Unavailable' : '—')
                    : statistics.errorLogs.toLocaleString()}
                </div>
                <p className={`text-xs ${!statsLoading && statistics ? (statistics.errorRate > 0 ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400') : 'text-muted-foreground'}`}>
                  {!statsLoading && statistics ? `${statistics.errorRate.toFixed(1)}% error rate` : ''}
                </p>
              </CardContent>
            </Card>

            <Card className="h-full">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Warnings</CardTitle>
                <CardDescription className="text-xs">{getTimeRangeLabel(statisticsTimeFilter)}</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                <div className={statsLoading || !statistics ? (statsError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
                  {statsLoading
                    ? 'Syncing warning count...'
                    : !statistics
                    ? (statsError ? 'Unavailable' : '—')
                    : statistics.warnLogs.toLocaleString()}
                </div>
                <p className="text-xs text-muted-foreground">{!statsLoading && statistics && statistics.totalLogs > 0 ? `${(statistics.warnLogs / statistics.totalLogs * 100).toFixed(1)}% of total` : ''}</p>
              </CardContent>
            </Card>

            <Card className="h-full">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Info</CardTitle>
                <CardDescription className="text-xs">{getTimeRangeLabel(statisticsTimeFilter)}</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-1 justify-center">
                <div className={statsLoading || !statistics ? (statsError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
                  {statsLoading
                    ? 'Syncing info count...'
                    : !statistics
                    ? (statsError ? 'Unavailable' : '—')
                    : statistics.infoLogs.toLocaleString()}
                </div>
                <p className="text-xs text-muted-foreground">
                  {!statsLoading && statistics ? `+ ${statistics.debugLogs.toLocaleString()} debug logs` : ''}
                </p>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Log Activity</CardTitle>
              <CardDescription>Hourly log volume and error trend</CardDescription>
            </CardHeader>
            <CardContent>
              {statsLoading ? (
                <div className="flex items-center justify-center h-[300px]">
                  <div className="text-center">
                    <Loader2 className="h-8 w-8 animate-spin mx-auto mb-3 text-muted-foreground" />
                    <p className="text-sm text-muted-foreground">Loading hourly log activity...</p>
                  </div>
                </div>
              ) : statsError ? (
                <div className="flex flex-col items-center justify-center h-[300px] text-muted-foreground">
                  <Activity className="h-12 w-12 mb-3 opacity-40" />
                  <p className="text-sm">{statsError}</p>
                </div>
              ) : !statistics?.hourlyStats || statistics.hourlyStats.length === 0 ? (
                <div className="flex flex-col items-center justify-center h-[300px] text-muted-foreground">
                  <Activity className="h-12 w-12 mb-3 opacity-40" />
                  <p className="text-sm">No log activity data in this period.</p>
                </div>
              ) : (
                <ResponsiveContainer width="100%" height={300}>
                  <AreaChart data={statistics.hourlyStats}>
                    <CartesianGrid stroke={chartGridColor} strokeDasharray="3 3" />
                    <XAxis dataKey="hour" stroke={chartAxisColor} tick={{ fill: chartAxisColor }} />
                    <YAxis stroke={chartAxisColor} tick={{ fill: chartAxisColor }} />
                    <RechartsTooltip
                      contentStyle={{ backgroundColor: 'hsl(var(--background))', borderColor: chartGridColor, color: chartTextColor }}
                      labelStyle={{ color: chartTextColor }}
                      itemStyle={{ color: chartTextColor }}
                    />
                    <Legend wrapperStyle={{ color: chartTextColor }} />
                    <Area type="monotone" dataKey="totalCount" name="Total" stroke="#3b82f6" fill="#3b82f6" fillOpacity={0.1} />
                    <Area type="monotone" dataKey="errorCount" name="Errors" stroke="#ef4444" fill="#ef4444" fillOpacity={0.1} />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Log Sources</CardTitle>
              <CardDescription>Distribution by source domain</CardDescription>
            </CardHeader>
            <CardContent>
              {statsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <p className="text-sm text-muted-foreground">Loading log source breakdown...</p>
                </div>
              ) : statsError ? (
                <div className="flex items-center justify-center py-8">
                  <p className="text-sm text-muted-foreground">{statsError}</p>
                </div>
              ) : !statistics?.domainStats || statistics.domainStats.length === 0 ? (
                <div className="flex items-center justify-center py-8">
                  <p className="text-sm text-muted-foreground">No log sources in this period.</p>
                </div>
              ) : (
                <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Source</TableHead>
                      <TableHead>Logs</TableHead>
                      <TableHead>Errors</TableHead>
                      <TableHead>Share</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {statistics.domainStats.map((stat) => (
                      <TableRow key={stat.domain}>
                        <TableCell>{stat.label}</TableCell>
                        <TableCell>{stat.logCount.toLocaleString()}</TableCell>
                        <TableCell className={stat.errorCount > 0 ? 'text-red-600 dark:text-red-400' : ''}>
                          {stat.errorCount.toLocaleString()}
                        </TableCell>
                        <TableCell>{stat.percentage.toFixed(1)}%</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

export default function LogsPage() {
  return (
    <Suspense
      fallback={(
        <div className="space-y-6">
          <Card>
            <CardContent className="py-10">
              <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground" role="status">
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                <span>Loading logs workspace...</span>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    >
      <LogsPageContent />
    </Suspense>
  )
}
