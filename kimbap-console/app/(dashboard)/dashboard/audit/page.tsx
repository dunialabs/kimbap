'use client';

import {
  ClipboardList,
  Search,
  Eye,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  RefreshCw,
  User,
  Timer,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Copy,
  Inbox,
} from 'lucide-react';
import { useState, useEffect, useCallback, useRef } from 'react';
import { toast } from 'sonner';

import { api } from '@/lib/api-client';
import { useDebounce } from '@/hooks/use-debounce';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  ScrollableDialogContent,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { getDomainLabel } from '@/lib/log-utils';

interface LogDetails {
  method?: string;
  url?: string;
  statusCode?: number;
  responseTime?: number;
  userAgent?: string;
  ip?: string;
  tokenId?: string;
  toolName?: string;
  errorType?: string;
  stackTrace?: string;
}

interface LogEntry {
  id: string;
  timestamp: string;
  level: 'INFO' | 'WARN' | 'ERROR' | 'DEBUG';
  message: string;
  source: string;
  requestId?: string;
  userId?: string;
  rawData: string;
  details: LogDetails;
}

interface LogsApiResponse {
  logs: LogEntry[];
  totalCount: number;
  totalPages: number;
  availableSources: string[];
}

type AuditStatus = 'all' | 'success' | 'warning' | 'error';

const STATUS_TO_LEVEL: Record<string, string | undefined> = {
  all: undefined,
  success: 'INFO',
  warning: 'WARN',
  error: 'ERROR',
};

function getStatusFromLevel(level: string): AuditStatus {
  switch (level) {
    case 'ERROR':
      return 'error';
    case 'WARN':
      return 'warning';
    default:
      return 'success';
  }
}

function getStatusBadge(level: string) {
  const status = getStatusFromLevel(level);
  switch (status) {
    case 'error':
      return (
        <Badge variant="destructive" className="text-xs gap-1">
          <XCircle className="h-3 w-3" aria-hidden="true" />
          Error
        </Badge>
      );
    case 'warning':
      return (
        <Badge
          variant="outline"
          className="border-yellow-500 bg-yellow-50 text-yellow-700 dark:bg-yellow-950/20 dark:text-yellow-400 dark:border-yellow-700 text-xs gap-1"
        >
          <AlertTriangle className="h-3 w-3" aria-hidden="true" />
          Warning
        </Badge>
      );
    default:
      return (
        <Badge
          variant="outline"
          className="border-green-500 bg-green-50 text-green-700 dark:bg-green-950/20 dark:text-green-400 dark:border-green-700 text-xs gap-1"
        >
          <CheckCircle2 className="h-3 w-3" aria-hidden="true" />
          Success
        </Badge>
      );
  }
}

