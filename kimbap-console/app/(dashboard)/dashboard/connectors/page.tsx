'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import {
  Plug,
  Search,
  RefreshCw,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  Loader2,
} from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

interface Integration {
  id: string;
  name: string;
  provider: string;
  runningState: number;
  enabled: boolean;
  category: number;
  enabledFunctions: number;
  totalFunctions: number;
}

const TOOL_TYPE_PROVIDERS: Record<number, string> = {
  1: 'GitHub',
  2: 'Notion',
  3: 'PostgreSQL',
  4: 'Slack',
  5: 'OpenAI',
  6: 'Stripe',
  7: 'Linear',
  8: 'AWS',
  9: 'Figma',
  10: 'Google Drive',
  11: 'Sequential Thinking',
  12: 'WCGW',
  13: 'MongoDB',
  14: 'MySQL',
  15: 'Redis',
  16: 'Elasticsearch',
  17: 'Salesforce',
  18: 'Brave Search',
  19: 'Google Calendar',
  20: 'Canva',
  21: 'Zendesk',
};

const CATEGORIES: Record<number, string> = {
  1: 'Template',
  2: 'Custom Remote',
  3: 'REST API',
  4: 'Skills',
  5: 'Custom Stdio',
};

function healthBadge(runningState: number) {
  switch (runningState) {
    case 0:
      return (
        <Badge className="bg-emerald-100 text-emerald-800 border-emerald-200 dark:bg-emerald-900 dark:text-emerald-300 dark:border-emerald-800 hover:bg-emerald-200">
          <CheckCircle2 className="h-3 w-3 mr-1" />
          Online
        </Badge>
      );
    case 2:
      return (
        <Badge className="bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800 hover:bg-blue-200">
          <Loader2 className="h-3 w-3 mr-1 animate-spin" />
          Connecting
        </Badge>
      );
    case 3:
      return (
        <Badge className="bg-red-100 text-red-800 border-red-200 dark:bg-red-900 dark:text-red-300 dark:border-red-800 hover:bg-red-200">
          <AlertTriangle className="h-3 w-3 mr-1" />
          Error
        </Badge>
      );
    default:
      return (
        <Badge className="bg-slate-100 text-slate-800 border-slate-200 dark:bg-slate-900 dark:text-slate-300 dark:border-slate-800 hover:bg-slate-200">
          <XCircle className="h-3 w-3 mr-1" />
          Offline
        </Badge>
      );
  }
}

