'use client';

import { useState } from 'react';
import { Blocks, Search, RefreshCw } from 'lucide-react';

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

interface Skill {
  name: string;
  version: string;
  actionsCount: number;
  status: 'active' | 'disabled' | 'error';
  installedAt: string;
}

const placeholderSkills: Skill[] = [
  {
    name: 'web-search',
    version: '1.2.0',
    actionsCount: 3,
    status: 'active',
    installedAt: '2026-03-10',
  },
  {
    name: 'file-manager',
    version: '2.0.1',
    actionsCount: 8,
    status: 'active',
    installedAt: '2026-03-08',
  },
  {
    name: 'code-executor',
    version: '0.9.4',
    actionsCount: 2,
    status: 'active',
    installedAt: '2026-03-05',
  },
  {
    name: 'email-sender',
    version: '1.1.0',
    actionsCount: 4,
    status: 'disabled',
    installedAt: '2026-02-28',
  },
  {
    name: 'database-query',
    version: '3.0.0',
    actionsCount: 6,
    status: 'active',
    installedAt: '2026-02-20',
  },
  {
    name: 'slack-notify',
    version: '1.0.2',
    actionsCount: 2,
    status: 'error',
    installedAt: '2026-02-15',
  },
];

function statusBadge(status: Skill['status']) {
  switch (status) {
    case 'active':
      return (
        <Badge className="bg-emerald-100 text-emerald-800 border-emerald-200 dark:bg-emerald-900 dark:text-emerald-300 dark:border-emerald-800 hover:bg-emerald-200">
          Active
        </Badge>
      );
    case 'disabled':
      return <Badge variant="secondary">Disabled</Badge>;
    case 'error':
      return (
        <Badge className="bg-red-100 text-red-800 border-red-200 dark:bg-red-900 dark:text-red-300 dark:border-red-800 hover:bg-red-200">
          Error
        </Badge>
      );
  }
}

export default function SkillsPage() {
  const [search, setSearch] = useState('');

  const filtered = placeholderSkills.filter((s) =>
    s.name.toLowerCase().includes(search.toLowerCase()),
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[30px] font-bold tracking-tight flex items-center gap-2">
            <Blocks className="h-6 w-6" />
            Skills
          </h1>
          <p className="text-base text-muted-foreground">
            Installed skill packages and their registered actions.
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
              <CardTitle className="text-base">Skill Inventory</CardTitle>
              <CardDescription>
                {filtered.length} skill{filtered.length !== 1 ? 's' : ''} installed
              </CardDescription>
            </div>
            <div className="relative w-full sm:w-[240px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
              <Input
                placeholder="Filter skills…"
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
              <Blocks className="h-10 w-10 text-muted-foreground/40 mb-3" />
              <p className="text-sm text-muted-foreground">
                {search ? 'No skills match your search' : 'No skills installed yet'}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Version</TableHead>
                    <TableHead className="text-center">Actions</TableHead>
                    <TableHead className="text-center">Status</TableHead>
                    <TableHead>Installed</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filtered.map((skill) => (
                    <TableRow key={skill.name}>
                      <TableCell className="font-mono text-sm font-medium">{skill.name}</TableCell>
                      <TableCell className="font-mono text-sm text-muted-foreground">
                        {skill.version}
                      </TableCell>
                      <TableCell className="text-center text-sm">{skill.actionsCount}</TableCell>
                      <TableCell className="text-center">{statusBadge(skill.status)}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(skill.installedAt).toLocaleDateString()}
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
