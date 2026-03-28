'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { toast } from 'sonner';
import {
  CheckCircle2,
  XCircle,
  Clock,
  Eye,
  RefreshCw,
  Filter,
  AlertTriangle,
  ShieldCheck,
  Loader2,
} from 'lucide-react';

import { api } from '@/lib/api-client';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
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
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { formatDateTime, formatDisplayNumber, formatNullableText, formatRelativeMinutes } from '@/lib/utils';

// ─── Types ──────────────────────────────────────────────

interface ApprovalRequest {
  id: string;
  resumeToken?: string;
  userId: string;
  serverId: string | null;
  toolName: string;
  canonicalArgs: Record<string, any>;
  redactedArgs: Record<string, any>;
  requestHash: string;
  status: 'PENDING' | 'APPROVED' | 'REJECTED' | 'EXPIRED' | 'EXECUTING' | 'EXECUTED' | 'FAILED';
  reason?: string | null;
  decisionReason?: string;
  decidedByUserId?: string | null;
  decidedByRole?: number | null;
  decisionChannel?: 'admin_api' | 'socket' | null;
  executedAt?: string | null;
  executionError?: string | null;
  executionResultAvailable?: boolean;
  policyVersion: number;
  uniformRequestId?: string | null;
  createdAt: string;
  expiresAt: string;
  decidedAt?: string;
}

type StatusFilter =
  | 'all'
  | 'PENDING'
  | 'APPROVED'
  | 'REJECTED'
  | 'EXPIRED'
  | 'EXECUTING'
  | 'EXECUTED'
  | 'FAILED';

const BASE_PAGE_SIZE = 20;
const DEFAULT_STATUS_FILTER: StatusFilter = 'PENDING';

// ─── Helpers ────────────────────────────────────────────