export default function ConnectorsPage() {
  const [search, setSearch] = useState('');
  const [connectors, setConnectors] = useState<Integration[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [serverConfigured, setServerConfigured] = useState(true);
  const isLoadingRef = useRef(false);

  const loadConnectors = useCallback(async () => {
    if (isLoadingRef.current) return;

    try {
      isLoadingRef.current = true;
      setIsLoading(true);
      setError(null);

      const { api } = await import('@/lib/api-client');

      const serverInfoResponse = await api.servers.getInfo();
      if (serverInfoResponse.data?.common?.code !== 0) {
        setError('Failed to load server info.');
        return;
      }
      const proxyId = serverInfoResponse.data?.data?.proxyId;

      if (proxyId == null) {
        setServerConfigured(false);
        setConnectors([]);
        return;
      }

      setServerConfigured(true);

      const response = await api.tools.getToolList({
        proxyId: Number(proxyId),
        handleType: 1,
      });

      if (response.data?.common?.code !== 0) {
        setError('Failed to load integrations.');
        return;
      }

      const toolList = response.data?.data?.toolList || [];

      const mapped: Integration[] = toolList.map(
        (tool: {
          toolId: string;
          name: string;
          toolType: number;
          enabled: boolean;
          runningState?: number;
          category?: number;
          toolFuncs?: Array<{ funcName: string; enabled: boolean }>;
        }) => ({
          id: tool.toolId,
          name: tool.name,
          provider: TOOL_TYPE_PROVIDERS[tool.toolType] || tool.name,
          runningState: tool.runningState ?? 1,
          enabled: tool.enabled ?? false,
          category: tool.category ?? 1,
          enabledFunctions: tool.toolFuncs?.filter((f) => f.enabled).length ?? 0,
          totalFunctions: tool.toolFuncs?.length ?? 0,
        }),
      );

      setConnectors(mapped);
    } catch {
      setError('Failed to load integrations. Check your connection and try again.');
    } finally {
      setIsLoading(false);
      isLoadingRef.current = false;
    }
  }, []);

  useEffect(() => {
    loadConnectors();
  }, [loadConnectors]);

  const filtered = connectors.filter(
    (c) =>
      c.name.toLowerCase().includes(search.toLowerCase()) ||
      c.provider.toLowerCase().includes(search.toLowerCase()),
  );

  const errorCount = connectors.filter((c) => c.runningState === 3).length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[30px] font-bold tracking-tight flex items-center gap-2">
            <Plug className="h-6 w-6" />
            Integrations
          </h1>
          <p className="text-base text-muted-foreground">
            Integration and OAuth connection health — status, refresh, and recovery.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={loadConnectors} disabled={isLoading}>
          {isLoading ? (
            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4 mr-2" />
          )}
          Refresh
        </Button>
      </div>

      {!error && errorCount > 0 && (
        <Card className="border-red-500/30 bg-red-500/5">
          <CardContent className="flex items-center gap-3 py-3">
            <AlertTriangle className="h-5 w-5 text-red-600 dark:text-red-400 shrink-0" />
            <span className="text-sm font-medium">
              {errorCount} integration{errorCount !== 1 ? 's' : ''} in error state — check
              configuration and connectivity.
            </span>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <CardTitle className="text-base">Integration Registry</CardTitle>
              <CardDescription>
                {isLoading
                  ? 'Loading integrations…'
                  : search && filtered.length !== connectors.length
                    ? `${filtered.length} of ${connectors.length} integration${connectors.length !== 1 ? 's' : ''} matching`
                    : `${connectors.length} integration${connectors.length !== 1 ? 's' : ''} configured`}
              </CardDescription>
            </div>
            <div className="relative w-full sm:w-[240px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
              <Input
                placeholder="Filter integrations…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10 h-9"
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <Loader2 className="h-8 w-8 animate-spin mx-auto mb-3 text-muted-foreground" />
                <p className="text-sm text-muted-foreground">Loading integrations…</p>
              </div>
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <XCircle className="h-10 w-10 text-muted-foreground/40 mb-3" />
              <p className="text-sm text-muted-foreground mb-3">{error}</p>
              <Button variant="outline" size="sm" onClick={loadConnectors}>
                <RefreshCw className="h-4 w-4 mr-2" />
                Retry
              </Button>
            </div>
          ) : !serverConfigured ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Plug className="h-10 w-10 opacity-40 mb-3" />
              <p className="text-sm font-medium mb-1">Server Not Configured</p>
              <p className="text-sm text-muted-foreground">
                Set up your Kimbap server to manage integrations.
              </p>
            </div>
          ) : filtered.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Plug className="h-10 w-10 text-muted-foreground/40 mb-3" />
              <p className="text-sm font-medium mb-1">
                {search ? 'No Matches' : 'No Integrations Configured'}
              </p>
              <p className="text-sm text-muted-foreground">
                {search
                  ? 'No integrations match your search.'
                  : 'Add integrations from the tool configuration page.'}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Provider</TableHead>
                    <TableHead className="text-center">Health</TableHead>
                    <TableHead className="text-center">Config</TableHead>
                    <TableHead>Category</TableHead>
                    <TableHead>Functions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filtered.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="font-mono text-sm font-medium">{item.name}</TableCell>
                      <TableCell className="text-sm">{item.provider}</TableCell>
                      <TableCell className="text-center">
                        {healthBadge(item.runningState)}
                      </TableCell>
                      <TableCell className="text-center">
                        {item.enabled ? (
                          <span className="text-xs font-medium text-emerald-600 dark:text-emerald-400">
                            Enabled
                          </span>
                        ) : (
                          <span className="text-xs font-medium text-muted-foreground">
                            Disabled
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {CATEGORIES[item.category] || 'Unknown'}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {item.enabledFunctions}/{item.totalFunctions} enabled
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