function formatDuration(ms: number | undefined): string {
  if (ms == null) return '—';
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function formatRelativeTime(timestamp?: string): string {
  if (!timestamp) return '—';
  const iso = timestamp.includes('T') ? timestamp : timestamp.replace(' ', 'T');
  const date = new Date(iso.endsWith('Z') || /[+-]\d{2}:?\d{2}$/.test(iso) ? iso : iso + 'Z');
  if (Number.isNaN(date.getTime())) return '—';
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  if (diffMs < 0) return '—';
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return 'Just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDays = Math.floor(diffHr / 24);
  return `${diffDays}d ago`;
}

export default function AuditPage() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const requestIdRef = useRef(0);

  const [timeFilter, setTimeFilter] = useState('24h');
  const [statusFilter, setStatusFilter] = useState<AuditStatus>('all');
  const [sourceFilter, setSourceFilter] = useState('admin');
  const [searchTerm, setSearchTerm] = useState('');
  const debouncedSearchTerm = useDebounce(searchTerm, 400);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(25);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const loadAuditRecords = useCallback(async (): Promise<boolean> => {
    const reqId = ++requestIdRef.current;
    setLoading(true);

    try {
      const response = await api.audit.getRecords({
        page: currentPage,
        pageSize,
        timeRange: timeFilter,
        level: STATUS_TO_LEVEL[statusFilter],
        source: sourceFilter,
        search: debouncedSearchTerm || undefined,
      });

      if (reqId !== requestIdRef.current) return false;

      if (response.data?.common?.code === 0) {
        const raw = response.data.data;
        const data: LogsApiResponse = {
          logs: raw?.logs ?? [],
          totalCount: raw?.totalCount ?? 0,
          totalPages: raw?.totalPages ?? 0,
          availableSources: raw?.availableSources ?? [],
        };
        const safeTotalPages = Math.max(data.totalPages || 0, 1);
        const clampedPage = Math.min(Math.max(currentPage, 1), safeTotalPages);
        if (clampedPage !== currentPage) {
          setCurrentPage(clampedPage);
          return true;
        }
        setLogs(data.logs);
        setTotalCount(data.totalCount);
        setTotalPages(data.logs.length === 0 ? 0 : data.totalPages);
        setLoadError(null);
        return true;
      } else {
        toast.error(
          'Unable to load audit records: ' + (response.data?.common?.message || 'Unknown error'),
        );
        setLogs([]);
        setTotalCount(0);
        setTotalPages(0);
        setLoadError('Unable to load audit records. Check your connection and try again.');
        return false;
      }
    } catch {
      if (reqId !== requestIdRef.current) return false;
      toast.error('Error loading audit records');
      setLogs([]);
      setTotalCount(0);
      setTotalPages(0);
      setLoadError('Unable to load audit records. Check your connection and try again.');
      return false;
    } finally {
      if (reqId === requestIdRef.current) setLoading(false);
    }
  }, [currentPage, pageSize, timeFilter, statusFilter, sourceFilter, debouncedSearchTerm]);

  useEffect(() => {
    loadAuditRecords();
  }, [loadAuditRecords]);

  const handleRefresh = async () => {
    const ok = await loadAuditRecords();
    if (ok) toast.success('Audit records refreshed');
  };

  const clearFilters = () => {
    setTimeFilter('24h');
    setStatusFilter('all');
    setSourceFilter('admin');
    setSearchTerm('');
    setCurrentPage(1);
  };

  const getTimeRangeLabel = (range: string) => {
    switch (range) {
      case '1h':
        return 'Last hour';
      case '6h':
        return 'Last 6 hours';
      case '24h':
        return 'Last 24 hours';
      case '7d':
        return 'Last 7 days';
      case '30d':
        return 'Last 30 days';
      case 'all':
        return 'All time';
      default:
        return range;
    }
  };

  const activeFilterBadges: string[] = [];
  if (timeFilter !== '24h') activeFilterBadges.push(`Time: ${getTimeRangeLabel(timeFilter)}`);
  if (statusFilter !== 'all') activeFilterBadges.push(`Status: ${statusFilter}`);
  if (sourceFilter !== 'admin') activeFilterBadges.push(`Domain: ${getDomainLabel(sourceFilter)}`);
  if (debouncedSearchTerm.trim()) activeFilterBadges.push(`Search: ${debouncedSearchTerm.trim()}`);
  const hasActiveFilters = activeFilterBadges.length > 0;

  const rangeStart = totalCount > 0 ? (currentPage - 1) * pageSize + 1 : 0;
  const rangeEnd = Math.min(currentPage * pageSize, totalCount);

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="space-y-0">
          <h1 className="text-[30px] font-bold flex items-center gap-2.5">
            <ClipboardList className="h-7 w-7 text-muted-foreground" aria-hidden="true" />
            Audit
          </h1>
          <p className="text-base text-muted-foreground">
            Audit records — who did what, when, and with what result.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={handleRefresh} disabled={loading}>
          <RefreshCw
            className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`}
            aria-hidden="true"
          />
          Refresh
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Filters</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-5 gap-3">
            <div className="space-y-2">
              <Label htmlFor="audit-search">Actor / Search</Label>
              <div className="relative">
                <Search
                  className="absolute left-3 top-3 h-4 w-4 text-muted-foreground pointer-events-none"
                  aria-hidden="true"
                />
                <Input
                  id="audit-search"
                  placeholder="Search by user ID, action, or error"
                  value={searchTerm}
                  onChange={(e) => {
                    setSearchTerm(e.target.value);
                    setCurrentPage(1);
                  }}
                  className="pl-10"
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="audit-time-range">Time Range</Label>
              <Select
                value={timeFilter}
                onValueChange={(value) => {
                  setTimeFilter(value);
                  setCurrentPage(1);
                }}
              >
                <SelectTrigger id="audit-time-range">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1h">Last Hour</SelectItem>
                  <SelectItem value="6h">Last 6 Hours</SelectItem>
                  <SelectItem value="24h">Last 24 Hours</SelectItem>
                  <SelectItem value="7d">Last 7 Days</SelectItem>
                  <SelectItem value="30d">Last 30 Days</SelectItem>
                  <SelectItem value="all">All Time</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="audit-domain">Domain</Label>
              <Select
                value={sourceFilter}
                onValueChange={(value) => {
                  setSourceFilter(value);
                  setCurrentPage(1);
                }}
              >
                <SelectTrigger id="audit-domain">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="admin">Admin</SelectItem>
                  <SelectItem value="auth">Auth</SelectItem>
                  <SelectItem value="oauth">OAuth</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="audit-status">Status</Label>
              <Select
                value={statusFilter}
                onValueChange={(value) => {
                  setStatusFilter(value as AuditStatus);
                  setCurrentPage(1);
                }}
              >
                <SelectTrigger id="audit-status">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Statuses</SelectItem>
                  <SelectItem value="success">Success</SelectItem>
                  <SelectItem value="warning">Warning</SelectItem>
                  <SelectItem value="error">Error</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="flex items-end">
              <Button variant="outline" onClick={clearFilters}>
                Reset filters
              </Button>
            </div>
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
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 text-xs"
                  onClick={clearFilters}
                >
                  Reset filters
                </Button>
              </div>
            ) : (
              <p className="text-xs text-muted-foreground">
                Showing Admin domain records from the last 24 hours with all statuses.
              </p>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>
              {loading ? 'Audit Records' : `Audit Records (${totalCount.toLocaleString()} total)`}
            </span>
            {!loading && totalPages > 1 && (
              <span className="text-sm font-normal text-muted-foreground">
                Page {currentPage} of {totalPages}
              </span>
            )}
          </CardTitle>
          {!loading && totalCount > 0 && (
            <CardDescription>
              Showing {rangeStart} to {rangeEnd} of {totalCount.toLocaleString()} records
            </CardDescription>
          )}
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <TooltipProvider>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[160px]">Time</TableHead>
                    <TableHead className="w-[120px]">Actor</TableHead>
                    <TableHead className="w-[150px]">Action</TableHead>
                    <TableHead className="w-[140px]">Target</TableHead>
                    <TableHead className="w-[110px]">Status</TableHead>
                    <TableHead className="w-[90px]">Duration</TableHead>
                    <TableHead className="w-[60px]" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loading ? (
                    <TableRow>
                      <TableCell colSpan={7} className="text-center py-12">
                        <div className="flex flex-col items-center gap-2">
                          <RefreshCw
                            className="h-5 w-5 animate-spin text-muted-foreground"
                            aria-hidden="true"
                          />
                          <p className="text-sm text-muted-foreground">Loading audit records...</p>
                        </div>
                      </TableCell>
                    </TableRow>
                  ) : logs.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={7} className="text-center py-12">
                        {loadError ? (
                          <div className="flex flex-col items-center gap-2">
                            <AlertTriangle
                              className="h-5 w-5 text-red-500 dark:text-red-400"
                              aria-hidden="true"
                            />
                            <p className="text-sm text-red-600 dark:text-red-400">{loadError}</p>
                            <Button variant="outline" size="sm" onClick={handleRefresh}>
                              Retry
                            </Button>
                          </div>
                        ) : (
                          <div className="flex flex-col items-center gap-3">
                            <Inbox
                              className="h-10 w-10 text-muted-foreground/40"
                              aria-hidden="true"
                            />
                            <p className="text-sm text-muted-foreground">
                              No audit records found for the current filters.
                            </p>
                            <p className="text-xs text-muted-foreground">
                              Try broadening the time range or resetting filters.
                            </p>
                            {hasActiveFilters && (
                              <Button variant="ghost" size="sm" onClick={clearFilters}>
                                Reset filters
                              </Button>
                            )}
                          </div>
                        )}
                      </TableCell>
                    </TableRow>
                  ) : (
                    logs.map((log) => (
                      <Dialog key={log.id}>
                        <DialogTrigger asChild>
                          <TableRow className="cursor-pointer hover:bg-muted/50">
                            <TableCell>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="font-mono text-xs">{log.timestamp}</span>
                                </TooltipTrigger>
                                <TooltipContent>{formatRelativeTime(log.timestamp)}</TooltipContent>
                              </Tooltip>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-1.5">
                                <User
                                  className="h-3.5 w-3.5 text-muted-foreground shrink-0"
                                  aria-hidden="true"
                                />
                                <span
                                  className="text-sm truncate max-w-[90px]"
                                  title={log.userId || 'System'}
                                >
                                  {log.userId || 'System'}
                                </span>
                              </div>
                            </TableCell>
                            <TableCell>
                              <Badge variant="secondary" className="text-xs font-normal">
                                {log.details.toolName || log.message.split(' ')[0] || 'Unknown'}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <Badge
                                variant="outline"
                                className="text-xs font-normal whitespace-nowrap"
                              >
                                {getDomainLabel(log.source)}
                              </Badge>
                            </TableCell>
                            <TableCell>{getStatusBadge(log.level)}</TableCell>
                            <TableCell>
                              <span className="font-mono text-xs text-muted-foreground flex items-center gap-1">
                                <Timer className="h-3 w-3" aria-hidden="true" />
                                {formatDuration(log.details.responseTime)}
                              </span>
                            </TableCell>
                            <TableCell>
                              <Button
                                variant="ghost"
                                size="sm"
                                aria-label="View audit record details"
                              >
                                <Eye className="h-4 w-4" />
                              </Button>
                            </TableCell>
                          </TableRow>
                        </DialogTrigger>
                        <ScrollableDialogContent className="max-w-3xl">
                          <DialogHeader>
                            <DialogTitle className="flex items-center gap-2">
                              <ClipboardList className="h-5 w-5" aria-hidden="true" />
                              Audit Record Detail
                            </DialogTitle>
                            <DialogDescription>
                              {log.timestamp} — {log.details.toolName || 'Unknown Action'} by{' '}
                              {log.userId || 'System'}
                            </DialogDescription>
                          </DialogHeader>

                          <div className="space-y-4">
                            <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
                              <div className="space-y-1">
                                <Label className="text-xs text-muted-foreground">Actor</Label>
                                <p className="text-sm font-medium">{log.userId || 'System'}</p>
                              </div>
                              <div className="space-y-1">
                                <Label className="text-xs text-muted-foreground">Action</Label>
                                <p className="text-sm font-medium">
                                  {log.details.toolName || 'Unknown'}
                                </p>
                              </div>
                              <div className="space-y-1">
                                <Label className="text-xs text-muted-foreground">Status</Label>
                                <div>{getStatusBadge(log.level)}</div>
                              </div>
                              <div className="space-y-1">
                                <Label className="text-xs text-muted-foreground">Target</Label>
                                <p className="text-sm font-medium">{getDomainLabel(log.source)}</p>
                              </div>
                              <div className="space-y-1">
                                <Label className="text-xs text-muted-foreground">Duration</Label>
                                <p className="text-sm font-medium">
                                  {formatDuration(log.details.responseTime)}
                                </p>
                              </div>
                              <div className="space-y-1">
                                <Label className="text-xs text-muted-foreground">Session ID</Label>
                                <p
                                  className="text-sm font-mono truncate"
                                  title={log.requestId || '—'}
                                >
                                  {log.requestId || '—'}
                                </p>
                              </div>
                            </div>

                            {log.details.statusCode !== undefined && log.details.statusCode > 0 && (
                              <div className="space-y-1">
                                <Label className="text-xs text-muted-foreground">HTTP Status</Label>
                                <p className="text-sm font-mono">
                                  {log.details.statusCode}
                                  {log.details.statusCode >= 200 &&
                                    log.details.statusCode < 300 &&
                                    ' OK'}
                                  {log.details.statusCode >= 400 &&
                                    log.details.statusCode < 500 &&
                                    ' Client Error'}
                                  {log.details.statusCode >= 500 && ' Server Error'}
                                </p>
                              </div>
                            )}

                            {log.details.ip && (
                              <div className="space-y-1">
                                <Label className="text-xs text-muted-foreground">IP Address</Label>
                                <p className="text-sm font-mono">{log.details.ip}</p>
                              </div>
                            )}

                            {log.details.errorType && (
                              <div className="space-y-1">
                                <Label className="text-xs text-muted-foreground">Error</Label>
                                <div className="p-3 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-900 rounded text-sm">
                                  <p className="font-medium text-red-700 dark:text-red-400">
                                    {log.details.errorType}
                                  </p>
                                  {log.details.stackTrace && (
                                    <pre className="mt-2 text-xs font-mono text-red-600 dark:text-red-300 whitespace-pre-wrap break-words">
                                      {log.details.stackTrace}
                                    </pre>
                                  )}
                                </div>
                              </div>
                            )}

                            <div className="space-y-1">
                              <Label className="text-xs text-muted-foreground">Summary</Label>
                              <p className="p-3 bg-muted rounded text-sm">{log.message}</p>
                            </div>

                            <div className="space-y-1">
                              <Label className="text-xs text-muted-foreground">Full Record</Label>
                              <pre className="p-3 bg-muted rounded text-xs font-mono whitespace-pre-wrap break-words max-h-[300px] overflow-y-auto">
                                {log.rawData}
                              </pre>
                            </div>
                          </div>

                          <DialogFooter className="border-t pt-4">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={async () => {
                                try {
                                  if (!navigator?.clipboard?.writeText) {
                                    toast.error('Clipboard not available');
                                    return;
                                  }
                                  await navigator.clipboard.writeText(log.rawData);
                                  toast.success('Record data copied to clipboard');
                                } catch {
                                  toast.error('Could not copy record data');
                                }
                              }}
                            >
                              <Copy className="mr-2 h-4 w-4" />
                              Copy Raw Data
                            </Button>
                          </DialogFooter>
                        </ScrollableDialogContent>
                      </Dialog>
                    ))
                  )}
                </TableBody>
              </Table>
            </TooltipProvider>
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4">
              <div className="text-sm text-muted-foreground">
                {rangeStart}–{rangeEnd} of {totalCount.toLocaleString()} records
              </div>
              <div className="flex items-center gap-1">
                <Button
                  variant="outline"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => setCurrentPage(1)}
                  disabled={currentPage === 1 || loading}
                  aria-label="First page"
                >
                  <ChevronsLeft className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => setCurrentPage((prev) => Math.max(1, prev - 1))}
                  disabled={currentPage === 1 || loading}
                  aria-label="Previous page"
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <span className="flex items-center px-3 py-1 text-sm text-muted-foreground">
                  {currentPage} / {totalPages}
                </span>
                <Button
                  variant="outline"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => setCurrentPage((prev) => Math.min(totalPages, prev + 1))}
                  disabled={currentPage === totalPages || loading}
                  aria-label="Next page"
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => setCurrentPage(totalPages)}
                  disabled={currentPage === totalPages || loading}
                  aria-label="Last page"
                >
                  <ChevronsRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
