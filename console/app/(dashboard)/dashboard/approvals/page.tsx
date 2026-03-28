'use client';

import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
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
  ScrollableDialogContent,
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
import { useDebounce } from '@/hooks/use-debounce';
import { cn, formatDateTime, formatDisplayNumber, formatNullableText, formatRelativeMinutes } from '@/lib/utils';

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
        <Badge className="border-amber-200 bg-amber-100 text-amber-800 transition-colors hover:bg-amber-200 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-300 dark:hover:bg-amber-950/50">
          <Clock className="h-3 w-3 mr-1" />
          Pending
        </Badge>
      );
    case 'APPROVED':
      return (
        <Badge className="border-green-200 bg-green-100 text-green-800 transition-colors hover:bg-green-200 dark:border-green-800 dark:bg-green-950/30 dark:text-green-300 dark:hover:bg-green-950/50">
          <CheckCircle2 className="h-3 w-3 mr-1" />
          Approved
        </Badge>
      );
    case 'REJECTED':
      return (
        <Badge className="border-red-200 bg-red-100 text-red-800 transition-colors hover:bg-red-200 dark:border-red-800 dark:bg-red-950/30 dark:text-red-300 dark:hover:bg-red-950/50">
          <XCircle className="h-3 w-3 mr-1" />
          Rejected
        </Badge>
      );
    case 'EXPIRED':
      return (
        <Badge className="border-amber-200 bg-amber-100 text-amber-800 transition-colors hover:bg-amber-200 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-300 dark:hover:bg-amber-950/50">
          <AlertTriangle className="h-3 w-3 mr-1" />
          Expired
        </Badge>
      );
    case 'EXECUTING':
      return (
        <Badge className="border-blue-200 bg-blue-100 text-blue-800 transition-colors hover:bg-blue-200 dark:border-blue-800 dark:bg-blue-950/30 dark:text-blue-300 dark:hover:bg-blue-950/50">
          <Loader2 className="h-3 w-3 mr-1 animate-spin" />
          Executing
        </Badge>
      );
    case 'EXECUTED':
      return (
        <Badge className="border-sky-200 bg-sky-100 text-sky-800 transition-colors hover:bg-sky-200 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-300 dark:hover:bg-sky-950/50">
          <CheckCircle2 className="h-3 w-3 mr-1" />
          Executed
        </Badge>
      );
    case 'FAILED':
      return (
        <Badge className="border-red-200 bg-red-100 text-red-800 transition-colors hover:bg-red-200 dark:border-red-800 dark:bg-red-950/30 dark:text-red-300 dark:hover:bg-red-950/50">
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

function getRequestSortTimestamp(iso?: string | null): number {
  if (!iso) return Number.MAX_SAFE_INTEGER;
  const parsed = Date.parse(iso);
  return Number.isNaN(parsed) ? Number.MAX_SAFE_INTEGER : parsed;
}

function formatStatusLabel(status: string): string {
  return `${status.charAt(0)}${status.slice(1).toLowerCase()}`;
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
  const debouncedUserFilter = useDebounce(userFilter, 300);
  const [detailDialog, setDetailDialog] = useState<ApprovalRequest | null>(null);
  const [decideDialog, setDecideDialog] = useState<{
    request: ApprovalRequest;
    decision: 'APPROVED' | 'REJECTED';
  } | null>(null);
  const [decideReason, setDecideReason] = useState('');
  const [deciding, setDeciding] = useState(false);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const decideReasonRef = useRef<HTMLTextAreaElement>(null);
  const lastDialogTriggerRef = useRef<HTMLElement | null>(null);
  const [refreshFailed, setRefreshFailed] = useState(false);
  const [timeAgo, setTimeAgo] = useState('');
  const [loadedPages, setLoadedPages] = useState(1);
  const hasActiveFilters = statusFilter !== DEFAULT_STATUS_FILTER || userFilter.trim().length > 0;
  const isPendingOnlyEmptyState = statusFilter === DEFAULT_STATUS_FILTER && !userFilter.trim();
  const orderedRequests = useMemo(() => {
    const statusRank: Record<ApprovalRequest['status'], number> = {
      PENDING: 0,
      EXECUTING: 1,
      FAILED: 2,
      APPROVED: 3,
      REJECTED: 4,
      EXECUTED: 5,
      EXPIRED: 6,
    };

    return [...requests].sort((a, b) => {
      const statusDelta = (statusRank[a.status] ?? 99) - (statusRank[b.status] ?? 99);
      if (statusDelta !== 0) {
        return statusDelta;
      }

      if (a.status === 'PENDING' && b.status === 'PENDING') {
        return getRequestSortTimestamp(a.expiresAt) - getRequestSortTimestamp(b.expiresAt);
      }

      return getRequestSortTimestamp(b.createdAt) - getRequestSortTimestamp(a.createdAt);
    });
  }, [requests]);
  const visibleStatusCounts = useMemo(
    () =>
      orderedRequests.reduce<Record<string, number>>((counts, request) => {
        counts[request.status] = (counts[request.status] || 0) + 1;
        return counts;
      }, {}),
    [orderedRequests],
  );
  const expiringSoonCount = useMemo(
    () =>
      orderedRequests.filter((request) => {
        if (request.status !== 'PENDING') return false;
        const remainingMs = new Date(request.expiresAt).getTime() - Date.now();
        return remainingMs > 0 && remainingMs <= 30 * 60 * 1000;
      }).length,
    [orderedRequests],
  );
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
      const userId = options?.userId ?? debouncedUserFilter;

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
          const newCount = countData?.count || 0;
          setPendingCount(newCount);
          window.dispatchEvent(new CustomEvent('kimbap:pending-approvals-updated', { detail: { count: newCount } }));
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
            auth: 'Session expired or access revoked. Sign in again.',
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
    [statusFilter, debouncedUserFilter],
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
      userId: debouncedUserFilter,
    });

    refreshTimerRef.current = setInterval(() => {
      void fetchData({
        page: 1,
        pageSize: loadedPagesRef.current * BASE_PAGE_SIZE,
        status: statusFilter,
        userId: debouncedUserFilter,
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
  }, [fetchData, statusFilter, debouncedUserFilter]);

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

  const rememberDialogTrigger = (element?: HTMLElement | null) => {
    lastDialogTriggerRef.current =
      element ?? (document.activeElement instanceof HTMLElement ? document.activeElement : null);
  };

  const openDetailDialog = (request: ApprovalRequest, trigger?: HTMLElement | null) => {
    rememberDialogTrigger(trigger);
    setDetailDialog(request);
  };

  const openDecideDialog = (request: ApprovalRequest, decision: 'APPROVED' | 'REJECTED', trigger?: HTMLElement | null) => {
    rememberDialogTrigger(trigger);
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
          auth: 'Session expired or access revoked. Sign in again.',
          network: `Could not ${actionLabel} ${decideDialog.request.toolName}. Check your connection and retry.`,
          fallback: `Could not ${actionLabel} ${decideDialog.request.toolName} request.`
        })
      );
    } finally {
      setDeciding(false);
    }
  };

  const openDecisionFromDetail = (decision: 'APPROVED' | 'REJECTED') => {
    if (!detailDialog) return;

    setDecideReason('');
    setDecideDialog({ request: detailDialog, decision });
    setDetailDialog(null);
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
          <p className="text-sm leading-6 text-muted-foreground">
            Review tool requests that are waiting for a decision. After each decision, the queue refreshes so you can continue with the next request.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          {refreshFailed && <span className="text-xs text-amber-600 dark:text-amber-400">Last refresh failed</span>}
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
            Refresh
          </Button>
        </div>
      </div>

      {/* Pending Count Banner */}
      {pendingCount > 0 && (
        <Card className="border-amber-500/30 bg-amber-500/5 dark:border-amber-800/70 dark:bg-amber-950/20">
          <CardContent className="flex flex-col items-start gap-3 py-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex flex-wrap items-center gap-3">
              <Clock className="h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" />
              <span className="text-sm font-medium">
                {formatDisplayNumber(pendingCount)} pending approval{pendingCount !== 1 ? 's' : ''} awaiting review
              </span>
              {expiringSoonCount > 0 ? (
                <Badge
                  variant="outline"
                  className="border-amber-300 bg-background/80 text-amber-700 dark:border-amber-800 dark:bg-background dark:text-amber-300"
                >
                  {formatDisplayNumber(expiringSoonCount)} expiring soon
                </Badge>
              ) : null}
            </div>
            {statusFilter !== DEFAULT_STATUS_FILTER && (
              <Button
                variant="outline"
                size="sm"
                className="min-h-11 w-full shrink-0 sm:w-auto"
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
                <CardDescription className="flex flex-wrap items-center gap-2">
                  <span>
                    {formatDisplayNumber(orderedRequests.length)} request{orderedRequests.length !== 1 ? 's' : ''}
                    {statusFilter !== 'all' ? ` (${formatStatusLabel(statusFilter)})` : ''}
                  </span>
                  {Object.entries(visibleStatusCounts).map(([status, count]) => (
                    <Badge key={status} variant="outline" className="text-[11px]">
                      {formatStatusLabel(status)} {formatDisplayNumber(count)}
                    </Badge>
                  ))}
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
                     placeholder="e.g., user_123"
                    value={userFilter}
                    onChange={(e) => setUserFilter(e.target.value)}
                    className="h-11 text-sm"
                    autoCapitalize="none"
                    autoCorrect="off"
                     spellCheck={false}
                     autoComplete="off"
                   />
                   <p className="text-xs text-muted-foreground">Results update automatically while you type.</p>
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
            <div role="alert" className="mb-4 flex flex-col items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300 sm:flex-row sm:items-center">
              <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{loadError}</span>
              <Button variant="outline" size="sm" className="min-h-11 w-full sm:ml-auto sm:w-auto" onClick={() => void handleManualRefresh()} disabled={refreshing}>Retry</Button>
            </div>
          ) : null}
          {loading ? (
            <div className="flex items-center justify-center min-h-[200px]" role="status">
              <div className="text-center">
                <Loader2 className="h-8 w-8 animate-spin mx-auto mb-3 text-muted-foreground" aria-hidden="true" />
                <p className="text-sm text-muted-foreground">Loading the approval queue…</p>
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
                  : isPendingOnlyEmptyState
                  ? 'No pending approvals right now. View all requests to review recent decisions, or wait for the next request.'
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
              ) : isPendingOnlyEmptyState ? (
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-2 min-h-11"
                  onClick={() => setStatusFilter('all')}
                >
                  View all requests
                </Button>
              ) : hasActiveFilters ? (
                <Button variant="ghost" size="sm" className="mt-2 min-h-11" onClick={resetFilters}>
                  Reset filters
                </Button>
              ) : null}
            </div>
          ) : (
            <>
              <div className="space-y-3 md:hidden">
                {orderedRequests.map((r) => {
                  const remainingMs = new Date(r.expiresAt).getTime() - Date.now()
                  const expiresSoon = r.status === 'PENDING' && remainingMs > 0 && remainingMs < 30 * 60 * 1000

                  return (
                    <Card key={r.id} className="border border-border/60 shadow-sm">
                      <CardContent className="space-y-4 p-4">
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0 space-y-1">
                            <button
                              type="button"
                              className="inline-flex min-h-11 items-center rounded text-left font-mono text-sm transition-colors duration-200 hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                              onClick={(event) => openDetailDialog(r, event.currentTarget)}
                              aria-label={`View details for ${r.toolName}`}
                            >
                              {r.toolName}
                            </button>
                            <div className="flex flex-wrap items-center gap-2">
                              <button
                                type="button"
                                className={cn(
                                 'inline-flex min-h-11 items-center rounded-md border px-3 py-2 font-mono text-xs transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
                                 userFilter.trim() === r.userId
                                   ? 'border-primary bg-accent text-accent-foreground shadow-sm'
                                   : 'border-input bg-background hover:bg-muted/50'
                               )}
                                title="Click to filter by this user"
                                aria-label={`Filter approvals by user ${r.userId}`}
                                aria-pressed={userFilter.trim() === r.userId}
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
                          <div>
                            <p className="text-xs text-muted-foreground">Created</p>
                            <p>{formatTime(r.createdAt)}</p>
                          </div>
                        </div>

                        <div>
                          <p className="text-xs text-muted-foreground">Why Required</p>
                          <p className="line-clamp-2 text-sm text-muted-foreground" title={r.reason || undefined}>{r.reason || '—'}</p>
                        </div>

                        <div className="flex flex-wrap gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            className="min-h-11 flex-1 sm:flex-none"
                            onClick={(event) => openDetailDialog(r, event.currentTarget)}
                          >
                            <Eye className="mr-1.5 h-3.5 w-3.5" />
                            Details
                          </Button>
                          {r.status === 'PENDING' && (
                            <>
                              <Button
                                size="sm"
                                className="min-h-11 flex-1 bg-emerald-600 text-white hover:bg-emerald-700"
                                onClick={(event) => openDecideDialog(r, 'APPROVED', event.currentTarget)}
                              >
                                <CheckCircle2 className="mr-1.5 h-3.5 w-3.5" />
                                Approve
                              </Button>
                              <Button
                                size="sm"
                                variant="destructive"
                                className="min-h-11 flex-1"
                                onClick={(event) => openDecideDialog(r, 'REJECTED', event.currentTarget)}
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
                <table className="min-w-[1120px] w-full caption-bottom text-sm">
                  <TableHeader>
                    <TableRow>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Tool</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Server</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]" title="Click a user badge to filter by that user">User</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))] text-center">Status</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Expires</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Created</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Why Required</TableHead>
                      <TableHead scope="col" className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))] text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {orderedRequests.map((r) => (
                      <TableRow key={r.id}>
                        <TableCell>
                          <button
                             type="button"
                             className="inline-flex min-h-11 max-w-[260px] items-center truncate rounded text-left font-mono text-sm transition-colors duration-200 hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                             onClick={(event) => openDetailDialog(r, event.currentTarget)}
                             aria-label={`View details for ${r.toolName}`}
                             title={r.toolName}
                           >
                             {r.toolName}
                           </button>
                        </TableCell>
                        <TableCell>
                          <span className="block max-w-[180px] truncate text-sm text-muted-foreground" title={r.serverId || undefined}>
                            {r.serverId || '—'}
                          </span>
                        </TableCell>
                        <TableCell>
                          <button
                            type="button"
                            className={cn(
                             'inline-flex min-h-11 items-center rounded-md border px-3 py-2 font-mono text-xs transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
                             userFilter.trim() === r.userId
                               ? 'border-primary bg-accent text-accent-foreground shadow-sm'
                               : 'border-input bg-background hover:bg-muted/50'
                           )}
                            title="Click to filter by this user"
                            aria-label={`Filter approvals by user ${r.userId}`}
                            aria-pressed={userFilter.trim() === r.userId}
                             onClick={() => setUserFilter(r.userId)}
                          >
                            {r.userId}
                          </button>
                        </TableCell>
                        <TableCell className="text-center">{statusBadge(r.status)}</TableCell>
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
                        <TableCell className="text-sm text-muted-foreground">
                          {formatTime(r.createdAt)}
                        </TableCell>
                         <TableCell className="max-w-[320px] text-sm text-muted-foreground" title={r.reason || undefined}>
                           <span className="block line-clamp-2">
                             {r.reason || '—'}
                           </span>
                         </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              className="min-h-11 px-3 text-xs"
                              aria-label="View request details"
                              onClick={(event) => openDetailDialog(r, event.currentTarget)}
                            >
                              <Eye className="mr-1 h-3.5 w-3.5" />
                              Details
                            </Button>
                            {r.status === 'PENDING' && (
                              <>
                                <Button
                                  size="sm"
                                  className="min-h-11 bg-emerald-600 px-3 text-xs text-white hover:bg-emerald-700"
                                  aria-label="Approve"
                                  onClick={(event) => openDecideDialog(r, 'APPROVED', event.currentTarget)}
                                >
                                  <CheckCircle2 className="mr-1 h-3.5 w-3.5" />
                                  Approve
                                </Button>
                                <Button
                                  variant="destructive"
                                  size="sm"
                                  className="min-h-11 px-3 text-xs"
                                  aria-label="Reject"
                                  onClick={(event) => openDecideDialog(r, 'REJECTED', event.currentTarget)}
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

      {!loading && orderedRequests.length > 0 && (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-xs text-muted-foreground">
            {formatDisplayNumber(orderedRequests.length)} requests
          </p>
          {hasMore && (
            <Button variant="outline" size="sm" className="min-h-11" onClick={handleLoadMore} disabled={loadingMore || refreshing}>
              {loadingMore ? (<><Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" aria-hidden="true" />Load more</>) : 'Load more'}
            </Button>
          )}
        </div>
      )}

      {/* Detail Dialog */}
      <Dialog open={!!detailDialog} onOpenChange={(open) => { if (!open) { setDetailDialog(null); } }}>
        {detailDialog && (
          <ScrollableDialogContent
            className="max-w-2xl"
            onCloseAutoFocus={(event) => {
              event.preventDefault()
              lastDialogTriggerRef.current?.focus()
            }}
          >
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2 text-base">
                 Review approval request
                 <span className="font-sans text-sm font-normal not-italic">{statusBadge(detailDialog.status)}</span>
                </DialogTitle>
                <DialogDescription>
                  {detailDialog.toolName}{detailDialog.serverId?.trim() ? ` on server ${detailDialog.serverId}` : ''}
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
                {(() => {
                   const { text: expiryText, urgent: expiryUrgent } = formatExpiryTime(detailDialog.expiresAt, detailDialog.status)
                   return (
                     <div>
                       <Label className="text-xs text-muted-foreground">Expires</Label>
                       <p className={`mt-1 ${expiryUrgent ? 'font-medium text-amber-600 dark:text-amber-400' : ''}`}>{expiryText}</p>
                     </div>
                   )
                 })()}
                <div>
                  <Label className="text-xs text-muted-foreground">Created</Label>
                  <p className="mt-1">{formatTime(detailDialog.createdAt)}</p>
                </div>
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
                  <Label className="text-xs text-muted-foreground">Decision note</Label>
                  <p className="mt-1 text-sm">{detailDialog.decisionReason}</p>
                </div>
              )}

              {(detailDialog.decidedByUserId || detailDialog.decisionChannel) && (
                <div className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
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

              <div className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
                <div>
                   <Label className="text-xs text-muted-foreground">Policy Version</Label>
                   <p className="mt-1 font-mono text-xs">v{detailDialog.policyVersion}</p>
                 </div>
                 {detailDialog.uniformRequestId && (
                   <div>
                     <Label className="text-xs text-muted-foreground">Correlation ID</Label>
                     <p className="mt-1 font-mono text-xs break-all">{detailDialog.uniformRequestId}</p>
                   </div>
                 )}
               </div>
            </div>
            <DialogFooter className="flex-col items-stretch gap-3 sm:flex-col">
              {detailDialog.status === 'PENDING' && (
                <div className="space-y-3 border-t pt-3">
                  <p className="text-sm text-muted-foreground">
                    Approve or reject in the next step. You can add an optional note before you confirm.
                  </p>
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Button
                      size="sm"
                      className="min-h-11 w-full sm:w-auto bg-emerald-600 text-white hover:bg-emerald-700"
                      onClick={() => openDecisionFromDetail('APPROVED')}
                    >
                      <CheckCircle2 className="mr-1.5 h-3.5 w-3.5" />
                      Approve…
                    </Button>
                    <Button
                      size="sm"
                      variant="destructive"
                      className="min-h-11 w-full sm:w-auto"
                      onClick={() => openDecisionFromDetail('REJECTED')}
                    >
                      <XCircle className="mr-1.5 h-3.5 w-3.5" />
                      Reject…
                    </Button>
                  </div>
                </div>
              )}
              <div className="flex justify-stretch sm:justify-end">
                <Button variant="outline" className="min-h-11 w-full sm:w-auto" onClick={() => setDetailDialog(null)}>
                  Close
                </Button>
              </div>
            </DialogFooter>
          </ScrollableDialogContent>
        )}
      </Dialog>

      {/* Decide Dialog */}
      <Dialog open={!!decideDialog} onOpenChange={(open) => !open && setDecideDialog(null)}>
        {decideDialog && (
          <DialogContent
            className="max-w-lg"
            onCloseAutoFocus={(event) => {
              event.preventDefault()
              lastDialogTriggerRef.current?.focus()
            }}
          >
            <DialogHeader>
              <DialogTitle className="text-base">
                {decideDialog.decision === 'APPROVED' ? 'Approve request' : 'Reject request'}
              </DialogTitle>
              <DialogDescription>
                {decideDialog.decision === 'APPROVED'
                  ? `Allow ${decideDialog.request.toolName} to proceed?`
                  : `Deny ${decideDialog.request.toolName}?`}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-3 py-2">
              {decideDialog.request.reason?.trim() && (
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Why approval is required</Label>
                  <p className="text-sm">{decideDialog.request.reason}</p>
                </div>
              )}
              {Object.keys(redactArgs(decideDialog.request.redactedArgs || {})).length > 0 && (
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Tool arguments</Label>
                  <pre className="max-h-36 overflow-auto rounded-md bg-muted/50 p-2 font-mono text-xs whitespace-pre-wrap break-all">
                    {JSON.stringify(redactArgs(decideDialog.request.redactedArgs || {}), null, 2)}
                  </pre>
                </div>
              )}
              <div className="space-y-1.5">
                <Label>Add note with this decision (optional)</Label>
                <Textarea
                  placeholder={
                    decideDialog.decision === 'APPROVED'
                      ? 'e.g., Approved for this incident'
                      : 'e.g., Not authorized for this request'
                  }
                  value={decideReason}
                  onChange={(e) => setDecideReason(e.target.value)}
                  ref={decideReasonRef}
                  rows={2}
                  disabled={deciding}
                  className="text-sm resize-none"
                />
                {decideReason.trim().length > 0 && (
                  <p className="text-[11px] text-muted-foreground">{decideReason.trim().length} characters</p>
                )}
                <p className="text-xs text-muted-foreground">
                  This note is saved with the request and shown in the detail view.
                </p>
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
                      Approve
                    </>
                  ) : 'Approve'}
                </Button>
              ) : (
                <Button variant="destructive" className="min-h-11" onClick={handleDecide} disabled={deciding}>
                  {deciding ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
                      Reject
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
