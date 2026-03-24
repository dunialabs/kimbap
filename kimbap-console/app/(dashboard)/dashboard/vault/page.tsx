'use client';

import { useState } from 'react';
import { Lock, Search, RefreshCw, EyeOff } from 'lucide-react';

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

interface VaultSecret {
  name: string;
  type: 'api_key' | 'oauth_token' | 'password' | 'certificate' | 'other';
  createdAt: string;
  lastUsed: string | null;
}

const placeholderSecrets: VaultSecret[] = [
  { name: 'OPENAI_API_KEY', type: 'api_key', createdAt: '2026-03-01', lastUsed: '2026-03-23' },
  { name: 'GITHUB_TOKEN', type: 'oauth_token', createdAt: '2026-02-15', lastUsed: '2026-03-22' },
  { name: 'DB_PASSWORD', type: 'password', createdAt: '2026-01-20', lastUsed: '2026-03-23' },
  { name: 'SLACK_WEBHOOK', type: 'api_key', createdAt: '2026-02-28', lastUsed: '2026-03-20' },
  { name: 'TLS_CERT', type: 'certificate', createdAt: '2026-01-10', lastUsed: null },
  { name: 'ANTHROPIC_KEY', type: 'api_key', createdAt: '2026-03-15', lastUsed: '2026-03-23' },
];

function typeBadge(type: VaultSecret['type']) {
  switch (type) {
    case 'api_key':
      return (
        <Badge variant="outline" className="text-xs">
          API Key
        </Badge>
      );
    case 'oauth_token':
      return (
        <Badge
          variant="outline"
          className="text-xs bg-blue-500/10 text-blue-600 border-blue-500/20"
        >
          OAuth
        </Badge>
      );
    case 'password':
      return (
        <Badge
          variant="outline"
          className="text-xs bg-amber-500/10 text-amber-600 border-amber-500/20"
        >
          Password
        </Badge>
      );
    case 'certificate':
      return (
        <Badge
          variant="outline"
          className="text-xs bg-purple-500/10 text-purple-600 border-purple-500/20"
        >
          Cert
        </Badge>
      );
    default:
      return (
        <Badge variant="outline" className="text-xs">
          Other
        </Badge>
      );
  }
}

export default function VaultPage() {
  const [search, setSearch] = useState('');

  const filtered = placeholderSecrets.filter((s) =>
    s.name.toLowerCase().includes(search.toLowerCase()),
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[30px] font-bold tracking-tight flex items-center gap-2">
            <Lock className="h-6 w-6" />
            Vault
          </h1>
          <p className="text-base text-muted-foreground">
            Sealed secrets metadata. Values are only accessible via CLI.
          </p>
        </div>
        <Button variant="outline" size="sm">
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      <Card className="border-amber-500/20 bg-amber-500/5">
        <CardContent className="flex items-center gap-3 py-3">
          <EyeOff className="h-5 w-5 text-amber-600 shrink-0" />
          <span className="text-sm">
            Secret values are never displayed in the console. Use{' '}
            <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono">
              kimbap vault get
            </code>{' '}
            via CLI to retrieve values.
          </span>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <CardTitle className="text-base">Stored Secrets</CardTitle>
              <CardDescription>
                {filtered.length} secret{filtered.length !== 1 ? 's' : ''} in vault
              </CardDescription>
            </div>
            <div className="relative w-full sm:w-[240px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
              <Input
                placeholder="Filter secrets…"
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
              <Lock className="h-10 w-10 text-muted-foreground/40 mb-3" />
              <p className="text-sm text-muted-foreground">
                {search ? 'No secrets match your search' : 'No secrets stored yet'}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Value</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead>Last Used</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filtered.map((secret) => (
                    <TableRow key={secret.name}>
                      <TableCell className="font-mono text-sm font-medium">{secret.name}</TableCell>
                      <TableCell>{typeBadge(secret.type)}</TableCell>
                      <TableCell className="font-mono text-sm text-muted-foreground">
                        ••••••••••••
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(secret.createdAt).toLocaleDateString()}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {secret.lastUsed ? new Date(secret.lastUsed).toLocaleDateString() : '—'}
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
