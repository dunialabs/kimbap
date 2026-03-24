'use client';

import { useMemo } from 'react';

import { TagInput } from '@/components/tag-input';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

type CacheScope = 'user' | 'tenant' | 'global';
type AdmissionPolicy = 'immediate' | 'second_hit';

export interface CachePolicyConfig {
  enabled?: boolean;
  ttlSeconds?: number;
  scope?: CacheScope;
  admissionPolicy?: AdmissionPolicy;
  admissionWindowSeconds?: number;
  maxEntryBytes?: number;
  key?: {
    denyFields?: string[];
    allowFields?: string[];
  };
}

interface CachePolicyEditorProps {
  cachePolicy?: CachePolicyConfig;
  onPolicyChange: (policy: CachePolicyConfig) => void;
  entityType: 'tool' | 'prompt' | 'resource';
  entityName: string;
  isDangerous?: boolean;
  globalCacheEnabled?: boolean;
}

export function CachePolicyEditor({
  cachePolicy,
  onPolicyChange,
  entityType,
  entityName,
  isDangerous = false,
  globalCacheEnabled = true,
}: CachePolicyEditorProps) {
  const policy = useMemo<CachePolicyConfig>(
    () => ({
      enabled: cachePolicy?.enabled ?? false,
      ttlSeconds: cachePolicy?.ttlSeconds,
      scope: cachePolicy?.scope ?? 'user',
      admissionPolicy: cachePolicy?.admissionPolicy ?? 'immediate',
      admissionWindowSeconds: cachePolicy?.admissionWindowSeconds,
      maxEntryBytes: cachePolicy?.maxEntryBytes,
      key: {
        denyFields: cachePolicy?.key?.denyFields ?? [],
        allowFields: cachePolicy?.key?.allowFields ?? [],
      },
    }),
    [cachePolicy],
  );

  const setPolicy = (next: Partial<CachePolicyConfig>) => {
    onPolicyChange({
      ...policy,
      ...next,
      key: {
        denyFields: next.key?.denyFields ?? policy.key?.denyFields ?? [],
        allowFields: next.key?.allowFields ?? policy.key?.allowFields ?? [],
      },
    });
  };

  const disabledByPolicy = isDangerous || !globalCacheEnabled;

  return (
    <div className="rounded-lg border p-4 space-y-4 bg-background">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-sm font-semibold text-foreground">{entityName}</p>
          <p className="text-xs text-muted-foreground">{entityType} cache policy</p>
        </div>
        <div className="flex items-center gap-2">
          {!globalCacheEnabled && (
            <Badge variant="secondary" className="text-xs">
              Result caching is disabled globally
            </Badge>
          )}
          {isDangerous && (
            <Badge variant="destructive" className="text-xs">
              Caching not available for approval-required tools
            </Badge>
          )}
          <div className="flex items-center gap-2">
            <Label className="text-xs text-muted-foreground">Enabled</Label>
            <Switch
              checked={Boolean(policy.enabled) && !disabledByPolicy}
              disabled={disabledByPolicy}
              onCheckedChange={(checked) => setPolicy({ enabled: checked })}
              size="sm"
            />
          </div>
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <div className="space-y-2">
          <Label className="text-xs">TTL seconds</Label>
          <Input
            type="number"
            min={1}
            value={policy.ttlSeconds ?? ''}
            disabled={disabledByPolicy}
            onChange={(e) =>
              setPolicy({
                ttlSeconds: e.target.value === '' ? undefined : Number(e.target.value),
              })
            }
          />
        </div>
        <div className="space-y-2">
          <div className="flex items-center gap-1">
            <Label className="text-xs">Scope</Label>
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="text-[11px] text-muted-foreground cursor-help">help</span>
                </TooltipTrigger>
                <TooltipContent className="max-w-xs text-xs">
                  user: cache per user. global: shared across all users. tenant: not yet available.
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
          <Select
            value={policy.scope}
            onValueChange={(value) => setPolicy({ scope: value as CacheScope })}
            disabled={disabledByPolicy}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select scope" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="user">user</SelectItem>
              <SelectItem
                value="tenant"
                disabled
                title="Coming soon - tenant scope requires backend support"
              >
                tenant (coming soon)
              </SelectItem>
              <SelectItem value="global">global</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label className="text-xs">Admission policy</Label>
          <Select
            value={policy.admissionPolicy}
            onValueChange={(value) => setPolicy({ admissionPolicy: value as AdmissionPolicy })}
            disabled={disabledByPolicy}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select admission policy" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="immediate">immediate</SelectItem>
              <SelectItem value="second_hit">second_hit</SelectItem>
            </SelectContent>
          </Select>
        </div>
        {policy.admissionPolicy === 'second_hit' && (
          <div className="space-y-2">
            <Label className="text-xs">Admission window seconds</Label>
            <Input
              type="number"
              min={1}
              value={policy.admissionWindowSeconds ?? ''}
              disabled={disabledByPolicy}
              onChange={(e) =>
                setPolicy({
                  admissionWindowSeconds:
                    e.target.value === '' ? undefined : Number(e.target.value),
                })
              }
            />
          </div>
        )}
        <div className="space-y-2">
          <Label className="text-xs">Max entry bytes</Label>
          <Input
            type="number"
            min={1024}
            value={policy.maxEntryBytes ?? ''}
            disabled={disabledByPolicy}
            onChange={(e) =>
              setPolicy({
                maxEntryBytes: e.target.value === '' ? undefined : Number(e.target.value),
              })
            }
          />
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <div className="space-y-2">
          <Label className="text-xs">Key deny fields</Label>
          <TagInput
            value={policy.key?.denyFields || []}
            onChange={(tags) => setPolicy({ key: { ...policy.key, denyFields: tags } })}
            placeholder="Add fields"
            disabled={disabledByPolicy}
          />
        </div>
        <div className="space-y-2">
          <Label className="text-xs">Key allow fields</Label>
          <TagInput
            value={policy.key?.allowFields || []}
            onChange={(tags) => setPolicy({ key: { ...policy.key, allowFields: tags } })}
            placeholder="Add fields"
            disabled={disabledByPolicy}
          />
        </div>
      </div>
    </div>
  );
}
