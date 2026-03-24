'use client';

import { useState } from 'react';
import { Key, Search, RefreshCw, CheckCircle2, XCircle, Clock } from 'lucide-react';

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

interface AgentToken {
  id: string;
  agentName: string;
  createdAt: string;
  expiresAt: string;
  lastUsed: string | null;
  status: 'active' | 'expired' | 'revoked';
}

const placeholderTokens: AgentToken[] = [
  {
    id: 'tok_a1b2c3',
    agentName: 'claude-dev',
    createdAt: '2026-03-10',
    expiresAt: '2026-06-10',
    lastUsed: '2026-03-23',
    status: 'active',
  },
  {
    id: 'tok_d4e5f6',
    agentName: 'ci-pipeline',
    createdAt: '2026-03-01',
    expiresAt: '2026-04-01',
    lastUsed: '2026-03-22',
    status: 'active',
  },
  {
    id: 'tok_g7h8i9',
    agentName: 'staging-bot',
    createdAt: '2026-02-01',
    expiresAt: '2026-03-01',
    lastUsed: '2026-02-28',
    status: 'expired',
  },
  {
    id: 'tok_j0k1l2',
    agentName: 'monitoring',
    createdAt: '2026-03-15',
    expiresAt: '2026-09-15',
    lastUsed: '2026-03-23',
    status: 'active',
  },
  {
    id: 'tok_m3n4o5',
    agentName: 'test-runner',
    createdAt: '2026-01-20',
    expiresAt: '2026-04-20',
    lastUsed: null,
    status: 'revoked',
  },
];

function statusBadge(status: AgentToken['status']) {
  switch (status) {
    case 'active':
      return (
        <Badge className="bg-emerald-100 text-emerald-800 border-emerald-200 dark:bg-emerald-900 dark:text-emerald-300 dark:border-emerald-800 hover:bg-emerald-200">
          <CheckCircle2 className="h-3 w-3 mr-1" />
          Active
        </Badge>
      );
    case 'expired':
      return (
        <Badge className="bg-amber-100 text-amber-800 border-amber-200 dark:bg-amber-900 dark:text-amber-300 dark:border-amber-800 hover:bg-amber-200">
          <Clock className="h-3 w-3 mr-1" />
          Expired
        </Badge>
      );
    case 'revoked':
      return (
        <Badge className="bg-red-100 text-red-800 border-red-200 dark:bg-red-900 dark:text-red-300 dark:border-red-800 hover:bg-red-200">
          <XCircle className="h-3 w-3 mr-1" />
          Revoked
        </Badge>
      );
  }
}

export default function TokensPage() {
  const [search, setSearch] = useState('');

  const filtered = placeholderTokens.filter(
    (t) =>
      t.agentName.toLowerCase().includes(search.toLowerCase()) ||
      t.id.toLowerCase().includes(search.toLowerCase()),
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[30px] font-bold tracking-tight flex items-center gap-2">
            <Key className="h-6 w-6" />
            Tokens
          </h1>
          <p className="text-base text-muted-foreground">
            Access tokens issued to agents and services.
          </p>
        </div>
        <Button variant="outline" size="sm">
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      <Card>
        <CardHeader>
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <CardTitle className="text-base">Token Registry</CardTitle>
              <CardDescription>
                {filtered.length} token{filtered.length !== 1 ? 's' : ''} issued
              </CardDescription>
            </div>
            <div className="relative w-full sm:w-[240px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
              <Input
                placeholder="Filter tokens…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10 h-9"
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {filtered.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Key className="h-10 w-10 text-muted-foreground/40 mb-3" />
              <p className="text-sm text-muted-foreground">
                {search ? 'No tokens match your search' : 'No tokens issued yet'}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Token ID</TableHead>
                    <TableHead>Agent</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead>Expires</TableHead>
                    <TableHead>Last Used</TableHead>
                    <TableHead className="text-center">Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filtered.map((token) => (
                    <TableRow key={token.id}>
                      <TableCell className="font-mono text-sm text-muted-foreground">
                        {token.id}
                      </TableCell>
                      <TableCell className="font-mono text-sm font-medium">
                        {token.agentName}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(token.createdAt).toLocaleDateString()}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(token.expiresAt).toLocaleDateString()}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {token.lastUsed ? new Date(token.lastUsed).toLocaleDateString() : '—'}
                      </TableCell>
                      <TableCell className="text-center">{statusBadge(token.status)}</TableCell>
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