function statusBadge(status: string) {
  switch (status) {
    case 'PENDING':
      return (
        <Badge className="bg-amber-100 text-amber-800 border-amber-200 dark:bg-amber-900 dark:text-amber-300 dark:border-amber-800 hover:bg-amber-200">
          <Clock className="h-3 w-3 mr-1" />
          Pending
        </Badge>
      );
    case 'APPROVED':
      return (
        <Badge className="bg-green-100 text-green-800 border-green-200 dark:bg-green-900 dark:text-green-300 dark:border-green-800 hover:bg-green-200">
          <CheckCircle2 className="h-3 w-3 mr-1" />
          Approved
        </Badge>
      );
    case 'REJECTED':
      return (
        <Badge className="bg-red-100 text-red-800 border-red-200 dark:bg-red-900 dark:text-red-300 dark:border-red-800 hover:bg-red-200">
          <XCircle className="h-3 w-3 mr-1" />
          Rejected
        </Badge>
      );
    case 'EXPIRED':
      return (
        <Badge className="bg-amber-100 text-amber-800 border-amber-200 dark:bg-amber-950/30 dark:text-amber-300 dark:border-amber-800 hover:bg-amber-200">
          <AlertTriangle className="h-3 w-3 mr-1" />
          Expired
        </Badge>
      );
    case 'EXECUTING':
      return (
        <Badge className="bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800 hover:bg-blue-200">
          <Loader2 className="h-3 w-3 mr-1 animate-spin" />
          Executing
        </Badge>
      );
    case 'EXECUTED':
      return (
        <Badge className="bg-sky-100 text-sky-800 border-sky-200 dark:bg-sky-900 dark:text-sky-300 dark:border-sky-800 hover:bg-sky-200">
          <CheckCircle2 className="h-3 w-3 mr-1" />
          Executed
        </Badge>
      );
    case 'FAILED':
      return (
        <Badge className="bg-red-100 text-red-800 border-red-200 dark:bg-red-950/30 dark:text-red-300 dark:border-red-800 hover:bg-red-200">
          <AlertTriangle className="h-3 w-3 mr-1" />
          Failed
        </Badge>
      );
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

function formatTime(iso: string): string {
  return formatDateTime(iso, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatExpiryTime(iso: string, status: string): { text: string; urgent: boolean } {
  if (!iso) return { text: '—', urgent: false };
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return { text: iso, urgent: false };

  if (status !== 'PENDING') {
    return { text: formatTime(iso), urgent: false };
  }

  const remainingMs = d.getTime() - Date.now();
  if (remainingMs <= 0) return { text: `Expired ${formatTime(iso)}`, urgent: false };

  const remainingMins = Math.ceil(remainingMs / 60000);
  if (remainingMins <= 60) {
    return { text: `Expires in ${remainingMins} min`, urgent: remainingMins <= 5 };
  }

  return { text: formatTime(iso), urgent: false };
}

function redactArgs(args: Record<string, any>): Record<string, any> {
  if (!args || typeof args !== 'object') return {};
  const redacted: Record<string, any> = {};
  for (const [key, value] of Object.entries(args)) {
    if (typeof value === 'string' && value.length > 100) {
      redacted[key] = value.slice(0, 80) + '...[redacted]';
    } else {
      redacted[key] = value;
    }
  }
  return redacted;
}

function roleLabel(role?: number | null): string {
  switch (role) {
    case 1:
      return 'Owner';
    case 2:
      return 'Admin';
    case 3:
      return 'User';
    case 4:
      return 'Guest';
    default:
      return '—';
  }
}

function channelLabel(channel: string | null | undefined): string {
  switch (channel) {
    case 'admin_api': return 'Admin API';
    case 'socket': return 'WebSocket';
    default: return channel || '—';
  }
}

function getRequestErrorMessage(
  error: unknown,
  messages: { auth: string; network: string; fallback: string }
): string {
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

// ─── Main Page ──────────────────────────────────────────

export default function ApprovalsPage() {
  const [requests, setRequests] = useState<ApprovalRequest[]>([]);
  const [pendingCount, setPendingCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>(DEFAULT_STATUS_FILTER);
  const [userFilter, setUserFilter] = useState('');
  const [detailDialog, setDetailDialog] = useState<ApprovalRequest | null>(null);
  const [decideDialog, setDecideDialog] = useState<{
    request: ApprovalRequest;
    decision: 'APPROVED' | 'REJECTED';
  } | null>(null);
  const [decideReason, setDecideReason] = useState('');
  const [deciding, setDeciding] = useState(false);
  const [detailDecideReason, setDetailDecideReason] = useState('');
  const [detailDeciding, setDetailDeciding] = useState(false);
  const [detailDecisionAction, setDetailDecisionAction] = useState<'APPROVED' | 'REJECTED' | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const decideReasonRef = useRef<HTMLTextAreaElement>(null);
  const [refreshFailed, setRefreshFailed] = useState(false);
  const [timeAgo, setTimeAgo] = useState('');
  const [loadedPages, setLoadedPages] = useState(1);
  const hasActiveFilters = statusFilter !== DEFAULT_STATUS_FILTER || userFilter.trim().length > 0;
  const refreshTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const tickRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const loadedPagesRef = useRef(1);
  const fetchVersionRef = useRef(0);


  useEffect(() => {
    document.title = 'Approvals | Kimbap Console';
  }, []);

  const fetchData = useCallback(
    async (options?: {
      page?: number;
      pageSize?: number;
      append?: boolean;
      status?: StatusFilter;
      userId?: string;
    }) => {
      const page = options?.page ?? 1;
      const pageSize = options?.pageSize ?? BASE_PAGE_SIZE;
      const append = options?.append === true;
      const status = options?.status ?? statusFilter;
      const userId = options?.userId ?? userFilter;

      if (append) {
        setLoadingMore(true);
      }

      const fetchVersion = ++fetchVersionRef.current;

      try {
        const [listRes, countRes] = await Promise.all([
          api.approvals.list({
            page,
            pageSize,
            ...(status !== 'all' ? { status } : {}),
            ...(userId.trim() ? { userId: userId.trim() } : {}),
          }),
          api.approvals.countPending().catch(() => null),
        ]);
        if (fetchVersion !== fetchVersionRef.current) {
          return;
        }

        const listData = listRes.data?.data || listRes.data;
        const countData = countRes?.data?.data || countRes?.data;
        const nextRequests = (listData?.requests || []) as ApprovalRequest[];

        setRequests((prev) => {
          if (!append) {
            return nextRequests;
          }

          const seen = new Set(prev.map((item) => item.id));
          return [...prev, ...nextRequests.filter((item) => !seen.has(item.id))];
        });
        if (countData) {
          setPendingCount(countData?.count || 0);
        }
        setHasMore(Boolean(listData?.hasMore));
        setLoadError(null);
        const resolvedPages = append ? page : Math.max(1, Math.ceil(pageSize / BASE_PAGE_SIZE));
        loadedPagesRef.current = resolvedPages;
        setLoadedPages(resolvedPages);
        setLastUpdated(new Date());
        setTimeAgo('just now');
        setRefreshFailed(false);
      } catch (error: unknown) {
        if (fetchVersion !== fetchVersionRef.current) {
          return;
        }
        setLoadError(
          getRequestErrorMessage(error, {
            auth: 'Could not load approval requests because your session expired or your access changed. Sign in again and retry.',
            network: 'Could not load approval requests. Check your connection and retry.',
            fallback: 'Could not load approval requests right now. Retry to refresh the queue.'
          })
        );
        setRefreshFailed(true);
      } finally {
        if (fetchVersion !== fetchVersionRef.current) {
          return;
        }
        setLoading(false);
        setLoadingMore(false);
      }
    },
    [statusFilter, userFilter],
  );

  useEffect(() => {
    setLoading(true);
    setHasMore(false);
    setRequests([]);
    loadedPagesRef.current = 1;
    setLoadedPages(1);

    void fetchData({
      page: 1,
      pageSize: BASE_PAGE_SIZE,
      status: statusFilter,
      userId: userFilter,
    });

    refreshTimerRef.current = setInterval(() => {
      void fetchData({
        page: 1,
        pageSize: loadedPagesRef.current * BASE_PAGE_SIZE,
        status: statusFilter,
        userId: userFilter,
      });
    }, 30000);
    tickRef.current = setInterval(() => {
      setLastUpdated((prev) => {
        if (!prev) return prev;
        const secs = Math.floor((Date.now() - prev.getTime()) / 1000);
        setTimeAgo(formatRelativeMinutes(secs / 60));
        return prev;
      });
    }, 10000);
    return () => {
      if (refreshTimerRef.current) clearInterval(refreshTimerRef.current);
      if (tickRef.current) clearInterval(tickRef.current);
    };
  }, [fetchData, statusFilter, userFilter]);

  useEffect(() => {
    if (!decideDialog) {
      return;
    }

    const frame = window.requestAnimationFrame(() => {
      decideReasonRef.current?.focus();
    });

    return () => window.cancelAnimationFrame(frame);
  }, [decideDialog]);

  const handleLoadMore = async () => {
    if (loadingMore || !hasMore) return;
    await fetchData({
      page: loadedPagesRef.current + 1,
      pageSize: BASE_PAGE_SIZE,
      append: true,
    });
  };

  const openDecideDialog = (request: ApprovalRequest, decision: 'APPROVED' | 'REJECTED') => {
    setDecideDialog({ request, decision });
    setDecideReason('');
  };

  const resetFilters = () => {
    setStatusFilter(DEFAULT_STATUS_FILTER);
    setUserFilter('');
  };

  const handleManualRefresh = async () => {
    setRefreshing(true);
    await fetchData({
      page: 1,
      pageSize: loadedPagesRef.current * BASE_PAGE_SIZE,
      status: statusFilter,
      userId: userFilter,
    });
    setRefreshing(false);
  };

  const handleDecide = async () => {
    if (!decideDialog) return;
    setDeciding(true);
    const decisionLabel = decideDialog.decision === 'APPROVED' ? 'approved' : 'rejected';
    const actionLabel = decideDialog.decision === 'APPROVED' ? 'approve' : 'reject';
    try {
      await api.approvals.decide({
        id: decideDialog.request.id,
        decision: decideDialog.decision,
        reason: decideReason.trim() || undefined,
      });
      toast.success(`${decideDialog.request.toolName} request ${decisionLabel}. The queue refreshed so you can review the next request.`);
      setDecideDialog(null);
      void fetchData({
        page: 1,
        pageSize: loadedPagesRef.current * BASE_PAGE_SIZE,
      });
    } catch (error: unknown) {
      toast.error(
        getRequestErrorMessage(error, {
          auth: `Could not ${actionLabel} ${decideDialog.request.toolName} because your session expired or your access changed. Sign in again and retry.`,
          network: `Could not ${actionLabel} ${decideDialog.request.toolName}. Check your connection and retry.`,
          fallback: `Could not ${actionLabel} ${decideDialog.request.toolName} request.`
        })
      );
    } finally {
      setDeciding(false);
    }
  };

  const handleDetailDecide = async (decision: 'APPROVED' | 'REJECTED') => {
    if (!detailDialog) return;
    setDetailDeciding(true);
    setDetailDecisionAction(decision);
    try {
      await api.approvals.decide({
        id: detailDialog.id,
        decision,
        reason: detailDecideReason.trim() || undefined,
      });
      const label = decision === 'APPROVED' ? 'approved' : 'rejected';
      toast.success(`${detailDialog.toolName} request ${label}. The queue refreshed so you can review the next request.`);
      setDetailDialog(null);
      setDetailDecideReason('');
      void fetchData({ page: 1, pageSize: loadedPagesRef.current * BASE_PAGE_SIZE });
    } catch (error: unknown) {
      const actionLabel = decision === 'APPROVED' ? 'approve' : 'reject';
      toast.error(
        getRequestErrorMessage(error, {
          auth: `Could not ${actionLabel} ${detailDialog.toolName} because your session expired or your access changed. Sign in again and retry.`,
          network: `Could not ${actionLabel} ${detailDialog.toolName}. Check your connection and retry.`,
          fallback: `Could not ${actionLabel} ${detailDialog.toolName} request.`
        })
      );
    } finally {
      setDetailDeciding(false);
      setDetailDecisionAction(null);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <div>
          <h1 className="text-[30px] font-bold tracking-tight flex items-center gap-2">
            <ShieldCheck className="h-6 w-6" />
            Approvals
          </h1>
          <p className="text-base text-muted-foreground">
            Review tool requests that are waiting for a decision. After each decision, the queue refreshes so you can continue with the next request.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {refreshFailed && <span className="text-xs text-amber-600">Last refresh failed</span>}
          {!refreshFailed && lastUpdated && (
            <span className="text-xs text-muted-foreground">Updated {timeAgo || 'just now'}</span>
          )}
          <Button
            variant="outline"
            size="sm"
            className="min-h-11"
            onClick={() => void handleManualRefresh()}
            disabled={loading || loadingMore || refreshing}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${loading || loadingMore || refreshing ? 'animate-spin' : ''}`} />
            {refreshing ? 'Refreshing...' : 'Refresh'}
          </Button>
        </div>
      </div>

      {/* Pending Count Banner */}
      {pendingCount > 0 && (
        <Card className="border-amber-500/30 bg-amber-500/5">
          <CardContent className="flex items-center justify-between gap-3 py-3">
            <div className="flex items-center gap-3">
              <Clock className="h-5 w-5 text-amber-600 shrink-0" />
              <span className="text-sm font-medium">
                {formatDisplayNumber(pendingCount)} pending approval{pendingCount !== 1 ? 's' : ''} awaiting review
              </span>
            </div>
            {statusFilter !== DEFAULT_STATUS_FILTER && (
              <Button
                variant="outline"
                size="sm"
                className="shrink-0"
                onClick={() => setStatusFilter(DEFAULT_STATUS_FILTER)}
              >
                View pending
              </Button>
            )}
          </CardContent>
        </Card>
      )}

      {/* Filters + Table */}
      <Card>
        <CardHeader>
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
                <CardTitle className="text-base">Approval Requests</CardTitle>
                <CardDescription>
                  {formatDisplayNumber(requests.length)} request{requests.length !== 1 ? 's' : ''}
                  {statusFilter !== 'all' ? ` (${statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1).toLowerCase()})` : ''}
                </CardDescription>
              </div>
            <div className="w-full rounded-lg border border-border/60 bg-muted/20 p-3 sm:w-auto">
              <div className="mb-3 flex items-center gap-1.5">
                <Filter className="h-3.5 w-3.5 text-muted-foreground" aria-hidden="true" />
                <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Filters</span>
              </div>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-[minmax(0,140px)_minmax(0,180px)_auto]">
                <div className="space-y-1.5">
                  <Label htmlFor="approvals-status-filter" className="text-xs">Status</Label>
                  <Select
                    value={statusFilter}
                    onValueChange={(v) => setStatusFilter(v as StatusFilter)}
                  >
                    <SelectTrigger id="approvals-status-filter" className="h-11 text-sm">
                      <SelectValue placeholder="Status" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Statuses</SelectItem>
                      <SelectItem value="PENDING">Pending</SelectItem>
                      <SelectItem value="APPROVED">Approved</SelectItem>
                      <SelectItem value="REJECTED">Rejected</SelectItem>
                      <SelectItem value="EXPIRED">Expired</SelectItem>
                      <SelectItem value="EXECUTING">Executing</SelectItem>
                      <SelectItem value="EXECUTED">Executed</SelectItem>
                      <SelectItem value="FAILED">Failed</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="approvals-user-filter" className="text-xs">User ID</Label>
                  <Input
                    id="approvals-user-filter"
                    placeholder="Any user"
                    value={userFilter}
                    onChange={(e) => setUserFilter(e.target.value)}
                    className="h-11 text-sm"
                    autoCapitalize="none"
                    autoCorrect="off"
                    spellCheck={false}
                    autoComplete="off"
                  />
                </div>
                <div className="flex items-end">
                  <Button
                    variant="ghost"
                    size="sm"
                    className="min-h-11 w-full sm:w-auto"
                    onClick={resetFilters}
                    disabled={!hasActiveFilters}
                  >
                    Reset filters
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {loadError ? (
            <div role="alert" className="mb-4 flex items-center gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300">
              <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{loadError}</span>
              <Button variant="outline" size="sm" className="ml-auto min-h-11" onClick={() => void handleManualRefresh()} disabled={refreshing}>Retry</Button>
            </div>
          ) : null}
          {loading ? (
            <div className="flex items-center justify-center min-h-[200px]" role="status">
              <div className="text-center">
                <Loader2 className="h-8 w-8 animate-spin mx-auto mb-3 text-muted-foreground" aria-hidden="true" />
                <p className="text-sm text-muted-foreground">Loading the approval queue...</p>
              </div>
            </div>
          ) : requests.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <ShieldCheck className="h-10 w-10 text-muted-foreground/40 mb-3" />
              <p className="text-sm text-muted-foreground">
                {loadError
                  ? 'Approval requests could not be loaded right now.'
                  : statusFilter === 'all' && !userFilter.trim()
                  ? 'No approval requests yet. New requests that need a decision will appear here.'
                  : statusFilter === DEFAULT_STATUS_FILTER && !userFilter.trim()
                  ? 'No pending approvals right now. New requests that need a decision will appear here.'
                  : 'No requests match your filters. Adjust or reset the filters and try again.'}
              </p>
              {loadError ? (
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-2 min-h-11"
                  onClick={() => void handleManualRefresh()}
                  disabled={refreshing}
                >
                  Retry
                </Button>
              ) : hasActiveFilters && (
                <Button variant="ghost" size="sm" className="mt-2 min-h-11" onClick={resetFilters}>
                  Reset filters
                </Button>
              )}
            </div>
          ) : (
            <>
              <div className="space-y-3 md:hidden">
                {requests.map((r) => {
                  const remainingMs = new Date(r.expiresAt).getTime() - Date.now()
                  const expiresSoon = r.status === 'PENDING' && remainingMs > 0 && remainingMs < 30 * 60 * 1000

                  return (
                    <Card key={r.id} className="border border-border/60 shadow-sm">
                      <CardContent className="space-y-4 p-4">
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0 space-y-1">
                            <button
                              type="button"
                              className="inline-flex min-h-11 items-center font-mono text-sm text-left rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 hover:underline"
                              onClick={() => setDetailDialog(r)}
                              aria-label={`View details for ${r.toolName}`}
                            >
                              {r.toolName}
                            </button>
                            <div className="flex flex-wrap items-center gap-2">
                              <button
                                type="button"
                                className="inline-flex min-h-11 items-center rounded-md border border-input bg-background px-3 py-2 font-mono text-xs transition-colors hover:bg-muted/50 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                                title="Click to filter by this user"
                                aria-label={`Filter approvals by user ${r.userId}`}
                                onClick={() => setUserFilter(r.userId)}
                              >
                                {r.userId}
                              </button>
                              <span className="text-xs text-muted-foreground">
                                {formatNullableText(r.serverId)}
                              </span>
                            </div>
                          </div>
                          <div className="shrink-0">{statusBadge(r.status)}</div>
                        </div>

                        <div className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
                          <div>
                            <p className="text-xs text-muted-foreground">Created</p>
                            <p>{formatTime(r.createdAt)}</p>
                          </div>
                          <div>
                            <p className="text-xs text-muted-foreground">Expires</p>
                            {(() => {
                              const { text: expiryText, urgent: expiryUrgent } = formatExpiryTime(r.expiresAt, r.status)
                              return (
                                <p className={expiresSoon || expiryUrgent ? 'font-medium text-amber-600 dark:text-amber-400' : 'text-muted-foreground'}>
                                  {expiryText}
                                </p>
                              )
                            })()}
                          </div>
                        </div>

                        <div>
                          <p className="text-xs text-muted-foreground">Why Required</p>
                          <p className="text-sm text-muted-foreground">{r.reason || '—'}</p>
                        </div>

                        <div className="flex flex-wrap gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            className="min-h-11 flex-1 sm:flex-none"
                            onClick={() => setDetailDialog(r)}
                          >
                            <Eye className="mr-1.5 h-3.5 w-3.5" />
                            Details
                          </Button>
                          {r.status === 'PENDING' && (
                            <>
                              <Button
                                size="sm"
                                className="min-h-11 flex-1 bg-emerald-600 text-white hover:bg-emerald-700"
                                onClick={() => openDecideDialog(r, 'APPROVED')}
                              >
                                <CheckCircle2 className="mr-1.5 h-3.5 w-3.5" />
                                Approve
                              </Button>
                              <Button
                                size="sm"
                                variant="destructive"
                                className="min-h-11 flex-1"
                                onClick={() => openDecideDialog(r, 'REJECTED')}
                              >
                                <XCircle className="mr-1.5 h-3.5 w-3.5" />
                                Reject
                              </Button>
                            </>
                          )}
                        </div>
                      </CardContent>
                    </Card>
                  )
                })}
              </div>

              <div className="hidden max-h-[min(65dvh,42rem)] overflow-auto md:block">
                <table className="min-w-[980px] w-full caption-bottom text-sm">
                  <TableHeader>
                    <TableRow>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Tool</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Server</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">User</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))] text-center">Status</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Created</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Expires</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Why Required</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))] text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {requests.map((r) => (
                      <TableRow key={r.id}>
                        <TableCell>
                          <button
                            type="button"
                            className="inline-flex min-h-11 items-center font-mono text-sm text-left rounded cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 hover:underline"
                            onClick={() => setDetailDialog(r)}
                            aria-label={`View details for ${r.toolName}`}
                          >
                            {r.toolName}
                          </button>
                        </TableCell>
                        <TableCell>
                          <span className="text-sm text-muted-foreground truncate max-w-[120px] block" title={r.serverId || undefined}>
                            {r.serverId || '—'}
                          </span>
                        </TableCell>
                        <TableCell>
                          <button
                            type="button"
                            className="inline-flex min-h-11 items-center rounded-md border border-input bg-background px-3 py-2 font-mono text-xs transition-colors hover:bg-muted/50 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                            title="Click to filter by this user"
                            aria-label={`Filter approvals by user ${r.userId}`}
                            onClick={() => setUserFilter(r.userId)}
                          >
                            {r.userId}
                          </button>
                        </TableCell>
                        <TableCell className="text-center">{statusBadge(r.status)}</TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {formatTime(r.createdAt)}
                        </TableCell>
                        {(() => {
                          const { text: expiryText, urgent: expiryUrgent } = formatExpiryTime(r.expiresAt, r.status)
                          const remainingMs = new Date(r.expiresAt).getTime() - Date.now()
                          const amberStyle = (r.status === 'PENDING' && remainingMs > 0 && remainingMs < 30 * 60 * 1000) || expiryUrgent
                          return (
                            <TableCell className={`text-sm ${amberStyle ? 'text-amber-600 dark:text-amber-400 font-medium' : 'text-muted-foreground'}`}>
                              {expiryText}
                            </TableCell>
                          )
                        })()}
                        <TableCell className="text-sm text-muted-foreground max-w-[180px]" title={r.reason || undefined}>
                          <span className="block truncate">
                            {r.reason || '—'}
                          </span>
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-1">
                            <TooltipProvider delayDuration={300}>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="min-h-11 px-3 text-xs"
                                    aria-label="View details"
                                    onClick={() => setDetailDialog(r)}
                                  >
                                    <Eye className="mr-1 h-3.5 w-3.5" />
                                    Details
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent>View details</TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                            {r.status === 'PENDING' && (
                              <>
                                <Button
                                  size="sm"
                                  className="min-h-11 bg-emerald-600 px-3 text-xs text-white hover:bg-emerald-700"
                                  aria-label="Approve"
                                  onClick={() => openDecideDialog(r, 'APPROVED')}
                                >
                                  <CheckCircle2 className="mr-1 h-3.5 w-3.5" />
                                  Approve
                                </Button>
                                <Button
                                  variant="destructive"
                                  size="sm"
                                  className="min-h-11 px-3 text-xs"
                                  aria-label="Reject"
                                  onClick={() => openDecideDialog(r, 'REJECTED')}
                                >
                                  <XCircle className="mr-1 h-3.5 w-3.5" />
                                  Reject
                                </Button>
                              </>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </table>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {!loading && requests.length > 0 && (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-xs text-muted-foreground">
            {formatDisplayNumber(requests.length)} requests
          </p>
          {hasMore && (
            <Button variant="outline" size="sm" className="min-h-11" onClick={handleLoadMore} disabled={loadingMore || refreshing}>
              {loadingMore ? 'Loading more approvals...' : 'Load more'}
            </Button>
          )}
        </div>
      )}

      {/* Detail Dialog */}
      <Dialog open={!!detailDialog} onOpenChange={(open) => { if (!open && !detailDeciding) { setDetailDialog(null); setDetailDecideReason(''); } }}>
        {detailDialog && (
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Approval Request Details</DialogTitle>
              <DialogDescription>
                Request for <strong>{detailDialog.toolName}</strong>
                {detailDialog.serverId?.trim() && (
                  <> on server <strong>{detailDialog.serverId}</strong></>
                )}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-2">
              <div className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
                <div>
                  <Label className="text-xs text-muted-foreground">Status</Label>
                  <div className="mt-1">{statusBadge(detailDialog.status)}</div>
                </div>
                <div>
                  <Label className="text-xs text-muted-foreground">User</Label>
                  <p className="mt-1 font-mono text-sm">{detailDialog.userId}</p>
                </div>
                <div>
                  <Label className="text-xs text-muted-foreground">Created</Label>
                  <p className="mt-1">{formatTime(detailDialog.createdAt)}</p>
                </div>
                {(() => {
                   const { text: expiryText, urgent: expiryUrgent } = formatExpiryTime(detailDialog.expiresAt, detailDialog.status)
                   return (
                     <div>
                       <Label className="text-xs text-muted-foreground">Expires</Label>
                       <p className={`mt-1 ${expiryUrgent ? 'font-medium text-amber-600 dark:text-amber-400' : ''}`}>{expiryText}</p>
                     </div>
                   )
                 })()}
                 {detailDialog.executedAt && (
                  <div>
                    <Label className="text-xs text-muted-foreground">Executed At</Label>
                    <p className="mt-1">{formatTime(detailDialog.executedAt)}</p>
                  </div>
                )}
                {detailDialog.decidedAt && (
                  <div>
                    <Label className="text-xs text-muted-foreground">Decided At</Label>
                    <p className="mt-1">{formatTime(detailDialog.decidedAt)}</p>
                  </div>
                )}
              </div>

              {detailDialog.reason && (
                <div>
                  <Label className="text-xs text-muted-foreground">Why Required</Label>
                  <p className="mt-1 text-sm">{detailDialog.reason}</p>
                </div>
              )}

              {detailDialog.decisionReason && (
                <div>
                  <Label className="text-xs text-muted-foreground">Decision Reason</Label>
                  <p className="mt-1 text-sm">{detailDialog.decisionReason}</p>
                </div>
              )}

              {(detailDialog.decidedByUserId || detailDialog.decisionChannel) && (
                <div className="grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <Label className="text-xs text-muted-foreground">Decided By</Label>
                    <p className="mt-1 font-mono text-xs">
                      {detailDialog.decidedByUserId || '—'}
                    </p>
                  </div>
                  <div>
                    <Label className="text-xs text-muted-foreground">Decision Source</Label>
                    <p className="mt-1 text-sm">
                      {channelLabel(detailDialog.decisionChannel)}
                      {detailDialog.decidedByRole != null
                        ? ` (${roleLabel(detailDialog.decidedByRole)})`
                        : ''}
                    </p>
                  </div>
                </div>
              )}

              {detailDialog.executionError && (
                <div>
                  <Label className="text-xs text-muted-foreground">Execution Error</Label>
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{detailDialog.executionError}</p>
                </div>
              )}

              {detailDialog.executionResultAvailable && (
                <div>
                  <Label className="text-xs text-muted-foreground">Execution Result</Label>
                  <p className="mt-1 text-sm">Execution result is available.</p>
                </div>
              )}

              {Object.keys(redactArgs(detailDialog.redactedArgs || {})).length > 0 && (
                <div>
                  <Label className="text-xs text-muted-foreground">Tool Arguments (Redacted)</Label>
                  <pre className="mt-1 text-xs bg-muted/50 p-3 rounded-md overflow-x-auto font-mono max-h-48 whitespace-pre-wrap break-all">
                    {JSON.stringify(redactArgs(detailDialog.redactedArgs || {}), null, 2)}
                  </pre>
                </div>
              )}

              <div className="text-sm">
                <div>
                  <Label className="text-xs text-muted-foreground">Policy Version</Label>
                  <p className="mt-1 font-mono text-xs">v{detailDialog.policyVersion}</p>
                </div>
              </div>
            </div>
            <DialogFooter className="flex-col items-stretch gap-3 sm:flex-col">
              {detailDialog.status === 'PENDING' && (
                <div className="space-y-3 border-t pt-3">
                  <div className="space-y-1.5">
                    <Label htmlFor="approval-detail-decision-reason">Decision reason (optional)</Label>
                    <Textarea
                      id="approval-detail-decision-reason"
                      placeholder="Add a reason for this decision..."
                      value={detailDecideReason}
                      onChange={(e) => setDetailDecideReason(e.target.value)}
                      rows={2}
                      disabled={detailDeciding}
                      className="text-sm resize-none"
                    />
                  </div>
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Button
                      size="sm"
                      className="min-h-11 w-full sm:w-auto bg-emerald-600 text-white hover:bg-emerald-700"
                      onClick={() => handleDetailDecide('APPROVED')}
                      disabled={detailDeciding}
                    >
                      {detailDeciding ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" /> : <CheckCircle2 className="mr-1.5 h-3.5 w-3.5" />}
                      {detailDeciding ? 'Approving...' : 'Approve'}
                    </Button>
                    <Button
                      size="sm"
                      variant="destructive"
                      className="min-h-11 w-full sm:w-auto"
                      onClick={() => handleDetailDecide('REJECTED')}
                      disabled={detailDeciding}
                    >
                      {detailDeciding ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" /> : <XCircle className="mr-1.5 h-3.5 w-3.5" />}
                      {detailDeciding ? 'Rejecting...' : 'Reject'}
                    </Button>
                  </div>
                </div>
              )}
              <div className="flex justify-stretch sm:justify-end">
                <Button variant="outline" className="min-h-11 w-full sm:w-auto" onClick={() => { setDetailDialog(null); setDetailDecideReason(''); }} disabled={detailDeciding}>
                  Close
                </Button>
              </div>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>

      {/* Decide Dialog */}
      <Dialog open={!!decideDialog} onOpenChange={(open) => !open && setDecideDialog(null)}>
        {decideDialog && (
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle className="font-mono text-base">
                {decideDialog.request.toolName}
              </DialogTitle>
              <DialogDescription>
                {decideDialog.decision === 'APPROVED'
                  ? `Allow this tool call to proceed?`
                  : `Deny this tool call?`}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-3 py-2">
              <div className="space-y-1.5">
                <Label>Decision reason (optional)</Label>
                <Textarea
                  placeholder="Add a reason for this decision..."
                  value={decideReason}
                  onChange={(e) => setDecideReason(e.target.value)}
                  ref={decideReasonRef}
                  rows={3}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" className="min-h-11" onClick={() => setDecideDialog(null)} disabled={deciding}>
                Cancel
              </Button>
              {decideDialog.decision === 'APPROVED' ? (
                <Button
                  className="min-h-11 bg-emerald-600 hover:bg-emerald-700 text-white"
                  onClick={handleDecide}
                  disabled={deciding}
                >
                  {deciding ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
                      Approving...
                    </>
                  ) : 'Approve'}
                </Button>
              ) : (
                <Button variant="destructive" className="min-h-11" onClick={handleDecide} disabled={deciding}>
                  {deciding ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
                      Rejecting...
                    </>
                  ) : 'Reject'}
                </Button>
              )}
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </div>
  );
}
