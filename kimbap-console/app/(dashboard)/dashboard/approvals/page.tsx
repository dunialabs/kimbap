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
        <Badge variant="secondary">
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
        <Badge className="bg-orange-100 text-orange-800 border-orange-200 dark:bg-orange-900 dark:text-orange-300 dark:border-orange-800 hover:bg-orange-200">
          <AlertTriangle className="h-3 w-3 mr-1" />
          Failed
        </Badge>
      );
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

function formatTime(iso: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return iso;
  }
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
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
      return String(role ?? '');
  }
}

function channelLabel(channel: string | null | undefined): string {
  switch (channel) {
    case 'admin_api': return 'Admin API';
    case 'socket': return 'WebSocket';
    default: return channel || '—';
  }
}

// ─── Main Page ──────────────────────────────────────────

export default function ApprovalsPage() {
  const [requests, setRequests] = useState<ApprovalRequest[]>([]);
  const [pendingCount, setPendingCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
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
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [refreshFailed, setRefreshFailed] = useState(false);
  const [timeAgo, setTimeAgo] = useState('');
  const [loadedPages, setLoadedPages] = useState(1);
  const refreshTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const tickRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const loadedPagesRef = useRef(1);
  const fetchVersionRef = useRef(0);

  const isInitialLoad = useRef(true);

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
        const resolvedPages = append ? page : Math.max(1, Math.ceil(pageSize / BASE_PAGE_SIZE));
        loadedPagesRef.current = resolvedPages;
        setLoadedPages(resolvedPages);
        setLastUpdated(new Date());
        setTimeAgo('just now');
        setRefreshFailed(false);
      } catch {
        if (fetchVersion !== fetchVersionRef.current) {
          return;
        }
        if (isInitialLoad.current) toast.error('Could not load approval requests');
        setRefreshFailed(true);
      } finally {
        if (fetchVersion !== fetchVersionRef.current) {
          return;
        }
        setLoading(false);
        setLoadingMore(false);
        isInitialLoad.current = false;
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
        if (secs < 30) setTimeAgo('just now');
        else if (secs < 60) setTimeAgo('less than a minute ago');
        else setTimeAgo(`${Math.floor(secs / 60)} min ago`);
        return prev;
      });
    }, 10000);
    return () => {
      if (refreshTimerRef.current) clearInterval(refreshTimerRef.current);
      if (tickRef.current) clearInterval(tickRef.current);
    };
  }, [fetchData, statusFilter, userFilter]);

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

  const handleDecide = async () => {
    if (!decideDialog) return;
    setDeciding(true);
    try {
      await api.approvals.decide({
        id: decideDialog.request.id,
        decision: decideDialog.decision,
        reason: decideReason.trim() || undefined,
      });
      toast.success(`Request ${decideDialog.decision.toLowerCase()}`);
      setDecideDialog(null);
      fetchData({
        page: 1,
        pageSize: loadedPagesRef.current * BASE_PAGE_SIZE,
      });
    } catch {
      toast.error('Could not submit decision');
    } finally {
      setDeciding(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[30px] font-bold tracking-tight flex items-center gap-2">
            <ShieldCheck className="h-6 w-6" />
            Approvals
          </h1>
          <p className="text-base text-muted-foreground">
            Review tool requests that are waiting for a decision.
          </p>
        </div>
        <div className="flex items-center gap-3">
          {refreshFailed && <span className="text-xs text-amber-600">Last refresh failed</span>}
          {!refreshFailed && lastUpdated && (
            <span className="text-xs text-muted-foreground">Updated {timeAgo || 'just now'}</span>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() =>
              fetchData({
                page: 1,
                pageSize: loadedPagesRef.current * BASE_PAGE_SIZE,
              })
            }
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Pending Count Banner */}
      {pendingCount > 0 && (
        <Card className="border-amber-500/30 bg-amber-500/5">
          <CardContent className="flex items-center gap-3 py-3">
            <Clock className="h-5 w-5 text-amber-600" />
            <span className="text-sm font-medium">
              {pendingCount.toLocaleString()} pending approval{pendingCount !== 1 ? 's' : ''} awaiting review
            </span>
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
                  {requests.length.toLocaleString()} request{requests.length !== 1 ? 's' : ''}
                  {statusFilter !== 'all' ? ` (${statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1).toLowerCase()})` : ''}
                </CardDescription>
              </div>
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1.5">
                <Filter className="h-3.5 w-3.5 text-muted-foreground" />
                <Select
                  value={statusFilter}
                  onValueChange={(v) => setStatusFilter(v as StatusFilter)}
                >
                  <SelectTrigger className="h-8 w-[130px] text-sm" aria-label="Filter by status">
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
              <Input
                placeholder="Filter by user ID"
                value={userFilter}
                onChange={(e) => setUserFilter(e.target.value)}
                className="h-8 w-[160px] text-sm"
                aria-label="Filter by user"
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center justify-center min-h-[200px]" role="status">
              <div className="text-center">
                <Loader2 className="h-8 w-8 animate-spin mx-auto mb-3 text-muted-foreground" aria-hidden="true" />
                <p className="text-sm text-muted-foreground">Loading approval requests...</p>
              </div>
            </div>
          ) : requests.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <ShieldCheck className="h-10 w-10 text-muted-foreground/40 mb-3" />
              <p className="text-sm text-muted-foreground">
                {statusFilter === 'all' && !userFilter.trim()
                  ? 'No approval requests right now'
                  : statusFilter === DEFAULT_STATUS_FILTER && !userFilter.trim()
                  ? 'No pending approvals right now.'
                  : 'No requests match your filters'}
              </p>
            </div>
          ) : (
            <div className="max-h-[min(65dvh,42rem)] overflow-auto">
              <table className="w-full caption-bottom text-sm">
                <TableHeader>
                  <TableRow>
                    <TableHead className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Tool</TableHead>
                    <TableHead className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Server</TableHead>
                    <TableHead className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">User</TableHead>
                    <TableHead className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))] text-center">Status</TableHead>
                    <TableHead className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Created</TableHead>
                    <TableHead className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Expires</TableHead>
                     <TableHead className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))]">Policy reason</TableHead>
                    <TableHead className="sticky top-0 z-10 bg-background shadow-[0_1px_0_0_hsl(var(--border))] text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {requests.map((r) => (
                    <TableRow key={r.id}>
                      <TableCell>
                        <button
                          type="button"
                          className="font-mono text-sm text-left rounded cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 hover:underline"
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
                        <Badge variant="outline" className="font-mono text-xs">
                          {r.userId}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-center">{statusBadge(r.status)}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatTime(r.createdAt)}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatTime(r.expiresAt)}
                      </TableCell>
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
                                  size="icon"
                                  className="h-8 w-8"
                                  aria-label="View details"
                                  onClick={() => setDetailDialog(r)}
                                >
                                  <Eye className="h-3.5 w-3.5" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>View details</TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                          {r.status === 'PENDING' && (
                            <>
                              <TooltipProvider delayDuration={300}>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <Button
                                      variant="ghost"
                                      size="icon"
                                      className="h-8 w-8 text-emerald-600 hover:text-emerald-600 hover:bg-green-100 dark:hover:bg-green-900"
                                      aria-label="Approve"
                                      onClick={() => openDecideDialog(r, 'APPROVED')}
                                    >
                                      <CheckCircle2 className="h-3.5 w-3.5" />
                                    </Button>
                                  </TooltipTrigger>
                                  <TooltipContent>Approve</TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                              <TooltipProvider delayDuration={300}>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <Button
                                      variant="ghost"
                                      size="icon"
                                      className="h-8 w-8 text-red-600 hover:text-red-600 hover:bg-red-100 dark:hover:bg-red-900"
                                      aria-label="Reject"
                                      onClick={() => openDecideDialog(r, 'REJECTED')}
                                    >
                                      <XCircle className="h-3.5 w-3.5" />
                                    </Button>
                                  </TooltipTrigger>
                                  <TooltipContent>Reject</TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            </>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {!loading && requests.length > 0 && (
        <div className="flex items-center justify-between">
          <p className="text-xs text-muted-foreground">
            {requests.length.toLocaleString()} requests
          </p>
          {hasMore && (
            <Button variant="outline" size="sm" onClick={handleLoadMore} disabled={loadingMore}>
              {loadingMore ? 'Loading...' : 'Load more'}
            </Button>
          )}
        </div>
      )}

      {/* Detail Dialog */}
      <Dialog open={!!detailDialog} onOpenChange={(open) => !open && setDetailDialog(null)}>
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
              <div className="grid grid-cols-2 gap-3 text-sm">
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
                <div>
                  <Label className="text-xs text-muted-foreground">Expires</Label>
                  <p className="mt-1">{formatTime(detailDialog.expiresAt)}</p>
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
                  <Label className="text-xs text-muted-foreground">Policy Reason</Label>
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
                  <p className="mt-1 text-sm text-red-600">{detailDialog.executionError}</p>
                </div>
              )}

              {detailDialog.executionResultAvailable && (
                <div>
                  <Label className="text-xs text-muted-foreground">Execution Result</Label>
                  <p className="mt-1 text-sm">Execution result is available.</p>
                </div>
              )}

              <div>
                <Label className="text-xs text-muted-foreground">Tool Arguments (Redacted)</Label>
                <pre className="mt-1 text-xs bg-muted/50 p-3 rounded-md overflow-x-auto font-mono max-h-48 whitespace-pre-wrap break-all">
                  {JSON.stringify(redactArgs(detailDialog.redactedArgs || {}), null, 2)}
                </pre>
              </div>

              <div className="text-sm">
                <div>
                  <Label className="text-xs text-muted-foreground">Policy Version</Label>
                  <p className="mt-1 font-mono text-xs">{detailDialog.policyVersion}</p>
                </div>
              </div>
            </div>
            <DialogFooter>
              {detailDialog.status === 'PENDING' && (
                <div className="flex gap-2 mr-auto">
                  <Button
                    size="sm"
                    className="bg-emerald-600 hover:bg-emerald-700 text-white"
                    onClick={() => {
                      setDetailDialog(null);
                      openDecideDialog(detailDialog, 'APPROVED');
                    }}
                  >
                    <CheckCircle2 className="h-3.5 w-3.5 mr-1" />
                    Approve
                  </Button>
                  <Button
                    size="sm"
                    variant="destructive"
                    onClick={() => {
                      setDetailDialog(null);
                      openDecideDialog(detailDialog, 'REJECTED');
                    }}
                  >
                    <XCircle className="h-3.5 w-3.5 mr-1" />
                    Reject
                  </Button>
                </div>
              )}
              <Button variant="outline" onClick={() => setDetailDialog(null)}>
                Close
              </Button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>

      {/* Decide Dialog */}
      <Dialog open={!!decideDialog} onOpenChange={(open) => !open && setDecideDialog(null)}>
        {decideDialog && (
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>
                {decideDialog.decision === 'APPROVED' ? 'Approve' : 'Reject'} Request
              </DialogTitle>
              <DialogDescription>
                {decideDialog.decision === 'APPROVED'
                  ? `Allow ${decideDialog.request.toolName} to proceed for user ${decideDialog.request.userId}.`
                  : `Deny ${decideDialog.request.toolName} for user ${decideDialog.request.userId}.`}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-3 py-2">
              <div className="space-y-1.5">
                <Label>Reason (optional)</Label>
                <Textarea
                  placeholder="Add a reason for this decision..."
                  value={decideReason}
                  onChange={(e) => setDecideReason(e.target.value)}
                  rows={3}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setDecideDialog(null)}>
                Cancel
              </Button>
              {decideDialog.decision === 'APPROVED' ? (
                <Button
                  className="bg-emerald-600 hover:bg-emerald-700 text-white"
                  onClick={handleDecide}
                  disabled={deciding}
                >
                  {deciding ? 'Approving...' : 'Approve'}
                </Button>
              ) : (
                <Button variant="destructive" onClick={handleDecide} disabled={deciding}>
                  {deciding ? 'Rejecting...' : 'Reject'}
                </Button>
              )}
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </div>
  );
}
