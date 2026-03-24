'use client';

import { useMemo, useState } from 'react';
import { Loader2, Trash2 } from 'lucide-react';
import { toast } from 'sonner';

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { api } from '@/lib/api-client';

interface CachePurgePanelProps {
  serverId?: string;
  toolName?: string;
  promptName?: string;
  resourceUri?: string;
  onPurgeComplete?: () => void;
}

type PurgeType = 'global' | 'server' | 'tool' | 'prompt' | 'resource';

interface PurgeActionItem {
  type: PurgeType;
  label: string;
  description: string;
}

export function CachePurgePanel({
  serverId,
  toolName,
  promptName,
  resourceUri,
  onPurgeComplete,
}: CachePurgePanelProps) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState('manual cache purge');
  const [loadingType, setLoadingType] = useState<PurgeType | null>(null);
  const [activeAction, setActiveAction] = useState<PurgeActionItem | null>(null);

  const actions = useMemo<PurgeActionItem[]>(() => {
    const list: PurgeActionItem[] = [
      {
        type: 'global',
        label: 'Purge Global Cache',
        description: 'Remove all cached entries across all servers.',
      },
    ];

    if (serverId) {
      list.push({
        type: 'server',
        label: 'Purge Server Cache',
        description: `Remove cache entries for server ${serverId}.`,
      });
    }

    if (serverId && toolName) {
      list.push({
        type: 'tool',
        label: 'Purge Tool Cache',
        description: `Remove cache entries for tool ${toolName}.`,
      });
    }

    if (serverId && promptName) {
      list.push({
        type: 'prompt',
        label: 'Purge Prompt Cache',
        description: `Remove cache entries for prompt ${promptName}.`,
      });
    }

    if (serverId && resourceUri) {
      list.push({
        type: 'resource',
        label: 'Purge Resource Cache',
        description: `Remove cache entries for resource ${resourceUri}.`,
      });
    }

    return list;
  }, [serverId, toolName, promptName, resourceUri]);

  const runPurge = async () => {
    if (!activeAction) return;
    setLoadingType(activeAction.type);

    try {
      if (activeAction.type === 'global') {
        await api.tools.purgeCacheGlobal({ reason });
      } else if (activeAction.type === 'server' && serverId) {
        await api.tools.purgeCacheServer({ serverId, reason });
      } else if (activeAction.type === 'tool' && serverId && toolName) {
        await api.tools.purgeCacheTool({ serverId, toolName, reason });
      } else if (activeAction.type === 'prompt' && serverId && promptName) {
        await api.tools.purgeCachePrompt({ serverId, promptName, reason });
      } else if (activeAction.type === 'resource' && serverId && resourceUri) {
        await api.tools.purgeCacheResource({ serverId, uri: resourceUri, reason });
      }

      toast.success('Cache purge completed');
      onPurgeComplete?.();
      setOpen(false);
      setActiveAction(null);
    } catch (error: any) {
      toast.error(error?.response?.data?.common?.message || error?.message || 'Cache purge failed');
    } finally {
      setLoadingType(null);
    }
  };

  return (
    <div className="rounded-lg border p-4 space-y-3">
      <div>
        <p className="text-sm font-semibold">Cache Purge</p>
        <p className="text-xs text-muted-foreground">
          Purge cache by scope to invalidate stale result entries.
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        {actions.map((action) => (
          <Button
            key={action.type}
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              setActiveAction(action);
              setOpen(true);
            }}
            className="gap-2"
          >
            <Trash2 className="h-3.5 w-3.5" />
            {action.label}
          </Button>
        ))}
      </div>

      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{activeAction?.label || 'Confirm purge'}</AlertDialogTitle>
            <AlertDialogDescription>{activeAction?.description}</AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-2">
            <Label htmlFor="cache-purge-reason">Reason</Label>
            <Input
              id="cache-purge-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="manual cache purge"
            />
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={Boolean(loadingType)}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={(e) => {
                e.preventDefault();
                void runPurge();
              }}
              disabled={Boolean(loadingType)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {loadingType ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  Purging...
                </span>
              ) : (
                'Confirm Purge'
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
