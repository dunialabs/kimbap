'use client'

import {
  Download,
  Search,
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
import { Pagination, PaginationContent, PaginationItem } from '@/components/ui/pagination'
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

import { formatDateTime, formatDisplayNumber, formatNullableText, formatPercentage, formatResponseTime } from '@/lib/utils'

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
const QUICK_TABLE_TIME_RANGES = ['1h', '6h', '24h', '7d', '30d'] as const
const QUICK_STATISTICS_TIME_RANGES = ['1h', '6h', '24h', '7d'] as const

const getStatisticsTimeRange = (range: string) => (
  STATISTICS_SUPPORTED_TIME_RANGES.has(range) ? range : DEFAULT_TIME_RANGE
)

const formatLogTimestamp = (timestamp: string) => formatDateTime(timestamp, {
  year: 'numeric',
  month: 'short',
  day: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit'
})

const LOG_LEVEL_DISPLAY: Record<string, string> = {
  ERROR: 'Error',
  WARN: 'Warning',
  INFO: 'Info',
  DEBUG: 'Debug',
}

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
    const tabTitles: Record<string, string> = {
      table: 'Logs & Monitoring',
      raw: 'Raw Logs',
      statistics: 'Log Statistics'
    }
    const tabTitle = tabTitles[activeTab] || 'Logs & Monitoring'

    document.title = `${tabTitle} | Kimbap Console`
  }, [activeTab])

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
  const [filtersOpen, setFiltersOpen] = useState<boolean>(false)
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
        if (!silent) {
          setLogs([])
          setTotalCount(0)
          setTotalPages(0)
          setAvailableSources([])
          setLatestLogId(0)
          setLoadError('Could not load log entries for the current filters. Retry to refresh the table and raw view.')
        }
        setRealtimeHealthy(false)
        return false
      }
    } catch (error) {
      if (requestSeq !== logsRequestSeqRef.current) {
        return false
      }
      if (!silent) {
        setLogs([])
        setTotalCount(0)
        setTotalPages(0)
        setAvailableSources([])
        setLatestLogId(0)
        setLoadError(
          getRequestErrorMessage(error, {
            auth: 'Could not load log entries because your session expired or your access changed. Sign in again and retry.',
            network: 'Could not load log entries. Check your connection and retry.',
            fallback: 'Could not load log entries for the current filters. Retry to refresh the table and raw view.'
          })
        )
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
        level: levelFilter === 'all' ? undefined : levelFilter,
        source: sourceFilter === 'all' ? undefined : sourceFilter,
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
        setStatsError('Could not load log statistics for this time range. Retry to refresh the charts and source table.')
        return false
      }
    } catch (error) {
      if (statsSeq !== statsRequestSeqRef.current) return false
      setStatistics(null)
      setStatsError(
        getRequestErrorMessage(error, {
          auth: 'Could not load log statistics because your session expired or your access changed. Sign in again and retry.',
          network: 'Could not load log statistics. Check your connection and retry.',
          fallback: 'Could not load log statistics for this time range. Retry to refresh the charts and source table.'
        })
      )
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
      toast.success('Logs refreshed.')
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
      toast.error('No logs are available to download.')
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

      toast.success(`Exported ${recordCount} log records.`)
    } catch {
      toast.error('Could not export logs. Try again.')
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

  const copyTextToClipboard = async (
    value: string,
    successMessage: string,
    errorMessage: string
  ) => {
    try {
      if (!navigator?.clipboard?.writeText) {
        toast.error('Clipboard is unavailable in this browser.')
        return
      }

      await navigator.clipboard.writeText(value)
      toast.success(successMessage)
    } catch {
      toast.error(errorMessage)
    }
  }

  const getLevelBadgeClass = (level: string) => {
    switch (getLevelColor(level)) {
      case 'warn':
        return 'border-amber-500 bg-amber-50 text-amber-700 dark:bg-amber-950/20 dark:text-amber-300 dark:border-amber-700 text-xs'
      case 'info':
        return 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-950/20 dark:text-blue-400 dark:border-blue-700 text-xs'
      case 'debug':
        return 'border-slate-300 bg-slate-50 text-slate-700 dark:border-slate-700 dark:bg-slate-900/60 dark:text-slate-200 text-xs'
      default:
        return 'text-xs'
    }
  }

  const renderLogDetailsDialogContent = (log: LogEntry) => (
    <ScrollableDialogContent className="max-w-4xl">
      <DialogHeader>
        <DialogTitle className="flex items-center gap-2">
          {getLevelIcon(log.level)}
          Log details
        </DialogTitle>
        <DialogDescription>
          {formatLogTimestamp(log.timestamp)} • {getDomainLabel(log.source)} • {LOG_LEVEL_DISPLAY[log.level] ?? log.level}
          {log.requestId ? ` • ${log.requestId}` : ''}
        </DialogDescription>
      </DialogHeader>

      <div className="space-y-4">
        <div>
          <Label className="text-sm font-medium">Message</Label>
          <p className="mt-1 rounded bg-muted p-3 text-sm">
            {log.message}
          </p>
        </div>

        {log.userId && (
          <div>
            <Label className="text-sm font-medium">User</Label>
            <p className="mt-1 font-mono text-sm text-muted-foreground">{log.userId}</p>
          </div>
        )}

        {Object.keys(log.details).length > 0 && (
          <div>
            <Label className="text-sm font-medium">
              Request Details
            </Label>
            <div className="mt-1 space-y-1 rounded bg-muted p-3 text-sm">
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
              {log.details.statusCode != null && (
                <div>
                  <strong>Status:</strong>{' '}
                  <span className={log.details.statusCode >= 500 ? 'text-red-600 dark:text-red-400 font-medium' : log.details.statusCode >= 400 ? 'text-amber-600 dark:text-amber-400 font-medium' : log.details.statusCode >= 200 && log.details.statusCode < 300 ? 'text-green-600 dark:text-green-400' : ''}>
                    {formatDisplayNumber(log.details.statusCode)}
                  </span>
                </div>
              )}
              {log.details.responseTime != null && (
                <div>
                  <strong>Response Time:</strong>{' '}
                  {formatResponseTime(log.details.responseTime)}
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
            className="mt-1 min-h-[200px] font-mono text-xs"
          />
        </div>
      </div>

      <DialogFooter className="border-t pt-4">
        <Button
          variant="outline"
          className="min-h-11"
          onClick={() => void copyTextToClipboard(log.rawData, 'Raw log data copied to clipboard', 'Could not copy raw log data')}
        >
          Copy Raw Data
        </Button>
      </DialogFooter>
    </ScrollableDialogContent>
  )

  // Clear filters function
  const clearFilters = () => {
    setLevelFilter('all')
    setSourceFilter('all')
    setTimeFilter(DEFAULT_TIME_RANGE)
    setStatisticsTimeFilter(getStatisticsTimeRange(DEFAULT_TIME_RANGE))
    setSearchTerm('')
    setCurrentPage(1)
  }

  const applyTimeRange = (value: string) => {
    if (activeTab === 'statistics') {
      setStatisticsTimeFilter(getStatisticsTimeRange(value))
    } else {
      setTimeFilter(value)
    }
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
  const chartPrimaryColor = 'hsl(var(--primary))'
  const chartDestructiveColor = 'hsl(var(--destructive))'
  const exportLoadFailed = activeTab === 'statistics'
    ? !!statsError
    : !!loadError
  const noExportableLogs = activeTab === 'statistics'
    ? ((statistics?.totalLogs ?? 0) === 0)
    : logs.length === 0
  const isRealtimePaused = currentPage !== 1 || activeTab !== 'table' || !!debouncedSearchTerm
  const liveStatusText = loading ? 'Syncing' : isRealtimePaused ? 'Paused' : realtimeHealthy ? 'Live' : 'Refresh required'
  const selectedTimeRange = activeTab === 'statistics' ? statisticsTimeFilter : timeFilter
  const quickTimeRanges = activeTab === 'statistics' ? QUICK_STATISTICS_TIME_RANGES : QUICK_TABLE_TIME_RANGES
  const activeFilterBadges: string[] = []

  if (selectedTimeRange !== DEFAULT_TIME_RANGE) {
    activeFilterBadges.push(`Time: ${getTimeRangeLabel(selectedTimeRange)}`)
  }

  if (tableScopedFiltersEnabled) {
    if (debouncedSearchTerm.trim()) {
      activeFilterBadges.push(`Search: ${debouncedSearchTerm.trim()}`)
    }

    if (levelFilter !== 'all') {
      activeFilterBadges.push(`Level: ${LOG_LEVEL_DISPLAY[levelFilter] || levelFilter}`)
    }

    if (sourceFilter !== 'all') {
      activeFilterBadges.push(`Source: ${getDomainLabel(sourceFilter)}`)
    }
  }

  const hasActiveFilters = activeFilterBadges.length > 0

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-2">
          <div className="space-y-0">
            <h1 className="text-[30px] font-bold tracking-tight">Logs & Monitoring</h1>
            <p className="text-sm leading-6 text-muted-foreground">
              Investigate requests, errors, and live activity.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline" className="text-xs">{getTimeRangeLabel(selectedTimeRange)}</Badge>
            <Badge
              variant="outline"
              className={loading || isRealtimePaused ? 'border-amber-500 text-amber-700 dark:text-amber-300' : realtimeHealthy ? 'border-green-500 text-green-600 dark:text-green-400' : 'border-red-500 text-red-600 dark:text-red-400'}
            >
              <Clock className="mr-1 h-3 w-3" />
              {liveStatusText}
            </Badge>
            {activeTab === 'statistics' ? (
              <Badge variant="outline" className="text-xs">
                {formatDisplayNumber(statistics?.totalLogs ?? 0)} logs
              </Badge>
            ) : (
              <>
                <Badge variant="outline" className="text-xs">{formatDisplayNumber(totalCount)} total</Badge>
                <Badge variant="outline" className="text-xs">{formatDisplayNumber(logs.length)} shown</Badge>
              </>
            )}
          </div>
        </div>
        <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
          <Button
            variant="outline"
            size="sm"
            className="min-h-11 w-full sm:w-auto"
            onClick={handleRefresh}
            disabled={loading}
          >
            <RefreshCw
              className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`}
              aria-hidden="true"
            />
            Refresh
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="min-h-11 w-full sm:w-auto"
            onClick={handleDownloadLogs}
            disabled={loading || exportLoading || exportLoadFailed || noExportableLogs}
            title={exportLoadFailed ? 'Retry loading logs before exporting' : noExportableLogs ? 'No logs available to download' : 'Download logs'}
            aria-label={exportLoadFailed ? 'Download logs disabled: retry loading logs before exporting' : noExportableLogs ? 'Download logs disabled: no logs available' : 'Download logs'}
          >
            <Download className="mr-2 h-4 w-4" />
            Download logs
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
              <Button variant="ghost" size="sm" className="h-11 w-11 p-0">
                {filtersOpen ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                <span className="sr-only">{filtersOpen ? 'Collapse filters' : 'Expand filters'}</span>
              </Button>
            </CollapsibleTrigger>
          </CardHeader>
          <CollapsibleContent>
            <CardContent>
              <div className="mb-4 flex flex-wrap items-center gap-2">
                <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Quick ranges</span>
                {quickTimeRanges.map((range) => (
                  <Button
                    key={range}
                    type="button"
                    variant={selectedTimeRange === range ? 'default' : 'outline'}
                    size="sm"
                    className="h-11 px-3 text-xs"
                    onClick={() => applyTimeRange(range)}
                    aria-pressed={selectedTimeRange === range}
                  >
                    {getTimeRangeLabel(range)}
                  </Button>
                ))}
              </div>
              <div className={`grid grid-cols-1 ${tableScopedFiltersEnabled ? 'md:grid-cols-4' : 'md:grid-cols-2'} gap-3`}>
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
                      placeholder="e.g., req_123, user_42, timeout, or curl/8.0"
                      value={searchTerm}
                      onChange={(e) => { setSearchTerm(e.target.value); setCurrentPage(1) }}
                      className="h-11 pl-10"
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
                    onValueChange={applyTimeRange}
                  >
                    <SelectTrigger id="time-range-filter" className="h-11">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {activeTab === 'statistics' ? null : <SelectItem value="all">All Time</SelectItem>}
                      <SelectItem value="1h">Last hour</SelectItem>
                      <SelectItem value="6h">Last 6 hours</SelectItem>
                      <SelectItem value="24h">Last 24 hours</SelectItem>
                      <SelectItem value="7d">Last 7 days</SelectItem>
                      {activeTab === 'statistics' ? null : <SelectItem value="30d">Last 30 days</SelectItem>}
                    </SelectContent>
                  </Select>
                </div>

                {tableScopedFiltersEnabled ? (
                <div className="space-y-2">
                  <Label htmlFor="level-filter">Level</Label>
                  <Select value={levelFilter} onValueChange={(value) => { setLevelFilter(value); setCurrentPage(1) }}>
                    <SelectTrigger id="level-filter" className="h-11">
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
                    <SelectTrigger id="source-filter" className="h-11">
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
                    <Button variant="ghost" size="sm" className="min-h-11 px-3 text-xs" onClick={clearFilters}>
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
      {activeTab !== "statistics" && loadError ? (
        <div role="alert" className="flex flex-col items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300 sm:flex-row sm:items-center">
          <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
          <span>{loadError}</span>
          <Button variant="outline" size="sm" className="min-h-11 w-full sm:ml-auto sm:w-auto" onClick={handleRefresh}>Retry</Button>
        </div>
      ) : null}

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="h-11 w-full justify-start overflow-x-auto">
          <TabsTrigger value="table" className="min-h-11 px-4">Table View</TabsTrigger>
          <TabsTrigger value="raw" className="min-h-11 px-4">Raw View</TabsTrigger>
          <TabsTrigger value="statistics" className="min-h-11 px-4">Statistics</TabsTrigger>
        </TabsList>

        <TabsContent value="table">
          <Card>
            <CardHeader>
              <CardTitle className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <span>
                  {loading ? 'Server Logs' : `Server Logs (${formatDisplayNumber(totalCount)} total, showing ${formatDisplayNumber(logs.length)})`}
                </span>
                <Badge variant="outline" className={loading || currentPage !== 1 || activeTab !== 'table' || debouncedSearchTerm ? 'border-amber-500 text-amber-700 dark:text-amber-300' : realtimeHealthy ? 'border-green-500 text-green-600 dark:text-green-400' : 'border-red-500 text-red-600 dark:text-red-400'}>
                  <Clock className="mr-1 h-3 w-3" />
                  {liveStatusText}
                </Badge>
              </CardTitle>
              <CardDescription>
                {!loading && totalPages > 1 && (
                  <span>
                    Page {formatDisplayNumber(currentPage)} of {formatDisplayNumber(totalPages)}
                  </span>
                )}
                <span className="block text-xs mt-1">
                  {loading
                    ? 'Fetching logs…'
                    : isRealtimePaused
                    ? activeTab === 'statistics'
                      ? 'Live updates paused in Statistics view.'
                      : 'Live updates pause while searching or browsing older pages.'
                    : realtimeHealthy
                    ? 'Live updates run every 10 seconds.'
                    : 'Live updates stopped. Select Refresh to load the latest logs.'}
                </span>
              </CardDescription>
            </CardHeader>
            <CardContent>
              {loading ? (
                <div className="flex items-center justify-center py-8 md:hidden" role="status" aria-live="polite">
                  <div className="text-center">
                    <Loader2 className="mx-auto mb-2 h-5 w-5 animate-spin text-muted-foreground" aria-hidden="true" />
                    <p className="text-sm text-muted-foreground">Loading filtered logs...</p>
                  </div>
                </div>
              ) : logs.length === 0 ? (
                <div className="flex flex-col items-center gap-2 py-8 text-center md:hidden">
                  {loadError ? (
                    <div className="flex flex-col items-center gap-2" role="alert">
                      <AlertTriangle className="h-5 w-5 text-red-500 dark:text-red-400" aria-hidden="true" />
                      <p className="text-sm text-red-600 dark:text-red-400">{loadError}</p>
                      <Button variant="outline" size="sm" className="min-h-11" onClick={handleRefresh}>
                        Retry
                      </Button>
                    </div>
                  ) : (
                    <>
                      <p className="text-sm text-muted-foreground">No logs found for the current filters.</p>
                      <p className="text-xs text-muted-foreground">
                        {debouncedSearchTerm.trim()
                          ? `No results for "${debouncedSearchTerm.trim()}". Try a different search term or reset filters.`
                          : 'Try broadening the time range or resetting filters.'}
                      </p>
                      {hasActiveFilters ? (
                        <Button variant="ghost" size="sm" className="min-h-11" onClick={clearFilters}>
                          Reset filters
                        </Button>
                      ) : timeFilter === DEFAULT_TIME_RANGE ? (
                        <Button variant="outline" size="sm" className="min-h-11" onClick={() => { setTimeFilter('7d'); setCurrentPage(1) }}>
                          Show last 7 days
                        </Button>
                      ) : null}
                    </>
                  )}
                </div>
              ) : (
                <div className="space-y-3 md:hidden">
                  {logs.map((log) => {
                    const requestSummary = [
                      log.details.method || null,
                      log.details.statusCode != null ? `Status ${formatDisplayNumber(log.details.statusCode)}` : null,
                      log.details.responseTime != null ? formatResponseTime(log.details.responseTime) : null,
                    ].filter(Boolean).join(' · ')

                    return (
                      <Dialog key={log.id}>
                        <Card className="border border-border/60 shadow-sm">
                          <CardContent className="space-y-4 p-4">
                            <div className="flex items-start justify-between gap-3">
                              <div className="min-w-0 space-y-2">
                                <p className="font-mono text-xs text-muted-foreground">
                                  {formatLogTimestamp(log.timestamp)}
                                </p>
                                <div className="flex flex-wrap items-center gap-2">
                                  <button
                                    type="button"
                                    className={`rounded-full transition-opacity duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${levelFilter === log.level ? 'ring-2 ring-primary ring-offset-1' : 'hover:opacity-75'}`}
                                    aria-label={levelFilter === log.level ? `Clear ${LOG_LEVEL_DISPLAY[log.level] ?? log.level} filter` : `Filter logs to ${LOG_LEVEL_DISPLAY[log.level] ?? log.level} level`}
                                    aria-pressed={levelFilter === log.level}
                                    onClick={() => { setLevelFilter(prev => prev === log.level ? 'all' : log.level); setCurrentPage(1) }}
                                  >
                                    <Badge
                                      variant={getLevelColor(log.level) === 'destructive' ? 'destructive' : 'outline'}
                                      className={getLevelBadgeClass(log.level)}
                                    >
                                      {getLevelIcon(log.level)}
                                      <span className="ml-1">{LOG_LEVEL_DISPLAY[log.level] ?? log.level}</span>
                                    </Badge>
                                  </button>
                                  <button
                                    type="button"
                                    className={`rounded-sm transition-opacity duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${sourceFilter === log.source ? 'ring-2 ring-primary ring-offset-1' : 'hover:opacity-75'}`}
                                    aria-label={sourceFilter === log.source ? `Clear ${getDomainLabel(log.source)} source filter` : `Filter logs to ${getDomainLabel(log.source)} source`}
                                    aria-pressed={sourceFilter === log.source}
                                    onClick={() => { setSourceFilter(prev => prev === log.source ? 'all' : log.source); setCurrentPage(1) }}
                                  >
                                    <Badge variant="outline" className="text-xs">
                                      {getDomainLabel(log.source)}
                                    </Badge>
                                  </button>
                                </div>
                              </div>
                              {log.requestId ? (
                                <button
                                  type="button"
                                  className={`shrink-0 inline-flex items-center rounded-md border px-3 py-2 font-mono text-xs transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${searchTerm.trim() === log.requestId ? 'border-primary bg-accent text-accent-foreground shadow-sm' : 'border-input bg-background hover:bg-muted/50'}`}
                                  title={`${log.requestId} — click to filter by this request`}
                                  aria-label={`Filter logs by request ${log.requestId}`}
                                  aria-pressed={searchTerm.trim() === log.requestId}
                                  onClick={() => { setSearchTerm(log.requestId!); setCurrentPage(1) }}
                                >
                                  {log.requestId.slice(-6)}
                                </button>
                              ) : null}
                            </div>

                            <div className="space-y-1">
                              <p className="text-xs text-muted-foreground">Message</p>
                              <p className="line-clamp-3 text-sm" title={log.message}>
                                {log.message}
                              </p>
                            </div>

                            <div className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
                              <div>
                                <p className="text-xs text-muted-foreground">Request ID</p>
                                <p className="mt-1 break-all font-mono text-xs text-muted-foreground">
                                  {log.requestId || '—'}
                                </p>
                              </div>
                              <div>
                                <p className="text-xs text-muted-foreground">Request</p>
                                <p className="mt-1 text-sm text-muted-foreground">
                                  {requestSummary || 'No request metadata'}
                                </p>
                              </div>
                            </div>

                            <DialogTrigger asChild>
                              <Button variant="outline" size="sm" className="min-h-11 w-full">
                                View details
                              </Button>
                            </DialogTrigger>
                          </CardContent>
                        </Card>
                        {renderLogDetailsDialogContent(log)}
                      </Dialog>
                    )
                  })}
                </div>
              )}

              <div className="hidden overflow-x-auto md:block">
                <Table className="min-w-[900px]">
                  <TableHeader>
                    <TableRow>
                      <TableHead scope="col" className="w-[180px]">Timestamp</TableHead>
                      <TableHead scope="col" className="w-[96px]">Level</TableHead>
                      <TableHead scope="col" className="w-[180px]">Source</TableHead>
                      <TableHead scope="col" className="w-[360px]">Message</TableHead>
                      <TableHead scope="col" className="w-[140px]">Request ID</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {loading ? (
                      <TableRow>
                        <TableCell colSpan={5} className="py-8">
                          <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground" role="status" aria-live="polite">
                            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                            <span>Loading filtered logs...</span>
                          </div>
                        </TableCell>
                      </TableRow>
                    ) : logs.map((log) => (
                      <Dialog key={log.id}>
                        <TableRow>
                          <TableCell className="font-mono text-xs">
                            {formatLogTimestamp(log.timestamp)}
                          </TableCell>
                          <TableCell>
                            <button
                              type="button"
                              className={`rounded-full transition-opacity duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${levelFilter === log.level ? 'ring-2 ring-primary ring-offset-1' : 'hover:opacity-75'}`}
                              aria-label={levelFilter === log.level ? `Clear ${LOG_LEVEL_DISPLAY[log.level] ?? log.level} filter` : `Filter logs to ${LOG_LEVEL_DISPLAY[log.level] ?? log.level} level`}
                              aria-pressed={levelFilter === log.level}
                              onClick={() => { setLevelFilter(prev => prev === log.level ? 'all' : log.level); setCurrentPage(1) }}
                            >
                              <Badge
                                variant={getLevelColor(log.level) === 'destructive' ? 'destructive' : 'outline'}
                                className={getLevelBadgeClass(log.level)}
                              >
                                {getLevelIcon(log.level)}
                                <span className="ml-1">{LOG_LEVEL_DISPLAY[log.level] ?? log.level}</span>
                              </Badge>
                            </button>
                          </TableCell>
                          <TableCell>
                            <button
                              type="button"
                              className={`rounded-sm transition-opacity duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${sourceFilter === log.source ? 'ring-2 ring-primary ring-offset-1' : 'hover:opacity-75'}`}
                              aria-label={sourceFilter === log.source ? `Clear ${getDomainLabel(log.source)} source filter` : `Filter logs to ${getDomainLabel(log.source)} source`}
                              aria-pressed={sourceFilter === log.source}
                              onClick={() => { setSourceFilter(prev => prev === log.source ? 'all' : log.source); setCurrentPage(1) }}
                            >
                              <Badge
                                variant="outline"
                                className="whitespace-nowrap text-xs"
                              >
                                {getDomainLabel(log.source)}
                              </Badge>
                            </button>
                          </TableCell>
                          <TableCell className="max-w-[300px]">
                            <DialogTrigger asChild>
                              <button
                                type="button"
                                className="min-h-11 w-full truncate rounded py-2 text-left transition-colors duration-200 hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                                aria-label="Open log details"
                                title={log.message}
                              >
                                {log.message}
                              </button>
                            </DialogTrigger>
                          </TableCell>
                          <TableCell className="font-mono text-xs">
                            {log.requestId ? (
                              <button
                                type="button"
                                className={`inline-flex items-center rounded-md border px-3 py-2 font-mono text-xs transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ${searchTerm.trim() === log.requestId ? 'border-primary bg-accent text-accent-foreground shadow-sm' : 'border-input bg-background hover:bg-muted/50'}`}
                                title={`${log.requestId} — click to filter by this request`}
                                aria-label={`Filter logs by request ${log.requestId}`}
                                aria-pressed={searchTerm.trim() === log.requestId}
                                onClick={() => { setSearchTerm(log.requestId!); setCurrentPage(1) }}
                              >
                                {log.requestId.slice(-6)}
                              </button>
                            ) : (
                              <span className="text-muted-foreground">—</span>
                            )}
                          </TableCell>
                        </TableRow>
                        {renderLogDetailsDialogContent(log)}
                      </Dialog>
                    ))}
                    {logs.length === 0 && !loading && (
                      <TableRow>
                        <TableCell colSpan={5} className="py-8 text-center">
                          {loadError ? (
                            <div className="flex flex-col items-center gap-2" role="alert">
                              <AlertTriangle className="h-5 w-5 text-red-500 dark:text-red-400" aria-hidden="true" />
                              <p className="text-sm text-red-600 dark:text-red-400">{loadError}</p>
                              <Button variant="outline" size="sm" className="min-h-11" onClick={handleRefresh}>
                                Retry
                              </Button>
                            </div>
                          ) : (
                            <div className="flex flex-col items-center gap-2">
                              <p className="text-sm text-muted-foreground">No logs found for the current filters.</p>
                              <p className="text-xs text-muted-foreground">
                                {debouncedSearchTerm.trim()
                                  ? `No results for "${debouncedSearchTerm.trim()}". Try a different search term or reset filters.`
                                  : 'Try broadening the time range or resetting filters.'}
                              </p>
                              {hasActiveFilters ? (
                                <Button variant="ghost" size="sm" className="min-h-11" onClick={clearFilters}>
                                  Reset filters
                                </Button>
                              ) : timeFilter === DEFAULT_TIME_RANGE ? (
                                <Button variant="outline" size="sm" className="min-h-11" onClick={() => { setTimeFilter('7d'); setCurrentPage(1) }}>
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
                <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div className="text-sm text-muted-foreground">
                    Showing {(currentPage - 1) * pageSize + 1} to {formatDisplayNumber(Math.min(currentPage * pageSize, totalCount))} of {formatDisplayNumber(totalCount)} logs
                  </div>
                  <Pagination className="mx-0 w-auto justify-start sm:justify-end">
                    <PaginationContent>
                      <PaginationItem>
                        <Button
                          variant="outline"
                          size="sm"
                          className="min-h-11"
                          onClick={() =>
                            setCurrentPage((prev) => Math.max(1, prev - 1))
                          }
                          disabled={currentPage === 1 || loading}
                        >
                          Previous
                        </Button>
                      </PaginationItem>
                      <PaginationItem>
                        <span className="flex min-h-11 items-center px-3 text-sm">
                          Page {formatDisplayNumber(currentPage)} of {formatDisplayNumber(totalPages)}
                        </span>
                      </PaginationItem>
                      <PaginationItem>
                        <Button
                          variant="outline"
                          size="sm"
                          className="min-h-11"
                          onClick={() =>
                            setCurrentPage((prev) => Math.min(totalPages, prev + 1))
                          }
                          disabled={currentPage === totalPages || loading}
                        >
                          Next
                        </Button>
                      </PaginationItem>
                    </PaginationContent>
                  </Pagination>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="raw">
            <Card>
            <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <CardTitle>Raw Log View</CardTitle>
                <CardDescription>
                  {loading
                   ? 'Preparing raw logs for the current filtered page...'
                   : `Raw server logs for this filtered page (${formatDisplayNumber(logs.length)} rows on page ${formatDisplayNumber(currentPage)} of ${formatDisplayNumber(Math.max(totalPages, 1))})${debouncedSearchTerm.trim() ? ` — filtered by "${debouncedSearchTerm.trim()}"` : ''}`}
                </CardDescription>
              </div>
              {!loading && logs.length > 0 && (
                <Button
                  variant="outline"
                  size="sm"
                  className="min-h-11 w-full sm:w-auto sm:shrink-0"
                  onClick={() => void copyTextToClipboard(logs.map((log) => log.rawData).join('\n\n'), 'Raw logs copied to clipboard', 'Could not copy raw logs')}
                >
                  Copy All
                </Button>
              )}
            </CardHeader>
            <CardContent>
              {loading ? (
                <div className="flex items-center justify-center py-16 text-muted-foreground">
                  <div className="text-center">
                    <Loader2 className="h-6 w-6 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                    <p className="text-sm">Preparing raw log output...</p>
                  </div>
                </div>
              ) : logs.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-16 text-muted-foreground" role={loadError ? 'alert' : undefined}>
                  <Activity className="h-12 w-12 mb-3 opacity-40" />
                  <p className={loadError ? 'text-sm text-red-600 dark:text-red-400' : 'text-sm'}>{loadError || 'No logs available'}</p>
                  <p className="text-xs mt-1">{loadError ? 'Try Refresh to load data again' : 'Try adjusting your filters or check back later'}</p>
                  {loadError ? (
                    <Button variant="outline" size="sm" className="mt-3 min-h-11" onClick={handleRefresh}>
                      Retry
                    </Button>
                  ) : hasActiveFilters ? (
                    <Button variant="ghost" size="sm" className="mt-3 min-h-11" onClick={clearFilters}>
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
                  
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="statistics" className="space-y-4">
          {statsError ? (
            <div role="alert" className="flex flex-col items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300 sm:flex-row sm:items-center">
              <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{statsError}</span>
              <Button variant="outline" size="sm" className="min-h-11 w-full sm:ml-auto sm:w-auto" onClick={() => void loadStatistics()}>Retry</Button>
            </div>
          ) : null}
          {!STATISTICS_SUPPORTED_TIME_RANGES.has(timeFilter) ? (
            <div className="flex items-start gap-2 rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-sm text-blue-700 dark:border-blue-900/40 dark:bg-blue-950/20 dark:text-blue-300">
              <Info className="h-4 w-4 shrink-0 mt-0.5" aria-hidden="true" />
              <span>
                Statistics support up to 7 days. Your table filter ({getTimeRangeLabel(timeFilter)}) isn&apos;t available here, so this view is showing {getTimeRangeLabel(statisticsTimeFilter)} instead.
              </span>
            </div>
          ) : null}
          <div className="flex items-start gap-2 rounded-md border border-border/60 bg-muted/20 px-3 py-2 text-sm text-muted-foreground">
            <Info className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
            <p>
              These totals count log entries in {getTimeRangeLabel(statisticsTimeFilter)}. Error rate is the share of all logs marked <span className="font-medium text-foreground">ERROR</span>, and debug logs are tracked separately from <span className="font-medium text-foreground">Info Logs</span>.
            </p>
          </div>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <Card className="h-full">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Total Logs</CardTitle>
                <CardDescription className="text-xs">All log levels in {getTimeRangeLabel(statisticsTimeFilter)}</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col justify-center gap-1">
                <div className={statsLoading || !statistics ? (statsError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
                  {statsLoading
                    ? '—'
                    : !statistics
                    ? (statsError ? 'Unavailable' : '—')
                    : formatDisplayNumber(statistics.totalLogs, { compact: true })}
                </div>
                <p className="text-xs text-muted-foreground">Includes error, warning, info, and debug entries.</p>
              </CardContent>
            </Card>

            <Card className="h-full">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Error Logs</CardTitle>
                <CardDescription className="text-xs">Entries marked ERROR in {getTimeRangeLabel(statisticsTimeFilter)}</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col justify-center gap-1">
                <div className={statsLoading || !statistics ? (statsError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
                  {statsLoading
                    ? '—'
                    : !statistics
                    ? (statsError ? 'Unavailable' : '—')
                    : formatDisplayNumber(statistics.errorLogs, { compact: true })}
                </div>
                <p className={`text-xs ${!statsLoading && statistics ? (statistics.errorRate > 0 ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400') : 'text-muted-foreground'}`}>
                  {!statsLoading && statistics ? `${formatPercentage(statistics.errorRate)} of all logs` : ''}
                </p>
              </CardContent>
            </Card>

            <Card className="h-full">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Warning Logs</CardTitle>
                <CardDescription className="text-xs">Entries marked WARN in {getTimeRangeLabel(statisticsTimeFilter)}</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col justify-center gap-1">
                <div className={statsLoading || !statistics ? (statsError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
                  {statsLoading
                    ? '—'
                    : !statistics
                    ? (statsError ? 'Unavailable' : '—')
                    : formatDisplayNumber(statistics.warnLogs, { compact: true })}
                </div>
                <p className="text-xs text-muted-foreground">{!statsLoading && statistics && statistics.totalLogs > 0 ? `${formatPercentage((statistics.warnLogs / statistics.totalLogs) * 100)} of all logs` : ''}</p>
              </CardContent>
            </Card>

            <Card className="h-full">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Info Logs</CardTitle>
                <CardDescription className="text-xs">Entries marked INFO in {getTimeRangeLabel(statisticsTimeFilter)}</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col justify-center gap-1">
                <div className={statsLoading || !statistics ? (statsError ? "text-sm text-red-600 dark:text-red-400" : "text-sm text-muted-foreground") : "text-2xl font-bold"}>
                  {statsLoading
                    ? '—'
                    : !statistics
                    ? (statsError ? 'Unavailable' : '—')
                    : formatDisplayNumber(statistics.infoLogs, { compact: true })}
                </div>
                <p className="text-xs text-muted-foreground">
                  {!statsLoading && statistics ? `${formatDisplayNumber(statistics.debugLogs)} debug logs tracked separately` : ''}
                </p>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Log Activity</CardTitle>
              <CardDescription>Total logs by hour with errors overlaid for the same period.</CardDescription>
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
                <div className="flex flex-col items-center justify-center h-[300px] gap-2 text-muted-foreground" role="alert">
                  <Activity className="h-12 w-12 mb-1 opacity-40" />
                  <p className="text-sm text-red-600 dark:text-red-400">{statsError}</p>
                  <Button variant="outline" size="sm" className="min-h-11" onClick={() => void loadStatistics()}>Retry</Button>
                </div>
              ) : !statistics?.hourlyStats || statistics.hourlyStats.length === 0 ? (
                <div className="flex flex-col items-center justify-center h-[300px] text-muted-foreground">
                  <Activity className="h-12 w-12 mb-3 opacity-40" />
                  <p className="text-sm">No log activity is available for this period. Expand the time range or check back after more traffic arrives.</p>
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
                    <Area type="monotone" dataKey="totalCount" name="Total" stroke={chartPrimaryColor} fill={chartPrimaryColor} fillOpacity={0.1} />
                    <Area type="monotone" dataKey="errorCount" name="Errors" stroke={chartDestructiveColor} fill={chartDestructiveColor} fillOpacity={0.1} />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Log Sources</CardTitle>
              <CardDescription>Each source's share of total logs in this time range.</CardDescription>
            </CardHeader>
            <CardContent>
              {statsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <div className="text-center">
                    <Loader2 className="h-6 w-6 animate-spin mx-auto mb-2 text-muted-foreground" aria-hidden="true" />
                    <p className="text-sm text-muted-foreground">Loading log source breakdown...</p>
                  </div>
                </div>
              ) : statsError ? (
                <div className="flex flex-col items-center justify-center gap-2 py-8" role="alert">
                  <p className="text-sm text-red-600 dark:text-red-400">{statsError}</p>
                  <Button variant="outline" size="sm" className="min-h-11" onClick={() => void loadStatistics()}>Retry</Button>
                </div>
              ) : !statistics?.domainStats || statistics.domainStats.length === 0 ? (
                <div className="flex items-center justify-center py-8">
                  <p className="text-sm text-muted-foreground">No log sources are available for this period. Expand the time range or check back after more traffic arrives.</p>
                </div>
              ) : (
                <>
                  <div className="space-y-3 md:hidden">
                    {statistics.domainStats.map((stat) => (
                      <Card key={stat.domain} className="border border-border/60 shadow-sm">
                        <CardContent className="space-y-3 p-4">
                          <div>
                            <p className="text-sm font-medium">{stat.label}</p>
                            <p className="mt-1 text-xs text-muted-foreground">{formatPercentage(stat.percentage)} of logs</p>
                          </div>
                          <div className="grid grid-cols-3 gap-3 text-sm">
                            <div>
                              <p className="text-xs text-muted-foreground">Logs</p>
                              <p className="mt-1">{formatDisplayNumber(stat.logCount, { compact: true })}</p>
                            </div>
                            <div>
                              <p className="text-xs text-muted-foreground">Errors</p>
                              <p className={stat.errorCount > 0 ? 'mt-1 font-medium text-red-600 dark:text-red-400' : 'mt-1 text-muted-foreground'}>
                                {stat.errorCount > 0 ? formatDisplayNumber(stat.errorCount, { compact: true }) : '—'}
                              </p>
                            </div>
                            <div>
                              <p className="text-xs text-muted-foreground">Share</p>
                              <p className="mt-1">{formatPercentage(stat.percentage)}</p>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>

                  <div className="hidden overflow-x-auto md:block">
                    <Table className="min-w-[480px]">
                      <TableHeader>
                        <TableRow>
                          <TableHead scope="col">Source</TableHead>
                          <TableHead scope="col">Logs</TableHead>
                          <TableHead scope="col">Errors</TableHead>
                          <TableHead scope="col">% Share</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {statistics.domainStats.map((stat) => (
                          <TableRow key={stat.domain}>
                            <TableCell>{stat.label}</TableCell>
                            <TableCell>{formatDisplayNumber(stat.logCount, { compact: true })}</TableCell>
                            <TableCell className={stat.errorCount > 0 ? 'text-red-600 dark:text-red-400 font-medium' : 'text-muted-foreground'}>
                              {stat.errorCount > 0 ? formatDisplayNumber(stat.errorCount, { compact: true }) : '—'}
                            </TableCell>
                            <TableCell>{formatPercentage(stat.percentage)}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </>
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
