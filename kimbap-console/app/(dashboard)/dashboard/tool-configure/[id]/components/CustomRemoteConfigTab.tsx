'use client';

import React, { useState, useEffect, useRef } from 'react';
import { Lock, Plus, X, AlertTriangle, Terminal, Eye, EyeOff } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { LazyStartConfiguration } from '@/components/tool-configure/LazyStartConfiguration';
import { PublicAccessConfiguration } from '@/components/tool-configure/PublicAccessConfiguration';
import { AnonymousAccessConfiguration } from '@/components/tool-configure/AnonymousAccessConfiguration';

interface CustomRemoteConfigTabProps {
  transportType?: 'url' | 'stdio';
  config: {
    url?: string;
    headers?: Record<string, string>;
    command?: string;
    args?: string[];
    env?: Record<string, string>;
    cwd?: string;
  };
  onConfigChange: (config: {
    url?: string;
    headers?: Record<string, string>;
    command?: string;
    args?: string[];
    env?: Record<string, string>;
    cwd?: string;
  }) => void;
  error: string | null;
  onDelete?: () => void;
  serverName?: string;
  lazyStartEnabled: boolean;
  onLazyStartEnabledChange: (value: boolean) => void;
  publicAccess: boolean;
  onPublicAccessChange: (value: boolean) => void;
  allowUserInput: boolean;
  isAdmin?: boolean;
  anonymousAccess: boolean;
  onAnonymousAccessChange: (value: boolean) => void;
  anonymousRateLimit: number;
  onAnonymousRateLimitChange: (value: number) => void;
}

const createHeaderId = () => `${Date.now()}-${Math.random().toString(36).slice(2)}`;

const normalizeQueryPart = (query: string) => {
  const trimmed = query.trim();
  if (!trimmed) return '';
  return trimmed.startsWith('?') ? trimmed : `?${trimmed}`;
};

const SENSITIVE_ENV_KEY_PATTERN = new RegExp(
  '(SECRET|TOKEN|PASSWORD|PASSPHRASE|API_KEY|ACCESS_KEY|SECRET_KEY|PRIVATE_KEY|CLIENT_SECRET|REFRESH_TOKEN|SESSION_SECRET|CREDENTIALS?)',
  'i',
);

const isSensitiveEnvKey = (key: string) => SENSITIVE_ENV_KEY_PATTERN.test(key);

