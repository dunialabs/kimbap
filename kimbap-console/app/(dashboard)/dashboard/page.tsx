'use client';

import {
  Activity,
  AlertTriangle,
  ArrowRight,
  BarChart3,
  CheckCircle2,
  ClipboardList,
  Plug,
  ScrollText,
  TrendingDown,
  TrendingUp,
  UserCheck,
  XCircle,
  Zap,
} from 'lucide-react';
import Link from 'next/link';
import { useEffect, useState } from 'react';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

interface AuditEvent {
  id: string;
  action: string;
  actor: string;
  timestamp: string;
  status: 'success' | 'warning' | 'error';
}

interface IntegrationStatus {
  name: string;
  status: string;
}

export default function DashboardPage() {
  const [isLoading, setIsLoading] = useState(true);
  const [pendingApprovals, setPendingApprovals] = useState<number | null>(null);
  const [serviceHealth, setServiceHealth] = useState<string | null>(null);
  const [recentErrors, setRecentErrors] = useState<number | null>(null);
  const [integrationSummary, setIntegrationSummary] = useState<string | null>(null);
  const [integrations, setIntegrations] = useState<IntegrationStatus[]>([]);
  const [auditEvents, setAuditEvents] = useState<AuditEvent[]>([]);
  const [usageHighlights, setUsageHighlights] = useState<{
    totalRequests: number | null;
    changePercent: number | null;
    avgResponseTime: number | null;
    responseTimeChange: number | null;
    toolsInUse: number | null;
    mostActiveTool: string | null;
  }>({
    totalRequests: null,
    changePercent: null,
    avgResponseTime: null,
    responseTimeChange: null,
    toolsInUse: null,
    mostActiveTool: null,
  });

  useEffect(() => {
    const loadDashboard = async () => {
      try {
        const { api } = await import('@/lib/api-client');

        const results = await Promise.allSettled([
          api.approvals.countPending(),
          api.servers.getDashboardOverview(),
          api.logs.getStatistics({ timeRange: '24h' }),
          api.audit.getRecentActivity({ limit: 5, timeRange: '24h' }),
          api.usage.getOverviewSummary({ timeRange: 1 }),
        ]);

        if (results[0].status === 'fulfilled' && results[0].value.data?.common?.code === 0) {
          const countData = results[0].value.data?.data || results[0].value.data;
          setPendingApprovals(countData?.count ?? 0);
        }

        if (results[1].status === 'fulfilled' && results[1].value.data?.common?.code === 0) {
          const overview = results[1].value.data?.data || results[1].value.data;
          const rawHealth = overview?.health ?? overview?.status ?? null;
          setServiceHealth(typeof rawHealth === 'string' ? rawHealth.toLowerCase() : rawHealth);
          const connectors =
            overview?.connectors || overview?.integrations || overview?.servers || [];
          if (Array.isArray(connectors) && connectors.length > 0) {
            const connected = connectors.filter((c: { status?: string; enabled?: boolean }) =>
              c.status ? c.status === 'connected' || c.status === 'healthy' : c.enabled,
            ).length;
            setIntegrationSummary(`${connected}/${connectors.length}`);
            setIntegrations(
              connectors
                .slice(0, 5)
                .map(
                  (c: {
                    name?: string;
                    serverName?: string;
                    toolId?: string;
                    status?: string;
                    enabled?: boolean;
                  }) => ({
                    name: c.name || c.serverName || c.toolId || 'Unknown',
                    status: c.status || (c.enabled ? 'connected' : 'disconnected'),
                  }),
                ),
            );
          } else if (overview?.totalServers != null) {
            const total = overview.totalServers;
            const active = overview.activeServers ?? overview.connectedServers ?? total;
            setIntegrationSummary(`${active}/${total}`);
          }
        }

        if (results[2].status === 'fulfilled' && results[2].value.data?.common?.code === 0) {
          const logStats = results[2].value.data?.data || results[2].value.data;
          setRecentErrors(
            logStats?.errorCount ?? logStats?.errors ?? logStats?.byLevel?.ERROR ?? null,
          );
        }

        if (results[3].status === 'fulfilled' && results[3].value.data?.common?.code === 0) {
          const auditData = results[3].value.data?.data || results[3].value.data;
          const logs = auditData?.logs || [];
          setAuditEvents(
            logs
              .slice(0, 5)
              .map(
                (log: {
                  id: string;
                  details?: { toolName?: string };
                  message?: string;
                  userId?: string;
                  timestamp?: string;
                  level?: string;
                }) => {
                  const ts = log.timestamp ?? '';
                  const iso = ts.includes('T') ? ts : ts.replace(' ', 'T');
                  const parsedTime = new Date(
                    iso.endsWith('Z') || /[+-]\d{2}:?\d{2}$/.test(iso) ? iso : iso + 'Z',
                  ).getTime();
                  let relTime = '—';
                  const diffMs = Date.now() - parsedTime;
                  if (!Number.isNaN(parsedTime) && diffMs >= 0) {
                    const diffMin = Math.floor(diffMs / 60000);
                    relTime = 'Just now';
                    if (diffMin >= 1440) relTime = `${Math.floor(diffMin / 1440)}d ago`;
                    else if (diffMin >= 60) relTime = `${Math.floor(diffMin / 60)}h ago`;
                    else if (diffMin >= 1) relTime = `${diffMin}m ago`;
                  }
                  return {
                    id: log.id,
                    action: log.details?.toolName || log.message || 'Unknown action',
                    actor: log.userId || 'system',
                    timestamp: relTime,
                    status:
                      log.level === 'ERROR'
                        ? ('error' as const)
                        : log.level === 'WARN'
                          ? ('warning' as const)
                          : ('success' as const),
                  };
                },
              ),
          );
        }

        if (results[4].status === 'fulfilled' && results[4].value.data?.common?.code === 0) {
          const usage = results[4].value.data?.data || results[4].value.data;
          setUsageHighlights({
            totalRequests: usage?.totalRequests24h ?? null,
            changePercent: usage?.requestsChangePercent ?? null,
            avgResponseTime: usage?.avgResponseTime ?? null,
            responseTimeChange: usage?.responseTimeChange ?? null,
            toolsInUse: usage?.toolsInUse ?? null,
            mostActiveTool: usage?.mostActiveToolName || null,
          });
        }
      } catch {
      } finally {
        setIsLoading(false);
      }
    };

    loadDashboard();
  }, []);

  const statusIcon = (status: AuditEvent['status']) => {
    switch (status) {
      case 'success':
        return <div className="w-2 h-2 rounded-full bg-emerald-500" />;
      case 'warning':
        return <div className="w-2 h-2 rounded-full bg-amber-500" />;
      case 'error':
        return <div className="w-2 h-2 rounded-full bg-red-500" />;
    }
  };

  const healthBadge = (status: string) => {
    const normalized = status.toLowerCase();
    if (normalized === 'connected' || normalized === 'healthy' || normalized === 'running') {
      return (
        <Badge
          variant="outline"
          className="bg-emerald-500/10 text-emerald-600 border-emerald-500/20"
        >
          <CheckCircle2 className="h-3 w-3 mr-1" />
          Connected
        </Badge>
      );
    }
    if (normalized === 'degraded' || normalized === 'warning') {
      return (
        <Badge variant="outline" className="bg-amber-500/10 text-amber-600 border-amber-500/20">
          <AlertTriangle className="h-3 w-3 mr-1" />
          Degraded
        </Badge>
      );
    }
    return (
      <Badge variant="outline" className="bg-red-500/10 text-red-600 border-red-500/20">
        <XCircle className="h-3 w-3 mr-1" />
        Disconnected
      </Badge>
    );
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div aria-live="polite" className="text-center">
          <div
            className="w-8 h-8 border-2 border-muted-foreground/30 border-t-foreground rounded-full animate-spin mx-auto mb-4"
            aria-hidden="true"
          />
          <h3 className="text-lg font-semibold mb-2">Loading Dashboard</h3>
          <p className="text-muted-foreground">Loading runtime data…</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-[30px] font-bold tracking-tight">Overview</h1>
        <p className="text-base text-muted-foreground">
          Operations overview — health, approvals, audit, and integration status.
        </p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Link href="/dashboard/stats" className="block group">
          <Card className="h-full transition-colors group-hover:bg-muted/50">
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <Activity className="h-5 w-5 text-muted-foreground" />
                <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold flex items-center gap-2">
                {serviceHealth == null ? (
                  '—'
                ) : (
                  <>
                    <div
                      className={`w-2.5 h-2.5 rounded-full ${
                        serviceHealth === 'healthy' || serviceHealth === 'running'
                          ? 'bg-emerald-500'
                          : serviceHealth === 'degraded'
                            ? 'bg-amber-500'
                            : 'bg-red-500'
                      }`}
                    />
                    <span className="capitalize">{serviceHealth}</span>
                  </>
                )}
              </div>
              <p className="text-sm text-muted-foreground">Service Health</p>
            </CardContent>
          </Card>
        </Link>

        <Link href="/dashboard/approvals" className="block group">
          <Card
            className={`h-full transition-colors group-hover:bg-muted/50 ${pendingApprovals && pendingApprovals > 0 ? 'border-amber-500/40' : ''}`}
          >
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <UserCheck className="h-5 w-5 text-muted-foreground" />
                <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {pendingApprovals == null ? '—' : pendingApprovals}
              </div>
              <p className="text-sm text-muted-foreground">Pending Approvals</p>
            </CardContent>
          </Card>
        </Link>

        <Link href="/dashboard/logs" className="block group">
          <Card
            className={`h-full transition-colors group-hover:bg-muted/50 ${recentErrors && recentErrors > 0 ? 'border-red-500/40' : ''}`}
          >
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <AlertTriangle className="h-5 w-5 text-muted-foreground" />
                <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{recentErrors == null ? '—' : recentErrors}</div>
              <p className="text-sm text-muted-foreground">Recent Errors (24h)</p>
            </CardContent>
          </Card>
        </Link>

        <Link href="/dashboard/integrations" className="block group">
          <Card className="h-full transition-colors group-hover:bg-muted/50">
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <Plug className="h-5 w-5 text-muted-foreground" />
                <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{integrationSummary ?? '—'}</div>
              <p className="text-sm text-muted-foreground">Integrations</p>
            </CardContent>
          </Card>
        </Link>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <Link
          href="/dashboard/approvals"
          className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
        >
          <Card className="w-full flex items-center gap-2 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
            <UserCheck className="h-4 w-4" />
            <span>Approvals</span>
          </Card>
        </Link>
        <Link
          href="/dashboard/audit"
          className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
        >
          <Card className="w-full flex items-center gap-2 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
            <ClipboardList className="h-4 w-4" />
            <span>Audit</span>
          </Card>
        </Link>
        <Link
          href="/dashboard/logs"
          className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
        >
          <Card className="w-full flex items-center gap-2 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
            <ScrollText className="h-4 w-4" />
            <span>Logs</span>
          </Card>
        </Link>
        <Link
          href="/dashboard/stats"
          className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
        >
          <Card className="w-full flex items-center gap-2 justify-center h-[44px] cursor-pointer hover:bg-muted/50 transition-colors">
            <BarChart3 className="h-4 w-4" />
            <span>Stats</span>
          </Card>
        </Link>
      </div>

      {(usageHighlights.totalRequests != null ||
        usageHighlights.avgResponseTime != null ||
        usageHighlights.toolsInUse != null ||
        usageHighlights.mostActiveTool != null) && (
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="text-base">Usage Highlights</CardTitle>
                <CardDescription>Last 24 hours</CardDescription>
              </div>
              <Link
                href="/dashboard/stats"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                View details →
              </Link>
            </div>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              {usageHighlights.totalRequests != null && (
                <div className="space-y-1">
                  <p className="text-2xl font-bold">
                    {usageHighlights.totalRequests.toLocaleString()}
                  </p>
                  <div className="flex items-center gap-1">
                    <p className="text-xs text-muted-foreground">Requests</p>
                    {usageHighlights.changePercent != null &&
                      usageHighlights.changePercent !== 0 && (
                        <span
                          className={`text-xs flex items-center gap-0.5 ${usageHighlights.changePercent > 0 ? 'text-emerald-600' : 'text-red-600'}`}
                        >
                          {usageHighlights.changePercent > 0 ? (
                            <TrendingUp className="h-3 w-3" />
                          ) : (
                            <TrendingDown className="h-3 w-3" />
                          )}
                          {Math.abs(usageHighlights.changePercent).toFixed(0)}%
                        </span>
                      )}
                  </div>
                </div>
              )}
              {usageHighlights.avgResponseTime != null && (
                <div className="space-y-1">
                  <p className="text-2xl font-bold">
                    {usageHighlights.avgResponseTime < 1000
                      ? `${Math.round(usageHighlights.avgResponseTime)}ms`
                      : `${(usageHighlights.avgResponseTime / 1000).toFixed(1)}s`}
                  </p>
                  <div className="flex items-center gap-1">
                    <p className="text-xs text-muted-foreground">Avg Response</p>
                    {usageHighlights.responseTimeChange != null &&
                      usageHighlights.responseTimeChange !== 0 && (
                        <span
                          className={`text-xs flex items-center gap-0.5 ${usageHighlights.responseTimeChange < 0 ? 'text-emerald-600' : 'text-red-600'}`}
                        >
                          {usageHighlights.responseTimeChange < 0 ? (
                            <TrendingDown className="h-3 w-3" />
                          ) : (
                            <TrendingUp className="h-3 w-3" />
                          )}
                          {Math.abs(usageHighlights.responseTimeChange) < 1000
                            ? `${Math.abs(usageHighlights.responseTimeChange)}ms`
                            : `${(Math.abs(usageHighlights.responseTimeChange) / 1000).toFixed(1)}s`}
                        </span>
                      )}
                  </div>
                </div>
              )}
              {usageHighlights.toolsInUse != null && (
                <div className="space-y-1">
                  <p className="text-2xl font-bold">{usageHighlights.toolsInUse}</p>
                  <p className="text-xs text-muted-foreground">Active Tools</p>
                </div>
              )}
              {usageHighlights.mostActiveTool && (
                <div className="space-y-1">
                  <div className="flex items-center gap-1.5">
                    <Zap className="h-4 w-4 text-amber-500" />
                    <p className="text-sm font-semibold truncate">
                      {usageHighlights.mostActiveTool}
                    </p>
                  </div>
                  <p className="text-xs text-muted-foreground">Most Active</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Recent Admin Activity</CardTitle>
            <CardDescription>Latest admin operations</CardDescription>
          </CardHeader>
          <CardContent>
            {auditEvents.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <ClipboardList className="h-8 w-8 text-muted-foreground/40 mb-2" />
                <p className="text-sm text-muted-foreground">No recent audit activity</p>
              </div>
            ) : (
              <div className="space-y-3">
                {auditEvents.map((event) => (
                  <div key={event.id} className="flex items-center gap-3">
                    <div className="flex items-center gap-2 shrink-0">
                      {statusIcon(event.status)}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm truncate">{event.action}</p>
                      <p className="text-xs text-muted-foreground">
                        {event.actor} · {event.timestamp}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            )}
            <div className="mt-4 pt-3 border-t">
              <Link
                href="/dashboard/audit"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                View all audit events →
              </Link>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Integration Health</CardTitle>
            <CardDescription>Integration status at a glance</CardDescription>
          </CardHeader>
          <CardContent>
            {integrations.length > 0 ? (
              <div className="space-y-3">
                {integrations.map((integration) => (
                  <div key={integration.name} className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Plug className="h-4 w-4 text-muted-foreground" />
                      <span className="text-sm">{integration.name}</span>
                    </div>
                    {healthBadge(integration.status)}
                  </div>
                ))}
              </div>
            ) : (
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <Plug className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm text-muted-foreground">No integrations configured</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  Connect services from the Integrations page to monitor health here.
                </p>
              </div>
            )}
            <div className="mt-4 pt-3 border-t">
              <Link
                href="/dashboard/integrations"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                Manage integrations →
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