export const CustomRemoteConfigTab = React.memo<CustomRemoteConfigTabProps>(
  ({
    transportType = 'url',
    config,
    onConfigChange,
    error,
    onDelete,
    serverName,
    lazyStartEnabled,
    onLazyStartEnabledChange,
    publicAccess,
    onPublicAccessChange,
    allowUserInput,
    isAdmin = false,
    anonymousAccess,
    onAnonymousAccessChange,
    anonymousRateLimit,
    onAnonymousRateLimitChange,
  }) => {
    const isStdio = transportType === 'stdio';

    // --- URL mode state ---
    // Parse URL into base and query parts
    const parseUrl = (url: string) => {
      const questionMarkIndex = url.indexOf('?');
      if (questionMarkIndex === -1) {
        return { basePart: url, queryPart: '' };
      }
      return {
        basePart: url.substring(0, questionMarkIndex),
        queryPart: url.substring(questionMarkIndex), // includes '?'
      };
    };

    const { basePart, queryPart } = parseUrl(config.url || '');
    const [editableQueryPart, setEditableQueryPart] = useState(queryPart);

    // Headers state - convert object to array for UI
    const [headers, setHeaders] = useState<Array<{ id: string; key: string; value: string }>>(
      () => {
        const entries = Object.entries(config.headers || {});
        return entries.length > 0
          ? entries.map(([key, value]) => ({ id: createHeaderId(), key, value }))
          : [{ id: createHeaderId(), key: '', value: '' }];
      },
    );

    // Update parent when query or headers change (URL mode only)
    useEffect(() => {
      if (isStdio) return;
      const normalizedQuery = normalizeQueryPart(editableQueryPart);
      const newUrl = basePart + normalizedQuery;
      const newHeaders = headers.reduce(
        (acc, h) => {
          if (h.key.trim()) {
            acc[h.key] = h.value;
          }
          return acc;
        },
        {} as Record<string, string>,
      );

      onConfigChange({ url: newUrl, headers: newHeaders });
    }, [editableQueryPart, headers, basePart, onConfigChange, isStdio]);

    // --- Stdio mode state ---
    const [envVars, setEnvVars] = useState<Array<{ id: string; key: string; value: string }>>(
      () => {
        const entries = Object.entries(config.env || {});
        return entries.length > 0
          ? entries.map(([key, value]) => ({ id: createHeaderId(), key, value }))
          : [{ id: createHeaderId(), key: '', value: '' }];
      },
    );
    const [hiddenEnvKeys, setHiddenEnvKeys] = useState<Set<string>>(new Set());
    const hasInitializedHiddenKeys = useRef(false);

    useEffect(() => {
      if (hasInitializedHiddenKeys.current) {
        return;
      }
      const initialHiddenKeys = new Set<string>();
      Object.keys(config.env || {}).forEach((key) => {
        if (isSensitiveEnvKey(key)) {
          initialHiddenKeys.add(key);
        }
      });
      setHiddenEnvKeys(initialHiddenKeys);
      hasInitializedHiddenKeys.current = true;
    }, [config.env]);

    const handleArgsChange = (value: string) => {
      const args = value === '' ? [] : value.split('\n');
      onConfigChange({ ...config, args });
    };

    const handleAddEnvVar = () => {
      setEnvVars([...envVars, { id: createHeaderId(), key: '', value: '' }]);
    };

    const handleRemoveEnvVar = (envId: string) => {
      if (envVars.length > 1) {
        const updated = envVars.filter((e) => e.id !== envId);
        setEnvVars(updated);
        const newEnv = updated.reduce(
          (acc, e) => {
            if (e.key.trim()) acc[e.key] = e.value;
            return acc;
          },
          {} as Record<string, string>,
        );
        onConfigChange({ ...config, env: newEnv });
      }
    };

    const handleEnvVarChange = (envId: string, field: 'key' | 'value', value: string) => {
      const updated = envVars.map((e) => (e.id === envId ? { ...e, [field]: value } : e));
      setEnvVars(updated);

      if (field === 'key') {
        const previousKey = envVars.find((e) => e.id === envId)?.key;
        setHiddenEnvKeys((prev) => {
          const next = new Set(prev);
          if (previousKey && previousKey !== value) {
            next.delete(previousKey);
          }
          if (isSensitiveEnvKey(value)) {
            next.add(value);
          } else {
            next.delete(value);
          }
          return next;
        });
      }

      const newEnv = updated.reduce(
        (acc, e) => {
          if (e.key.trim()) acc[e.key] = e.value;
          return acc;
        },
        {} as Record<string, string>,
      );
      onConfigChange({ ...config, env: newEnv });
    };

    const toggleEnvVarVisibility = (key: string) => {
      setHiddenEnvKeys((prev) => {
        const next = new Set(prev);
        if (next.has(key)) {
          next.delete(key);
        } else {
          next.add(key);
        }
        return next;
      });
    };

    const handleAddHeader = () => {
      setHeaders([...headers, { id: createHeaderId(), key: '', value: '' }]);
    };

    const handleRemoveHeader = (headerId: string) => {
      if (headers.length > 1) {
        setHeaders(headers.filter((header) => header.id !== headerId));
      }
    };

    const handleHeaderChange = (headerId: string, field: 'key' | 'value', value: string) => {
      setHeaders(
        headers.map((header) => (header.id === headerId ? { ...header, [field]: value } : header)),
      );
    };

    return (
      <div className="space-y-4">
        {isStdio ? (
          <>
            <div className="space-y-3">
              <h3 className="text-lg font-semibold text-foreground">
                Custom MCP Stdio Configuration
              </h3>
              {allowUserInput && (
                <p className="text-sm text-muted-foreground">
                  This tool is configured to allow user input via Kimbap Desk. Each user will receive
                  their own isolated configuration.
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label className="text-sm font-medium">Command</Label>
              {isAdmin && !allowUserInput ? (
                <Input
                  value={config.command || ''}
                  onChange={(e) => onConfigChange({ ...config, command: e.target.value })}
                  placeholder="/path/to/command"
                  className="font-mono text-sm"
                />
              ) : (
                <div className="bg-gray-50 dark:bg-gray-900 rounded-md p-3 border border-gray-200 dark:border-gray-700">
                  <div className="flex items-start gap-2">
                    <Terminal className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                    <div className="flex-1">
                      <p className="text-xs text-muted-foreground mb-1">Command (Read-only)</p>
                      <p className="font-mono text-sm break-all text-foreground">
                        {config.command}
                      </p>
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* Args textarea */}
            <div className="space-y-2">
              <Label className="text-sm font-medium">Arguments</Label>
              <Textarea
                value={(config.args || []).join('\n')}
                onChange={(e) => handleArgsChange(e.target.value)}
                placeholder="One argument per line"
                className="font-mono text-sm min-h-[100px]"
              />
              <p className="text-xs text-muted-foreground">
                Command-line arguments passed to the server process. Enter each argument on a
                separate line.
              </p>
            </div>

            {/* Env key-value editor */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label className="text-sm font-medium">Environment Variables</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handleAddEnvVar}
                  className="h-8"
                >
                  <Plus className="h-3 w-3 mr-1" />
                  Add Variable
                </Button>
              </div>

              <div className="space-y-2">
                {envVars.map((envVar, index) => (
                  <div key={envVar.id} className="flex gap-2 items-start">
                    <Input
                      aria-label={`Environment variable ${index + 1} key`}
                      placeholder="Key"
                      value={envVar.key}
                      onChange={(e) => handleEnvVarChange(envVar.id, 'key', e.target.value)}
                      className="flex-1"
                    />
                    <Input
                      aria-label={`Environment variable ${index + 1} value`}
                      type={
                        isSensitiveEnvKey(envVar.key) && hiddenEnvKeys.has(envVar.key)
                          ? 'password'
                          : 'text'
                      }
                      placeholder="Value"
                      value={envVar.value}
                      onChange={(e) => handleEnvVarChange(envVar.id, 'value', e.target.value)}
                      className="flex-1"
                    />
                    {isSensitiveEnvKey(envVar.key) && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => toggleEnvVarVisibility(envVar.key)}
                        aria-label={
                          hiddenEnvKeys.has(envVar.key)
                            ? `Show value for environment variable ${index + 1}`
                            : `Hide value for environment variable ${index + 1}`
                        }
                        className="h-10 w-10 p-0"
                      >
                        {hiddenEnvKeys.has(envVar.key) ? (
                          <Eye className="h-4 w-4" />
                        ) : (
                          <EyeOff className="h-4 w-4" />
                        )}
                      </Button>
                    )}
                    {envVars.length > 1 && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => handleRemoveEnvVar(envVar.id)}
                        aria-label={`Remove environment variable ${index + 1}`}
                        className="h-10 w-10 p-0 hover:bg-red-50 hover:text-red-600 dark:text-red-400 dark:hover:text-red-400 dark:hover:bg-red-900/20"
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">
                Environment variables for the stdio MCP server process
              </p>
            </div>

            <div className="space-y-2">
              <Label className="text-sm font-medium">Working Directory</Label>
              {isAdmin ? (
                <Input
                  value={config.cwd || ''}
                  onChange={(e) => onConfigChange({ ...config, cwd: e.target.value })}
                  placeholder="/path/to/project (optional)"
                  className="font-mono text-sm"
                />
              ) : (
                <div className="bg-gray-50 dark:bg-gray-900 rounded-md p-3 border border-gray-200 dark:border-gray-700">
                  <div className="flex items-start gap-2">
                    <Terminal className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                    <div className="flex-1">
                      <p className="text-xs text-muted-foreground mb-1">
                        Working Directory (Read-only)
                      </p>
                      <p className="font-mono text-sm break-all text-foreground">
                        {config.cwd || '-'}
                      </p>
                    </div>
                  </div>
                </div>
              )}
              <p className="text-xs text-muted-foreground">
                Optional working directory used when spawning the server process
              </p>
            </div>
          </>
        ) : (
          <>
            {/* Tool Name Display */}
            <div className="space-y-3">
              <h3 className="text-lg font-semibold text-foreground">
                Custom MCP URL Configuration
              </h3>
              {allowUserInput && (
                <p className="text-sm text-muted-foreground">
                  This tool is configured to allow user input via Kimbap Desk. Users will provide
                  their own credentials when using this tool.
                </p>
              )}
              <p className="text-sm text-muted-foreground">
                Edit the Custom MCP URL configuration below. Note: The base URL path cannot be
                modified.
              </p>
            </div>

            {/* URL Display/Edit Section */}
            <div className="space-y-2">
              <Label className="text-sm font-medium">MCP URL</Label>

              {/* Non-editable base part */}
              <div className="bg-gray-50 dark:bg-gray-900 rounded-md p-3 border border-gray-200 dark:border-gray-700">
                <div className="flex items-start gap-2">
                  <Lock className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                  <div className="flex-1">
                    <p className="text-xs text-muted-foreground mb-1">Base URL (Read-only)</p>
                    <p className="font-mono text-sm break-all text-foreground">{basePart}</p>
                  </div>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="query-params" className="text-sm">
                  Query Parameters (Editable)
                </Label>
                <Input
                  id="query-params"
                  value={editableQueryPart}
                  onChange={(e) => setEditableQueryPart(e.target.value)}
                  placeholder="?key1=value1&key2=value2"
                  className="font-mono text-sm"
                />
                <p className="text-xs text-muted-foreground">
                  You can modify query parameters only. The base URL path cannot be changed.
                </p>
              </div>

              {/* Full URL preview */}
              <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md p-2">
                <p className="text-xs text-blue-600 dark:text-blue-400 mb-1">Full URL Preview:</p>
                <p className="font-mono text-xs break-all text-blue-800 dark:text-blue-300">
                  {basePart + normalizeQueryPart(editableQueryPart)}
                </p>
              </div>
            </div>

            {/* Headers Editor */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label className="text-sm font-medium">Headers</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handleAddHeader}
                  className="h-8"
                >
                  <Plus className="h-3 w-3 mr-1" />
                  Add Header
                </Button>
              </div>

              <div className="space-y-2">
                {headers.map((header, index) => (
                  <div key={header.id} className="flex gap-2 items-start">
                    <Input
                      aria-label={`Header ${index + 1} key`}
                      placeholder="Key"
                      value={header.key}
                      onChange={(e) => handleHeaderChange(header.id, 'key', e.target.value)}
                      className="flex-1"
                    />
                    <Input
                      aria-label={`Header ${index + 1} value`}
                      placeholder="Value"
                      value={header.value}
                      onChange={(e) => handleHeaderChange(header.id, 'value', e.target.value)}
                      className="flex-1"
                    />
                    {headers.length > 1 && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => handleRemoveHeader(header.id)}
                        aria-label={`Remove header ${index + 1}`}
                        className="h-10 w-10 p-0 hover:bg-red-50 hover:text-red-600 dark:text-red-400 dark:hover:text-red-400 dark:hover:bg-red-900/20"
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">
                Custom headers for the MCP server connection
              </p>
            </div>
          </>
        )}

        {/* Error Display */}
        {error && (
          <div
            role="alert"
            className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3"
          >
            <div className="flex items-start gap-2">
              <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5" />
              <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
            </div>
          </div>
        )}

        {/* PublicAccessConfiguration */}
        <PublicAccessConfiguration checked={publicAccess} onCheckedChange={onPublicAccessChange} />

        {!allowUserInput && (
          <AnonymousAccessConfiguration
            checked={anonymousAccess}
            onCheckedChange={onAnonymousAccessChange}
            rateLimit={anonymousRateLimit}
            onRateLimitChange={onAnonymousRateLimitChange}
          />
        )}

        {/* LazyStartConfiguration */}
        <LazyStartConfiguration
          checked={lazyStartEnabled}
          onCheckedChange={onLazyStartEnabledChange}
          supportsIdleSleep={isStdio}
        />

        {/* Delete Tool Section */}
        {onDelete && (
          <div className="border-t border-gray-200 dark:border-gray-700 pt-4 mt-6">
            <Button
              variant="outline"
              size="sm"
              onClick={onDelete}
              className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 hover:bg-red-50 border-red-200 dark:border-red-800 dark:hover:bg-red-900/20"
            >
              Delete Tool
            </Button>
          </div>
        )}
      </div>
    );
  },
);

CustomRemoteConfigTab.displayName = 'CustomRemoteConfigTab';
