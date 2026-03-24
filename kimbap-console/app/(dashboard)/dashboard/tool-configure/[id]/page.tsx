'use client';

export const runtime = 'edge';

import { toast } from 'sonner';
import {
  ArrowLeft,
  Lock,
  Zap,
  Shield,
  Check,
  Settings,
  Users,
  Eye,
  EyeOff,
  Search,
  CheckSquare,
  Square,
  Minus,
  Database,
  Bot,
  LucideIcon,
  FileText,
  AlertTriangle,
  Building2,
  User,
  Info,
  ExternalLink,
  ShieldCheck,
} from 'lucide-react';
import Link from 'next/link';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import ReactMarkdown from 'react-markdown';

import { api } from '@/lib/api-client';
import { Tool, Credential, ServerAuthType } from '@/types/api';
import { MasterPasswordDialog } from '@/components/master-password-dialog';
import { AccessTokenDialog } from '@/components/access-token-dialog';
import { MasterPasswordManager } from '@/lib/crypto';
import { LazyStartConfiguration } from '@/components/tool-configure/LazyStartConfiguration';
import { PublicAccessConfiguration } from '@/components/tool-configure/PublicAccessConfiguration';
import { AnonymousAccessConfiguration } from '@/components/tool-configure/AnonymousAccessConfiguration';
import {
  CachePolicyConfig,
  CachePolicyEditor,
} from '@/components/tool-configure/cache-policy-editor';
import { CachePurgePanel } from '@/components/dashboard/cache-purge-panel';
import { CustomRemoteConfigTab } from './components/CustomRemoteConfigTab';
import { SkillsConfigTab } from '@/components/tool-configure/SkillsConfigTab';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { MCPServerConfig } from '@/types/mcp-server';
import { cn } from '@/lib/utils';
import { useMasterPassword } from '@/contexts/master-password-context';
import {
  OAUTH_AUTHORIZATION_URL_KEY,
  OAUTH_BASE_URL_KEY,
  OAUTH_ENDPOINT_OVERRIDE_KEYS,
  OAUTH_TOKEN_URL_KEY,
  resolveOAuthEndpoints,
  type OAuthEndpointOverrides,
} from '@/lib/oauth-endpoint-overrides';

// Constants
const TOOL_TYPE_MAP: Record<number, string> = {
  1: 'github',
  2: 'notion',
  3: 'postgresql',
  4: 'slack',
  5: 'openai',
  6: 'stripe',
  7: 'linear',
  8: 'aws',
  9: 'figma',
  10: 'googledrive',
  11: 'sequential-thinking',
  12: 'wcgw',
  13: 'mongodb',
  14: 'mysql',
  15: 'redis',
  16: 'elasticsearch',
  17: 'salesforce',
  18: 'brave-search',
  19: 'googlecalendar',
  20: 'canva',
  21: 'zendesk',
};

const STATUS_MAP: Record<number, MCPServerConfig['status']> = {
  0: 'connected',
  1: 'connect_failed',
  2: 'connecting',
  3: 'error',
  4: 'sleeping',
};

const ICON_MAP: Record<string, LucideIcon> = {
  Search,
  Database,
  Bot,
};

// Type definitions
interface ServerCapabilities {
  functions?: Array<{
    funcName: string;
    name?: string;
    description?: string;
    enabled: boolean;
    dangerLevel?: number;
  }>;
  resources?: Array<
    | {
        uri: string;
        enabled: boolean;
      }
    | string
  >;
}

interface CachePolicyState {
  tools: Record<string, CachePolicyConfig>;
  prompts: Record<string, CachePolicyConfig>;
  resources: {
    exact: Record<string, CachePolicyConfig>;
    patterns: any[];
  };
}

interface CacheHealthState {
  enabled: boolean;
  health: { ok: boolean; details?: string; backend: string };
  metrics?: Record<string, number>;
}

interface StepConfig {
  id: 'credentials' | 'functions' | 'permissions';
  title: string;
  description: string;
  icon: LucideIcon;
  bgColor: string;
}

const isCustomMcpCategory = (category: number | undefined): boolean =>
  category === 2 || category === 5;

// Utility functions
const getStatusFromRunningState = (runningState: number): MCPServerConfig['status'] =>
  STATUS_MAP[runningState] || 'connect_failed';

const getServerTypeFromToolType = (toolType: number): string => TOOL_TYPE_MAP[toolType] || 'github';

const OAUTH_CODE_VERIFIER_KEY = 'YOUR_OAUTH_PKCE_VERIFIER';
const DEFAULT_LOCAL_OAUTH_PORT = '3000';
const USER_MODE_LOCALHOST_REDIRECT_URI = 'http://localhost';
const USER_MODE_CANVA_REDIRECT_URI = 'http://127.0.0.1:34327';

const buildCredentialFieldKey = (credential: Credential, index: number): string =>
  `credential_${index}_${credential.name.toLowerCase().replace(/\s+/g, '_')}`;

const getCredentialValueByTemplateKey = (
  credentials: Credential[] | undefined,
  dynamicCredentials: Record<string, string>,
  targetKey: string,
): string | undefined => {
  if (!credentials?.length) return undefined;

  for (let i = 0; i < credentials.length; i++) {
    const credential = credentials[i];
    if (credential.key !== targetKey) continue;

    const value = dynamicCredentials[buildCredentialFieldKey(credential, i)]?.trim();
    if (!value || value === targetKey) return undefined;
    return value;
  }

  return undefined;
};

const hasOAuthEndpointCredentialKeys = (template: Tool | null | undefined): boolean =>
  Boolean(
    template?.credentials?.some((credential) =>
      OAUTH_ENDPOINT_OVERRIDE_KEYS.includes(
        credential.key as (typeof OAUTH_ENDPOINT_OVERRIDE_KEYS)[number],
      ),
    ),
  );

const requiresCustomOAuthApp = (template: Tool | null | undefined): boolean =>
  Boolean(template?.oAuthConfig?.requiresCustomApp || hasOAuthEndpointCredentialKeys(template));

const getOAuthEndpointOverridesFromCredentials = (
  template: Tool,
  dynamicCredentials: Record<string, string>,
): OAuthEndpointOverrides => ({
  authorizationUrl: getCredentialValueByTemplateKey(
    template.credentials,
    dynamicCredentials,
    OAUTH_AUTHORIZATION_URL_KEY,
  ),
  tokenUrl: getCredentialValueByTemplateKey(
    template.credentials,
    dynamicCredentials,
    OAUTH_TOKEN_URL_KEY,
  ),
  baseUrl: getCredentialValueByTemplateKey(
    template.credentials,
    dynamicCredentials,
    OAUTH_BASE_URL_KEY,
  ),
});

const DynamicIcon = React.memo(
  ({ iconName, className }: { iconName: string; className?: string }) => {
    const IconComponent = ICON_MAP[iconName] || Database;
    return <IconComponent className={className} />;
  },
);
DynamicIcon.displayName = 'DynamicIcon';

// Custom Hook: Server info fetching
const useServerInfo = () => {
  const [proxyId, setProxyId] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);

  const getServerInfo = useCallback(async () => {
    if (proxyId) return proxyId; // Cached result

    try {
      setLoading(true);
      const response = await api.servers.getInfo();
      const id = response.data?.data?.proxyId;
      setProxyId(id);
      return id;
    } catch (error) {
      // Failed to get server info:
      return null;
    } finally {
      setLoading(false);
    }
  }, [proxyId]);

  return { proxyId, getServerInfo, loading };
};

// Custom Hook: Server capabilities fetching
const useServerCapabilities = (toolId?: string) => {
  const [capabilities, setCapabilities] = useState<ServerCapabilities>({});
  const [loading, setLoading] = useState(false);

  const loadCapabilities = useCallback(async () => {
    if (!toolId) return;

    try {
      setLoading(true);
      const response = await api.tools.getServerCapabilities({ toolId });

      if (response.data?.data) {
        setCapabilities(response.data.data);
      }
    } catch (error) {
      // Failed to load server capabilities:
    } finally {
      setLoading(false);
    }
  }, [toolId]);

  useEffect(() => {
    loadCapabilities();
  }, [loadCapabilities]);

  return { capabilities, loading, refetch: loadCapabilities };
};

// CredentialsTab props interface
interface CredentialsTabProps {
  server: MCPServerConfig;
  onUpdate: (server: MCPServerConfig) => void;
  toolTemplate?: Tool;
  dynamicCredentials: Record<string, string>;
  onCredentialUpdate: (key: string, value: string) => void;
  onDelete?: () => void;
  isEdit: boolean;
  allowUserInputMode: boolean;
  lazyStartEnabled: boolean;
  onLazyStartEnabledChange: (value: boolean) => void;
  publicAccess: boolean;
  onPublicAccessChange: (value: boolean) => void;
  anonymousAccess: boolean;
  onAnonymousAccessChange: (value: boolean) => void;
  anonymousRateLimit: number;
  onAnonymousRateLimitChange: (value: number) => void;
  configMode: 'owner' | 'user';
  onConfigModeChange?: (mode: 'owner' | 'user') => void;
  credentialSource: 'kimbap' | 'custom';
  onCredentialSourceChange?: (source: 'kimbap' | 'custom') => void;
  isSetupMode?: boolean;
}

// CredentialsTab component with configMode and credentialSource
const CredentialsTab = React.memo<CredentialsTabProps>(
  ({
    server,
    onUpdate,
    toolTemplate,
    dynamicCredentials,
    onCredentialUpdate,
    onDelete,
    isEdit,
    allowUserInputMode,
    lazyStartEnabled,
    onLazyStartEnabledChange,
    publicAccess,
    onPublicAccessChange,
    anonymousAccess,
    onAnonymousAccessChange,
    anonymousRateLimit,
    onAnonymousRateLimitChange,
    configMode,
    onConfigModeChange,
    credentialSource,
    onCredentialSourceChange,
    isSetupMode = false,
  }) => {
    const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({});
    const requiresCustomApp = requiresCustomOAuthApp(toolTemplate);
    const isCanvaAuth = toolTemplate?.authType === ServerAuthType.CanvaAuth;
    const currentLocation = typeof window !== 'undefined' ? window.location : null;
    const currentPort = currentLocation?.port || DEFAULT_LOCAL_OAUTH_PORT;
    const currentOrigin = currentLocation?.origin || `http://localhost:${currentPort}`;
    const currentHostname = currentLocation?.hostname || '';
    const canvaPreferredBaseUrl = `http://127.0.0.1:${currentPort}`;
    const shouldWarnCanvaHostAccess =
      isCanvaAuth && currentHostname !== '' && currentHostname !== '127.0.0.1';

    const customOwnerRedirectUri = isCanvaAuth
      ? `${canvaPreferredBaseUrl}/dashboard/tool-configure`
      : `${currentOrigin}/dashboard/tool-configure`;
    const customRedirectUriGuide =
      configMode === 'owner'
        ? customOwnerRedirectUri
        : isCanvaAuth
          ? USER_MODE_CANVA_REDIRECT_URI
          : USER_MODE_LOCALHOST_REDIRECT_URI;

    const kimbapOwnerRedirectUri = `http://${isCanvaAuth ? '127.0.0.1' : 'localhost'}:${currentPort}/dashboard/tool-configure`;
    const kimbapRedirectUriGuide =
      configMode === 'owner'
        ? kimbapOwnerRedirectUri
        : isCanvaAuth
          ? USER_MODE_CANVA_REDIRECT_URI
          : USER_MODE_LOCALHOST_REDIRECT_URI;

    const toggleSecret = useCallback((field: string) => {
      setShowSecrets((prev) => ({ ...prev, [field]: !prev[field] }));
    }, []);

    const handleCredentialChange = useCallback(
      (key: string, value: any) => {
        const updatedServer = {
          ...server,
          credentials: {
            ...server.credentials,
            [key]: typeof value === 'string' ? value : { encrypted: true, value },
          },
        };
        onUpdate(updatedServer);
      },
      [server, onUpdate],
    );

    // Dynamic field renderer
    const renderCredentialField = useCallback(
      (credential: Credential, index: number) => {
        const { name, description, dataType, key, options } = credential;
        const fieldKey = buildCredentialFieldKey(credential, index);
        const fieldValue =
          fieldKey in dynamicCredentials ? dynamicCredentials[fieldKey] : key || '';

        const handleChange = (value: string) => {
          onCredentialUpdate(fieldKey, value);
          handleCredentialChange(fieldKey, value);
        };

        const isSecret = ['token', 'key', 'secret', 'password', 'api'].some((keyword) =>
          name.toLowerCase().includes(keyword),
        );

        const commonProps = {
          id: fieldKey,
          value: fieldValue,
          onChange: (e: React.ChangeEvent<HTMLInputElement>) => handleChange(e.target.value),
        };

        switch (dataType) {
          case 1: // Input field
            return (
              <div key={fieldKey} className="space-y-2">
                <Label htmlFor={fieldKey}>{name}</Label>
                <div className="relative">
                  <Input
                    {...commonProps}
                    type={isSecret && !showSecrets[fieldKey] ? 'password' : 'text'}
                    placeholder={description || `Enter ${name}`}
                    value={
                      isSecret && !showSecrets[fieldKey]
                        ? '*'.repeat(Math.min(fieldValue.length, 24))
                        : fieldValue
                    }
                  />
                  {isSecret && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="absolute right-2 top-1/2 -translate-y-1/2"
                      onClick={() => toggleSecret(fieldKey)}
                      aria-label={showSecrets[fieldKey] ? `Hide ${name}` : `Show ${name}`}
                    >
                      {showSecrets[fieldKey] ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </Button>
                  )}
                </div>
                {description && <p className="text-xs text-muted-foreground">{description}</p>}
              </div>
            );

          case 2: // Radio buttons
            return (
              <div key={fieldKey} className="space-y-2">
                <Label>{name}</Label>
                <div className="space-y-2">
                  {options.map((option) => (
                    <label key={option.key} className="flex items-center">
                      <input
                        type="radio"
                        name={fieldKey}
                        value={option.value}
                        checked={fieldValue === option.value}
                        onChange={(e) => handleChange(e.target.value)}
                        className="text-blue-600 dark:text-blue-400"
                      />
                      <span className="text-sm ml-2">{option.key}</span>
                    </label>
                  ))}
                </div>
                {description && <p className="text-xs text-muted-foreground">{description}</p>}
              </div>
            );

          case 3: // Checkbox
            return (
              <div key={fieldKey} className="space-y-2">
                <label className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    checked={fieldValue === 'true'}
                    onChange={(e) => handleChange(e.target.checked ? 'true' : 'false')}
                    className="text-blue-600 dark:text-blue-400"
                  />
                  <span className="text-sm font-medium">{name}</span>
                </label>
                {description && <p className="text-xs text-muted-foreground">{description}</p>}
              </div>
            );

          case 4: // Select dropdown
            return (
              <div key={fieldKey} className="space-y-2">
                <Label htmlFor={fieldKey}>{name}</Label>
                <select
                  id={fieldKey}
                  value={fieldValue}
                  onChange={(e) => handleChange(e.target.value)}
                  className="w-full p-2 border border-input bg-background rounded-md text-sm"
                >
                  <option value="">Select {name}</option>
                  {options.map((option) => (
                    <option key={option.key} value={option.value}>
                      {option.key}
                    </option>
                  ))}
                </select>
                {description && <p className="text-xs text-muted-foreground">{description}</p>}
              </div>
            );

          default:
            return (
              <div key={fieldKey} className="space-y-2">
                <Label htmlFor={fieldKey}>{name}</Label>
                <Input {...commonProps} placeholder={description || `Enter ${name}`} />
                {description && <p className="text-xs text-muted-foreground">{description}</p>}
              </div>
            );
        }
      },
      [dynamicCredentials, showSecrets, onCredentialUpdate, handleCredentialChange, toggleSecret],
    );

    const credentialFields = useMemo(() => {
      if (!toolTemplate?.credentials?.length) {
        return (
          <div className="text-center py-8 text-muted-foreground">
            <p>
              Template data is out of date. Return to Tool Configuration, refresh, and start the
              setup again.
            </p>
          </div>
        );
      }

      return (
        <div className="space-y-4">
          {toolTemplate.credentials.map(renderCredentialField)}
          {toolTemplate?.applyUrl && (
            <div className="flex items-center gap-2 p-3 bg-muted/50 rounded-lg">
              <Info className="w-4 h-4 text-muted-foreground flex-shrink-0" />
              <span className="text-sm text-muted-foreground">Don't have credentials?</span>
              <a
                href={toolTemplate.applyUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium inline-flex items-center gap-1 hover:underline"
              >
                Get credentials
                <ExternalLink className="w-3.5 h-3.5" />
              </a>
            </div>
          )}
        </div>
      );
    }, [toolTemplate?.credentials, toolTemplate?.applyUrl, renderCredentialField, isEdit]);

    // Setup mode: show configuration mode selection
    if (isSetupMode) {
      return (
        <div className="space-y-6">
          {/* Configuration Mode Selection */}
          <div className="space-y-2">
            <Label className="text-sm font-medium">Configuration Mode</Label>
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => onConfigModeChange?.('owner')}
                aria-pressed={configMode === 'owner'}
                className={cn(
                  'flex items-center gap-2 p-3 rounded-lg border-2 transition-all text-left',
                  configMode === 'owner'
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-border dark:border-gray-600 hover:border-muted-foreground/30 dark:hover:border-gray-500',
                )}
              >
                <Building2
                  className={cn(
                    'w-5 h-5',
                    configMode === 'owner'
                      ? 'text-blue-600 dark:text-blue-400'
                      : 'text-muted-foreground',
                  )}
                />
                <div>
                  <div
                    className={cn(
                      'font-medium text-sm',
                      configMode === 'owner'
                        ? 'text-blue-900 dark:text-blue-100'
                        : 'text-foreground',
                    )}
                  >
                    Owner
                  </div>
                  <div className="text-xs text-muted-foreground">Admin configuration for all</div>
                </div>
              </button>
              <button
                type="button"
                onClick={() => onConfigModeChange?.('user')}
                aria-pressed={configMode === 'user'}
                className={cn(
                  'flex items-center gap-2 p-3 rounded-lg border-2 transition-all text-left',
                  configMode === 'user'
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-border dark:border-gray-600 hover:border-muted-foreground/30 dark:hover:border-gray-500',
                )}
              >
                <User
                  className={cn(
                    'w-5 h-5',
                    configMode === 'user'
                      ? 'text-blue-600 dark:text-blue-400'
                      : 'text-muted-foreground',
                  )}
                />
                <div>
                  <div
                    className={cn(
                      'font-medium text-sm',
                      configMode === 'user'
                        ? 'text-blue-900 dark:text-blue-100'
                        : 'text-foreground',
                    )}
                  >
                    User
                  </div>
                  <div className="text-xs text-muted-foreground">Each user configuration</div>
                </div>
              </button>
            </div>
          </div>

          {/* App Credentials Selection - only show for Owner mode with OAuth config */}
          {toolTemplate?.oAuthConfig && (
            <div className="space-y-2">
              <Label className="text-sm font-medium">App Credentials</Label>
              <div className="grid grid-cols-2 gap-2">
                <button
                  type="button"
                  onClick={() => onCredentialSourceChange?.('kimbap')}
                  aria-pressed={credentialSource === 'kimbap'}
                  className={cn(
                    'flex items-center gap-2 p-3 rounded-lg border-2 transition-all text-left',
                    credentialSource === 'kimbap'
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-border dark:border-gray-600 hover:border-muted-foreground/30 dark:hover:border-gray-500',
                  )}
                >
                  <div>
                    <div
                      className={cn(
                        'font-medium text-sm',
                        credentialSource === 'kimbap'
                          ? 'text-green-900 dark:text-green-100'
                          : 'text-foreground',
                      )}
                    >
                      Use Kimbap's App
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Quick setup, managed by Kimbap
                    </div>
                  </div>
                </button>
                <button
                  type="button"
                  onClick={() => onCredentialSourceChange?.('custom')}
                  aria-pressed={credentialSource === 'custom'}
                  className={cn(
                    'flex items-center gap-2 p-3 rounded-lg border-2 transition-all text-left',
                    credentialSource === 'custom'
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-border dark:border-gray-600 hover:border-muted-foreground/30 dark:hover:border-gray-500',
                  )}
                >
                  <div>
                    <div
                      className={cn(
                        'font-medium text-sm',
                        credentialSource === 'custom'
                          ? 'text-purple-900 dark:text-purple-100'
                          : 'text-foreground',
                      )}
                    >
                      Use Your Own App
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Full control with your credentials
                    </div>
                  </div>
                </button>
              </div>
              {requiresCustomApp && (
                <p className="text-xs text-amber-700 dark:text-amber-300">
                  This integration requires your own OAuth app credentials.
                </p>
              )}
            </div>
          )}

          {/* Credential fields - only show when custom mode selected in Owner mode */}
          {toolTemplate?.oAuthConfig && credentialSource === 'custom' && (
            <>
              {/* Benefits Section */}
              <div className="bg-blue-50 dark:bg-blue-950/50 border border-blue-100 dark:border-blue-800 rounded-lg p-4">
                <div className="flex items-start gap-2 mb-2">
                  <ShieldCheck className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                  <div>
                    <h4 className="font-medium text-blue-900 dark:text-blue-100 text-sm">
                      Benefits of using your own credentials
                    </h4>
                  </div>
                </div>
                <ul className="space-y-1.5 ml-7">
                  <li className="text-sm text-blue-800 dark:text-blue-200 flex items-start gap-2">
                    <span className="text-blue-400 dark:text-blue-500 mt-1">•</span>
                    <span>Full control over OAuth permissions and scopes</span>
                  </li>
                  <li className="text-sm text-blue-800 dark:text-blue-200 flex items-start gap-2">
                    <span className="text-blue-400 dark:text-blue-500 mt-1">•</span>
                    <span>Access logs belong entirely to you</span>
                  </li>
                  <li className="text-sm text-blue-800 dark:text-blue-200 flex items-start gap-2">
                    <span className="text-blue-400 dark:text-blue-500 mt-1">•</span>
                    <span>Customize OAuth consent screen branding</span>
                  </li>
                </ul>
              </div>

              {/* Apply URL link */}
              {toolTemplate?.oAuthConfig?.applyUrl && (
                <div className="flex items-center gap-2 p-3 bg-muted/50 rounded-lg">
                  <Info className="w-4 h-4 text-muted-foreground flex-shrink-0" />
                  <span className="text-sm text-muted-foreground">Don't have credentials?</span>
                  <a
                    href={toolTemplate.oAuthConfig.applyUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium inline-flex items-center gap-1 hover:underline"
                  >
                    Apply here
                    <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                </div>
              )}

              <div className="bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-800 rounded-lg p-3 space-y-2">
                <p className="text-sm font-medium text-amber-900 dark:text-amber-100">
                  OAuth Redirect URI reminder
                </p>
                <p className="text-xs text-amber-800 dark:text-amber-200">
                  Register this redirect URI in your OAuth client settings. Authorization can fail
                  if it does not match.
                </p>
                <p className="text-xs font-mono break-all text-amber-900 dark:text-amber-100">
                  {customRedirectUriGuide}
                </p>
                {isCanvaAuth && (
                  <p
                    className={cn(
                      'text-xs',
                      shouldWarnCanvaHostAccess
                        ? 'text-amber-900 dark:text-amber-100'
                        : 'text-amber-800 dark:text-amber-200',
                    )}
                  >
                    Canva requires 127.0.0.1 callback domains. Open this page via{' '}
                    {canvaPreferredBaseUrl} before authorizing.
                  </p>
                )}
              </div>

              {/* Credential input fields */}
              <div className="border dark:border-gray-700 rounded-lg p-3">{credentialFields}</div>
            </>
          )}

          {/* Kimbap's App hint */}
          {credentialSource === 'kimbap' && toolTemplate?.oAuthConfig && !requiresCustomApp && (
            <div className="bg-green-50 dark:bg-green-950/50 border border-green-100 dark:border-green-800 rounded-lg p-4">
              <div className="flex items-start gap-2">
                <ShieldCheck className="w-5 h-5 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
                <div>
                  <h4 className="font-medium text-green-900 dark:text-green-100 text-sm mb-1">
                    Using Kimbap&apos;s Managed Service
                  </h4>
                  <p className="text-sm text-green-800 dark:text-green-200">
                    Kimbap will securely manage the {toolTemplate?.name} connection credentials for
                    you. Get started quickly without any configuration.
                  </p>
                  <div className="mt-3 space-y-2">
                    <p className="text-xs text-green-800 dark:text-green-200">
                      Kimbap&apos;s configured redirect URI for this mode:
                    </p>
                    <p className="text-xs font-mono break-all text-green-900 dark:text-green-100">
                      {kimbapRedirectUriGuide}
                    </p>
                    <p className="text-xs text-green-800 dark:text-green-200">
                      Use localhost to access this page (for Canva, use 127.0.0.1).
                    </p>
                    {isCanvaAuth && (
                      <p
                        className={cn(
                          'text-xs',
                          shouldWarnCanvaHostAccess
                            ? 'text-green-900 dark:text-green-100'
                            : 'text-green-800 dark:text-green-200',
                        )}
                      >
                        Canva requires 127.0.0.1 callback domains. Open this page via{' '}
                        {canvaPreferredBaseUrl} before authorizing.
                      </p>
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Non-OAuth tools: show credential fields directly */}
          {!toolTemplate?.oAuthConfig && (
            <div className="border dark:border-gray-700 rounded-lg p-3">{credentialFields}</div>
          )}

          <PublicAccessConfiguration
            checked={publicAccess}
            onCheckedChange={onPublicAccessChange}
          />

          {!allowUserInputMode && configMode !== 'user' && (
            <AnonymousAccessConfiguration
              checked={anonymousAccess}
              onCheckedChange={onAnonymousAccessChange}
              rateLimit={anonymousRateLimit}
              onRateLimitChange={onAnonymousRateLimitChange}
            />
          )}

          <LazyStartConfiguration
            checked={lazyStartEnabled}
            onCheckedChange={onLazyStartEnabledChange}
          />
        </div>
      );
    }

    // Edit mode: show standard credential form
    return (
      <div className="space-y-4">
        {/* Console Authentication */}
        {(!isEdit ||
          (server.authType === ServerAuthType.ApiKey && server.allowUserInput === 0)) && (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Settings className="h-4 w-4" />
                <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
                  Console Authentication
                </h3>
              </div>
            </div>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              Configure shared credentials for all users
            </p>
            <div className="border dark:border-gray-700 rounded-lg p-3">{credentialFields}</div>
          </div>
        )}

        <PublicAccessConfiguration checked={publicAccess} onCheckedChange={onPublicAccessChange} />

        {!allowUserInputMode && configMode !== 'user' && (
          <AnonymousAccessConfiguration
            checked={anonymousAccess}
            onCheckedChange={onAnonymousAccessChange}
            rateLimit={anonymousRateLimit}
            onRateLimitChange={onAnonymousRateLimitChange}
          />
        )}

        <LazyStartConfiguration
          checked={lazyStartEnabled}
          onCheckedChange={onLazyStartEnabledChange}
        />

        {/* Delete Tool Section */}
        {onDelete && (
          <div className="border-t dark:border-gray-700 pt-4 mt-6">
            <Button
              variant="outline"
              size="sm"
              onClick={onDelete}
              className="text-red-600 hover:text-red-700 hover:bg-red-50 border-red-200 dark:text-red-400 dark:hover:text-red-300 dark:hover:bg-red-900/20 dark:border-red-800"
            >
              Delete Tool
            </Button>
          </div>
        )}
      </div>
    );
  },
);
CredentialsTab.displayName = 'CredentialsTab';

// FunctionsTab component optimization
interface FunctionsTabProps {
  server: MCPServerConfig;
  onUpdate: (server: MCPServerConfig) => void;
  toolTemplate?: Tool;
  onAutoSave?: (functions: any[]) => void;
  cacheHealth?: CacheHealthState | null;
  cachePolicies?: CachePolicyState;
  onToolCachePolicyChange?: (toolName: string, policy: CachePolicyConfig) => void;
  onPromptCachePolicyChange?: (promptName: string, policy: CachePolicyConfig) => void;
  onResourceCachePolicyChange?: (uri: string, policy: CachePolicyConfig) => void;
  onRefreshCacheData?: () => void;
  isLoadingCache?: boolean;
}

interface ProcessedFunction {
  id: string;
  name: string;
  description: string;
  category: string;
  enabled?: boolean;
  dangerLevel?: number;
  executionMode?: 0 | 1 | 2; // dangerLevel: 0=no verification, 1=prompt only (reserved), 2=must verify
}

// Execution mode description
const getModeDescription = (mode: 0 | 1 | 2): string => {
  switch (mode) {
    case 0:
      return 'Function executes automatically without user notification';
    case 1:
      return 'Function executes automatically and displays result to user';
    case 2:
      return 'User must manually approve before function execution';
    default:
      return '';
  }
};

const FunctionsTab = React.memo<FunctionsTabProps>(
  ({
    server,
    onUpdate,
    toolTemplate,
    onAutoSave,
    cacheHealth,
    cachePolicies,
    onToolCachePolicyChange,
    onPromptCachePolicyChange,
    onResourceCachePolicyChange,
    onRefreshCacheData,
    isLoadingCache,
  }) => {
    const [searchQuery, setSearchQuery] = useState('');
    const [selectedFunctions, setSelectedFunctions] = useState<Set<string>>(
      new Set(server.enabledFunctions),
    );
    const [functionModes, setFunctionModes] = useState<Record<string, 0 | 1 | 2>>(() => {
      const modes: Record<string, 0 | 1 | 2> = {};
      server.enabledFunctions?.forEach((func) => {
        modes[func] = 0; // Default to 'Execute Silently'
      });
      return modes;
    });
    const autoSaveTimeoutRef = useRef<NodeJS.Timeout | null>(null);

    // Use ref to save latest state, avoid closure issues
    const selectedFunctionsRef = useRef(selectedFunctions);
    const functionModesRef = useRef(functionModes);

    // Update ref when state changes
    useEffect(() => {
      selectedFunctionsRef.current = selectedFunctions;
    }, [selectedFunctions]);

    useEffect(() => {
      functionModesRef.current = functionModes;
    }, [functionModes]);

    const { capabilities, loading: isLoadingCapabilities } = useServerCapabilities(server?.id);

    // Process server function data
    const processedFunctions = useMemo((): ProcessedFunction[] => {
      const serverFunctions = capabilities?.functions || [];

      if (serverFunctions.length === 0) return [];

      return serverFunctions.map((func) => ({
        id: func.funcName || func.name || '',
        name: func.funcName || func.name || '',
        description: func.description || `${func.funcName || func.name} function`,
        category: 'Server Functions',
        enabled: func.enabled,
        dangerLevel: func.dangerLevel,
      }));
    }, [capabilities?.functions]);

    // Update selected functions state (only on initial load)
    const [isInitialized, setIsInitialized] = useState(false);

    useEffect(() => {
      if (capabilities?.functions?.length && !isInitialized) {
        // Use enabled state from capabilities.functions for initialization, this is the actual state from protocol 10010
        const enabledFromCapabilities = new Set(
          capabilities.functions.filter((f) => f.enabled === true).map((f) => f.funcName),
        );

        // If there is complete function config info, also restore dangerLevel
        const modesFromCapabilities: Record<string, 0 | 1 | 2> = {};
        capabilities.functions.forEach((f) => {
          if (f.enabled && f.dangerLevel !== undefined) {
            modesFromCapabilities[f.funcName] = f.dangerLevel as 0 | 1 | 2;
          }
        });

        // Update all related states
        setSelectedFunctions(enabledFromCapabilities);
        selectedFunctionsRef.current = enabledFromCapabilities;
        setFunctionModes(modesFromCapabilities);
        functionModesRef.current = modesFromCapabilities;
        setIsInitialized(true);

        // Update parent component state, use actual state from capabilities
        const allFunctionsWithStatus = capabilities.functions.map((f) => ({
          funcName: f.funcName,
          enabled: enabledFromCapabilities.has(f.funcName), // Use actual state from capabilities
          dangerLevel: modesFromCapabilities[f.funcName] || 0,
          description: f.description || '',
        }));

        const updatedServer = {
          ...server,
          enabledFunctions: Array.from(enabledFromCapabilities),
          allFunctions: allFunctionsWithStatus,
        };

        onUpdate(updatedServer);
      }
    }, [capabilities?.functions?.length, server.id, onUpdate, functionModes, isInitialized]);

    // Remove conflicting sync logic, handled directly by handleFunctionToggle and handleModeChange

    // Group functions by category
    const functionsByCategory = useMemo(() => {
      return processedFunctions.reduce(
        (acc, func) => {
          if (!acc[func.category]) {
            acc[func.category] = [];
          }
          acc[func.category].push(func);
          return acc;
        },
        {} as Record<string, ProcessedFunction[]>,
      );
    }, [processedFunctions]);

    // Filter functions
    const filteredFunctionsByCategory = useMemo(() => {
      if (!searchQuery.trim()) return functionsByCategory;

      return Object.entries(functionsByCategory).reduce(
        (acc, [category, funcs]) => {
          const filtered = funcs.filter(
            (func) =>
              func.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
              func.description.toLowerCase().includes(searchQuery.toLowerCase()),
          );
          if (filtered.length > 0) {
            acc[category] = filtered;
          }
          return acc;
        },
        {} as Record<string, ProcessedFunction[]>,
      );
    }, [functionsByCategory, searchQuery]);

    // Debounced auto-save function - Use ref to get latest state
    const debouncedAutoSave = useCallback(() => {
      // Only auto-save if onAutoSave is provided (modification mode)
      if (!onAutoSave || !capabilities?.functions) return;

      if (autoSaveTimeoutRef.current) {
        clearTimeout(autoSaveTimeoutRef.current);
      }

      autoSaveTimeoutRef.current = setTimeout(async () => {
        try {
          // Use ref to get latest state values
          const currentSelectedFunctions = selectedFunctionsRef.current;
          const currentFunctionModes = functionModesRef.current;

          // Build functions array from current state and pass to parent
          const functionsForAPI =
            capabilities.functions?.map((f) => ({
              funcName: f.funcName,
              enabled: currentSelectedFunctions.has(f.funcName),
              dangerLevel: currentFunctionModes[f.funcName] || 0,
              description: f.description || '',
            })) || [];

          await onAutoSave(functionsForAPI);
        } catch (error) {
          // Auto-save failed:
        }
      }, 1000); // 1 second debounce
    }, [onAutoSave, capabilities?.functions]);

    const handleFunctionToggle = useCallback(
      (functionId: string) => {
        const newSelected = new Set(selectedFunctions);
        let newModes = { ...functionModes };

        if (newSelected.has(functionId)) {
          newSelected.delete(functionId);
          delete newModes[functionId];
        } else {
          newSelected.add(functionId);
          newModes[functionId] = 0; // Default to 'Execute Silently'
        }

        // Update local state
        setSelectedFunctions(newSelected);
        setFunctionModes(newModes);

        // If in creation flow (no onAutoSave), update parent state for Next button
        if (!onAutoSave && capabilities?.functions) {
          const allFunctionsWithStatus = capabilities.functions.map((f) => ({
            funcName: f.funcName,
            enabled: newSelected.has(f.funcName),
            dangerLevel: newModes[f.funcName] || 0,
            description: f.description || '',
          }));

          onUpdate({
            ...server,
            enabledFunctions: Array.from(newSelected),
            allFunctions: allFunctionsWithStatus,
          });
        } else {
          // Trigger debounced auto-save for edit mode
          debouncedAutoSave();
        }
      },
      [
        selectedFunctions,
        functionModes,
        debouncedAutoSave,
        onAutoSave,
        capabilities?.functions,
        onUpdate,
        server,
      ],
    );

    const handleModeChange = useCallback(
      (functionId: string, mode: 0 | 1 | 2) => {
        const newModes = { ...functionModes, [functionId]: mode };
        setFunctionModes(newModes);

        // If in creation flow (no onAutoSave), update parent state for Next button
        if (!onAutoSave && capabilities?.functions) {
          const allFunctionsWithStatus = capabilities.functions.map((f) => ({
            funcName: f.funcName,
            enabled: selectedFunctions.has(f.funcName),
            dangerLevel: newModes[f.funcName] || 0,
            description: f.description || '',
          }));

          onUpdate({
            ...server,
            enabledFunctions: Array.from(selectedFunctions),
            allFunctions: allFunctionsWithStatus,
          });
        } else {
          // Trigger debounced auto-save for edit mode
          debouncedAutoSave();
        }
      },
      [
        functionModes,
        selectedFunctions,
        debouncedAutoSave,
        onAutoSave,
        capabilities?.functions,
        onUpdate,
        server,
      ],
    );

    // Cleanup timeout on unmount
    useEffect(() => {
      return () => {
        if (autoSaveTimeoutRef.current) {
          clearTimeout(autoSaveTimeoutRef.current);
        }
      };
    }, []);

    const handleCategoryToggle = useCallback(
      (category: string) => {
        const categoryFunctions = functionsByCategory[category];
        const categoryFunctionIds = categoryFunctions.map((f) => f.id);
        const allSelected = categoryFunctionIds.every((id) => selectedFunctions.has(id));

        const newSelected = new Set(selectedFunctions);
        if (allSelected) {
          categoryFunctionIds.forEach((id) => newSelected.delete(id));
        } else {
          categoryFunctionIds.forEach((id) => newSelected.add(id));
        }

        setSelectedFunctions(newSelected);
      },
      [selectedFunctions, functionsByCategory],
    );

    const getCategorySelectionState = useCallback(
      (category: string) => {
        const categoryFunctions = functionsByCategory[category];
        const categoryFunctionIds = categoryFunctions.map((f) => f.id);
        const selectedCount = categoryFunctionIds.filter((id) => selectedFunctions.has(id)).length;

        if (selectedCount === 0) return 'none';
        if (selectedCount === categoryFunctionIds.length) return 'all';
        return 'some';
      },
      [functionsByCategory, selectedFunctions],
    );

    if (isLoadingCapabilities) {
      return (
        <Card>
          <CardContent className="pt-6 text-center">
            <div className="flex items-center justify-center">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
              <span className="ml-3 text-muted-foreground">Loading server capabilities...</span>
            </div>
          </CardContent>
        </Card>
      );
    }

    if (processedFunctions.length === 0) {
      return (
        <Card>
          <CardContent className="pt-6 text-center">
            <p className="text-muted-foreground">No functions available</p>
          </CardContent>
        </Card>
      );
    }

    return (
      <TooltipProvider delayDuration={300}>
        <div className="space-y-4">
          <div className="space-y-3 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
                  Result Cache
                </h3>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                  Configure per-capability result caching and purge stale cache entries.
                </p>
                {isLoadingCache && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Loading cache policy...
                  </p>
                )}
              </div>
              <div className="flex items-center gap-2 text-xs">
                <span className="rounded-full border px-2 py-0.5 text-gray-600 dark:text-gray-300">
                  Backend: {cacheHealth?.health?.backend || 'unknown'}
                </span>
                <span className="rounded-full border px-2 py-0.5 text-gray-600 dark:text-gray-300">
                  Status: {cacheHealth?.health?.ok ? 'healthy' : 'unknown'}
                </span>
              </div>
            </div>

            <div className="space-y-3">
              {processedFunctions.map((func) => (
                <CachePolicyEditor
                  key={`cache-tool-${func.id}`}
                  entityType="tool"
                  entityName={func.name}
                  cachePolicy={cachePolicies?.tools?.[func.id]}
                  isDangerous={(func.dangerLevel ?? functionModes[func.id] ?? 0) === 2}
                  globalCacheEnabled={cacheHealth?.enabled}
                  onPolicyChange={(policy) => onToolCachePolicyChange?.(func.id, policy)}
                />
              ))}
              {Object.entries(cachePolicies?.prompts || {}).map(([promptName, policy]) => (
                <CachePolicyEditor
                  key={`cache-prompt-${promptName}`}
                  entityType="prompt"
                  entityName={promptName}
                  cachePolicy={policy}
                  globalCacheEnabled={cacheHealth?.enabled}
                  onPolicyChange={(next) => onPromptCachePolicyChange?.(promptName, next)}
                />
              ))}
              {Object.entries(cachePolicies?.resources?.exact || {}).map(([uri, policy]) => (
                <CachePolicyEditor
                  key={`cache-resource-${uri}`}
                  entityType="resource"
                  entityName={uri}
                  cachePolicy={policy}
                  globalCacheEnabled={cacheHealth?.enabled}
                  onPolicyChange={(next) => onResourceCachePolicyChange?.(uri, next)}
                />
              ))}
            </div>

            <CachePurgePanel serverId={server.id} onPurgeComplete={onRefreshCacheData} />
          </div>

          {/* Header with count and search */}
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-xl font-semibold">
                Functions - {selectedFunctions.size}/{processedFunctions.length} enabled
              </h2>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                Start/stop server and manage basic configuration.
              </p>
            </div>
          </div>

          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 dark:text-gray-500" />
            <Input
              aria-label="Search functions"
              placeholder="Search functions..."
              className="pl-10 bg-gray-50 dark:bg-gray-800"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>

          {/* Functions list */}
          <div className="space-y-2">
            {Object.entries(filteredFunctionsByCategory).map(([category, categoryFunctions]) => (
              <div key={category} className="space-y-2">
                {categoryFunctions.map((func) => {
                  const isSelected = selectedFunctions.has(func.id);
                  const mode = (functionModes[func.id] ?? 0) as 0 | 1 | 2;

                  return (
                    <div
                      key={func.id}
                      className="flex items-center justify-between p-4 bg-white dark:bg-gray-800 border dark:border-gray-700 rounded-lg hover:shadow-sm transition-shadow"
                    >
                      <div className="flex items-start gap-3 flex-1">
                        <input
                          type="checkbox"
                          aria-label={`Select function ${func.name}`}
                          checked={isSelected}
                          onChange={() => handleFunctionToggle(func.id)}
                          className="mt-1 h-4 w-4 text-blue-600 dark:text-blue-400 rounded border-gray-300 dark:border-gray-600 focus:ring-blue-500"
                        />
                        <div className="flex-1">
                          <div className="font-medium text-sm text-gray-900 dark:text-white">
                            {func.name}
                          </div>
                          <div className="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                            <ReactMarkdown className="prose prose-sm max-w-none dark:prose-invert">
                              {func.description}
                            </ReactMarkdown>
                          </div>
                        </div>
                      </div>

                      {isSelected && (
                        <div className="ml-4 relative">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <select
                                aria-label={`Approval mode for ${func.name}`}
                                value={mode.toString()}
                                onChange={(e) => {
                                  const selectedValue = parseInt(e.target.value);
                                  if (selectedValue !== -1) {
                                    handleModeChange(func.id, selectedValue as 0 | 1 | 2);
                                  }
                                }}
                                className="inline-block px-3 py-1.5 text-sm border rounded-md bg-white dark:bg-gray-800 dark:border-gray-600 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                                title={getModeDescription(mode)} // Fallback for browsers that don't support advanced tooltips
                              >
                                <option value="0">Always allow</option>
                                <option value="1">Approval without Password</option>
                                <option value="2">Approval with Password</option>
                              </select>
                            </TooltipTrigger>
                            <TooltipContent side="top" sideOffset={5}>
                              <p className="text-sm max-w-xs">{getModeDescription(mode)}</p>
                            </TooltipContent>
                          </Tooltip>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            ))}
          </div>

          {Object.keys(filteredFunctionsByCategory).length === 0 && (
            <Card>
              <CardContent className="pt-6 text-center">
                <p className="text-muted-foreground">No functions match "{searchQuery}"</p>
              </CardContent>
            </Card>
          )}
        </div>
      </TooltipProvider>
    );
  },
);
FunctionsTab.displayName = 'FunctionsTab';

// PermissionsTab component optimization
interface PermissionsTabProps {
  server: MCPServerConfig;
  onUpdate: (server: MCPServerConfig) => void;
  toolTemplate?: Tool;
}

const PermissionsTab = React.memo<PermissionsTabProps>(({ server, onUpdate, toolTemplate }) => {
  const [newResource, setNewResource] = useState('');
  const { capabilities, loading: isLoadingCapabilities } = useServerCapabilities(server?.id);

  // Track if we've initialized from capabilities to prevent infinite loops
  const initializedFromCapabilitiesRef = useRef(false);

  // Handle server resource data
  const { serverResources, allServerResources } = useMemo(() => {
    const resources =
      capabilities?.resources?.map((res) => {
        if (typeof res === 'string') return res;
        return (res as any).uri || (res as any).name || '';
      }) || [];

    // Build complete state for all resources (including enabled: true and false)
    // Use server.allResources if available (from previous updates), otherwise compute from capabilities
    const existingAllResources = server.allResources || [];
    const allResources =
      capabilities?.resources?.map((res) => {
        if (typeof res === 'string') {
          // Check if we already have this resource in server.allResources
          const existing = existingAllResources.find((r) => r.uri === res);
          return {
            uri: res,
            enabled:
              existing?.enabled ??
              (server.dataPermissions?.allowedResources?.includes(res) || false),
          };
        }
        const uri = (res as any).uri || (res as any).name || '';
        const existing = existingAllResources.find((r) => r.uri === uri);
        return {
          uri: uri,
          enabled:
            existing?.enabled ??
            ((res as any).enabled === true ||
              server.dataPermissions?.allowedResources?.includes(uri) ||
              false),
        };
      }) || [];

    return { serverResources: resources, allServerResources: allResources };
  }, [capabilities?.resources, server.dataPermissions?.allowedResources, server.allResources]);

  const [resources, setResources] = useState<string[]>(
    server.dataPermissions?.allowedResources || [],
  );
  const [availableResources, setAvailableResources] = useState<string[]>(serverResources);

  // Update resource state - only initialize on first capabilities load
  useEffect(() => {
    if (capabilities?.resources && !initializedFromCapabilitiesRef.current) {
      setAvailableResources(serverResources);

      // Initialize allResources with complete state for all resources
      if (allServerResources.length > 0) {
        const enabledResources = allServerResources.filter((r) => r.enabled).map((r) => r.uri);
        setResources(enabledResources);

        // Update ref to prevent sync effect from triggering
        lastSyncedResourcesRef.current = enabledResources;

        // Only update if server doesn't already have allResources set
        if (!server.allResources || server.allResources.length === 0) {
          const updatedServer = {
            ...server,
            dataPermissions: {
              ...server.dataPermissions,
              type: server.dataPermissions?.type || 'default',
              allowedResources: enabledResources,
            },
            allResources: allServerResources,
          };
          onUpdate(updatedServer);
        }
        initializedFromCapabilitiesRef.current = true;
      }
    }
  }, [
    capabilities?.resources?.length,
    server.id,
    onUpdate,
    serverResources,
    allServerResources,
    server.allResources,
  ]);

  // Reset initialization flag when server.id changes (different tool)
  useEffect(() => {
    initializedFromCapabilitiesRef.current = false;
    lastSyncedResourcesRef.current = [];
  }, [server.id]);

  // Sync local resources state when server prop changes externally (but not from our own updates)
  // Use a ref to track the last synced value to prevent unnecessary updates
  const lastSyncedResourcesRef = useRef<string[]>([]);
  useEffect(() => {
    if (server.dataPermissions?.allowedResources) {
      const serverResources = server.dataPermissions.allowedResources;
      const serverResourcesStr = JSON.stringify(serverResources.sort());
      const lastSyncedStr = JSON.stringify(lastSyncedResourcesRef.current.sort());

      // Only update if different and not from our own update
      if (serverResourcesStr !== lastSyncedStr) {
        setResources(serverResources);
        lastSyncedResourcesRef.current = serverResources;
      }
    }
  }, [server.dataPermissions?.allowedResources]);

  const handleAddResource = useCallback(() => {
    if (newResource.trim() && !resources.includes(newResource.trim())) {
      const updatedResources = [...resources, newResource.trim()];
      setResources(updatedResources);
      setNewResource('');

      // Update ref to prevent sync effect from triggering
      lastSyncedResourcesRef.current = updatedResources;

      // Update allResources, add new resource and set to enabled: true
      const currentAllResources = server.allResources || [];
      const updatedAllResources = [
        ...currentAllResources.filter((r) => r.uri !== newResource.trim()),
        { uri: newResource.trim(), enabled: true },
      ];

      const updatedServer = {
        ...server,
        dataPermissions: {
          ...server.dataPermissions,
          type: server.dataPermissions?.type || 'default',
          allowedResources: updatedResources,
        },
        allResources: updatedAllResources,
      };
      onUpdate(updatedServer);
    }
  }, [newResource, resources, server, onUpdate]);

  const handleRemoveResource = useCallback(
    (resource: string) => {
      const updatedResources = resources.filter((r) => r !== resource);
      setResources(updatedResources);

      // Update ref to prevent sync effect from triggering
      lastSyncedResourcesRef.current = updatedResources;

      // Update allResources, remove this resource
      const currentAllResources = server.allResources || [];
      const updatedAllResources = currentAllResources.filter((r) => r.uri !== resource);

      const updatedServer = {
        ...server,
        dataPermissions: {
          ...server.dataPermissions,
          type: server.dataPermissions?.type || 'default',
          allowedResources: updatedResources,
        },
        allResources: updatedAllResources,
      };
      onUpdate(updatedServer);
    },
    [resources, server, onUpdate],
  );

  const handleResourceToggle = useCallback(
    (resource: string, checked: boolean) => {
      // Update enabled state for this resource in allResources
      const currentAllResources = server.allResources || [];
      const resourceExists = currentAllResources.some((r) => r.uri === resource);

      let updatedAllResources: Array<{ uri: string; enabled: boolean }>;
      if (resourceExists) {
        // Update existing resource enabled state
        updatedAllResources = currentAllResources.map((r) =>
          r.uri === resource ? { ...r, enabled: checked } : r,
        );
      } else {
        // Add new resource
        updatedAllResources = [...currentAllResources, { uri: resource, enabled: checked }];
      }

      let updatedResources: string[];
      if (checked) {
        if (!resources.includes(resource)) {
          updatedResources = [...resources, resource];
        } else {
          // Already in resources, just update allResources
          updatedResources = resources;
        }
      } else {
        updatedResources = resources.filter((r) => r !== resource);
      }

      // Update local state
      setResources(updatedResources);

      // Update ref to prevent sync effect from triggering
      lastSyncedResourcesRef.current = updatedResources;

      const updatedServer = {
        ...server,
        dataPermissions: {
          ...server.dataPermissions,
          type: server.dataPermissions?.type || 'default',
          allowedResources: updatedResources,
        },
        allResources: updatedAllResources,
      };
      onUpdate(updatedServer);
    },
    [resources, server, onUpdate],
  );

  const getPermissionDescription = () => {
    return toolTemplate
      ? `Configure resource access permissions for ${toolTemplate.name}. Define which resources this tool can access.`
      : 'Configure data access permissions for this tool';
  };

  const getResourcePlaceholder = () => {
    const sampleResource = toolTemplate?.toolResources?.[0]?.uri;
    return sampleResource ? `e.g., ${sampleResource}` : 'Resource identifier';
  };

  if (isLoadingCapabilities) {
    return (
      <Card>
        <CardContent className="pt-6 text-center">
          <div className="flex items-center justify-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            <span className="ml-3 text-muted-foreground">Loading server resources...</span>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Server resources info */}
      {capabilities?.resources?.length && (
        <div className="bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded-lg p-3">
          <div className="flex items-start gap-2">
            <Shield className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5 flex-shrink-0" />
            <div className="text-sm text-green-800 dark:text-green-300">
              <p className="font-medium mb-1">Server Resources Loaded from Tool</p>
              <p className="text-xs">
                Found {capabilities.resources.length} resource permissions configured
              </p>
            </div>
          </div>
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Resource Permissions</CardTitle>
          <CardDescription>{getPermissionDescription()}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Available Resources from Server */}
          {availableResources.length > 0 && (
            <div>
              <h4 className="text-sm font-medium mb-2">Available Resources from Server</h4>
              <div className="space-y-2">
                {availableResources.map((resource) => {
                  const isEnabled = resources.includes(resource);
                  return (
                    <div
                      key={resource}
                      className="flex items-center justify-between p-2 border rounded"
                    >
                      <span className="text-sm font-mono">{resource}</span>
                      <div className="flex items-center gap-2">
                        <Switch
                          aria-label={`${isEnabled ? 'Disable' : 'Enable'} resource ${resource}`}
                          checked={isEnabled}
                          onCheckedChange={(checked) => handleResourceToggle(resource, checked)}
                          size="sm"
                        />
                        <span className="text-xs text-muted-foreground">
                          {isEnabled ? 'Enabled' : 'Disabled'}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* Add Custom Resource */}
          <div>
            <h4 className="text-sm font-medium mb-2">Add Custom Resource</h4>
            <Label htmlFor="new-resource-input" className="sr-only">
              Custom resource
            </Label>
            <div className="flex gap-2">
              <Input
                id="new-resource-input"
                placeholder={getResourcePlaceholder()}
                value={newResource}
                onChange={(e) => setNewResource(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleAddResource()}
              />
              <Button onClick={handleAddResource}>Add Resource</Button>
            </div>
          </div>

          <div className="space-y-2">
            <Label>Allowed Resources ({resources.length})</Label>
            {resources.length === 0 ? (
              <div className="text-sm text-muted-foreground py-4 text-center border-2 border-dashed rounded-lg dark:border-gray-700">
                No resources configured. Add resources above to grant access permissions.
              </div>
            ) : (
              <div className="space-y-2">
                {resources.map((resource, index) => (
                  <div
                    key={index}
                    className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-800 rounded-lg"
                  >
                    <span className="text-sm font-mono">{resource}</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleRemoveResource(resource)}
                      className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                    >
                      Remove
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
});
PermissionsTab.displayName = 'PermissionsTab';

// RestApiConfigTab component for category === 3 tools
interface RestApiConfigTabProps {
  config: string;
  onConfigChange: (value: string) => void;
  error: string | null;
  onDelete?: () => void;
  serverName?: string;
  lazyStartEnabled: boolean;
  onLazyStartEnabledChange: (value: boolean) => void;
  publicAccess: boolean;
  onPublicAccessChange: (value: boolean) => void;
  anonymousAccess: boolean;
  onAnonymousAccessChange: (value: boolean) => void;
  anonymousRateLimit: number;
  onAnonymousRateLimitChange: (value: number) => void;
  allowUserInput: boolean;
}

const RestApiConfigTab = React.memo<RestApiConfigTabProps>(
  ({
    config,
    onConfigChange,
    error,
    onDelete,
    serverName,
    lazyStartEnabled,
    onLazyStartEnabledChange,
    publicAccess,
    onPublicAccessChange,
    anonymousAccess,
    onAnonymousAccessChange,
    anonymousRateLimit,
    onAnonymousRateLimitChange,
    allowUserInput,
  }) => {
    return (
      <div className="space-y-4">
        {/* Tool Name Display */}
        <div className="space-y-3">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
            {serverName || 'REST API Tool Configuration'}
          </h3>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Edit the REST API configuration below. Changes will be validated before saving.
          </p>
        </div>

        {/* Configuration Input */}
        <div className="space-y-2">
          <Label htmlFor="rest-api-config-edit">
            REST API Configuration <span className="text-red-500 dark:text-red-400">*</span>
          </Label>
          <Textarea
            id="rest-api-config-edit"
            className="min-h-[350px] font-mono text-sm resize-y"
            placeholder={`Paste your REST API configuration in JSON or YAML format...

Example JSON:
{
  "name": "weather-api",
  "description": "OpenWeather API Service",
  "baseUrl": "https://api.openweathermap.org/data/2.5",
  "auth": {
    "type": "query_param",
    "param": "appid",
    "value": "YOUR_API_KEY"
  },
  "tools": [
    {
      "name": "getCurrentWeather",
      "description": "Get current weather for a city",
      "endpoint": "/weather",
      "method": "GET",
      "parameters": [
        {
          "name": "city",
          "description": "City name",
          "type": "string",
          "required": true,
          "location": "query",
          "mapping": "q"
        }
      ]
    }
  ]
}`}
            value={config}
            onChange={(e) => onConfigChange(e.target.value)}
          />
          <div className="flex flex-col gap-2">
            <p className="text-xs text-muted-foreground">
              Supports: JSON format, YAML format, or OpenAPI 3.x specification
            </p>
            {error && (
              <div
                role="alert"
                className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3"
              >
                <div className="flex items-start gap-2">
                  <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
                  <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Information Box */}
        <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
          <div className="flex items-start gap-2">
            <FileText className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
            <div className="text-sm text-blue-800 dark:text-blue-200">
              <p className="font-medium mb-1">Important Notes:</p>
              <ul className="list-disc list-inside space-y-0.5 text-xs">
                <li>Required fields: name, description, baseUrl, tools</li>
                <li>
                  Replace environment variables like ${'{'}API_KEY{'}'} with actual values
                </li>
                <li>OpenAPI specifications will be automatically converted</li>
                <li>Configuration will be validated before submission</li>
              </ul>
            </div>
          </div>
        </div>

        <PublicAccessConfiguration checked={publicAccess} onCheckedChange={onPublicAccessChange} />

        {!allowUserInput && (
          <AnonymousAccessConfiguration
            checked={anonymousAccess}
            onCheckedChange={onAnonymousAccessChange}
            rateLimit={anonymousRateLimit}
            onRateLimitChange={onAnonymousRateLimitChange}
          />
        )}

        <LazyStartConfiguration
          checked={lazyStartEnabled}
          onCheckedChange={onLazyStartEnabledChange}
        />

        {/* Delete Tool Section */}
        {onDelete && (
          <div className="border-t dark:border-gray-700 pt-4 mt-6">
            <Button
              variant="outline"
              size="sm"
              onClick={onDelete}
              className="text-red-600 hover:text-red-700 hover:bg-red-50 border-red-200 dark:text-red-400 dark:hover:text-red-300 dark:hover:bg-red-900/20 dark:border-red-800"
            >
              Delete Tool
            </Button>
          </div>
        )}
      </div>
    );
  },
);
RestApiConfigTab.displayName = 'RestApiConfigTab';

// Main component optimization
export default function ToolConfigPage() {
  const params = useParams();
  const router = useRouter();
  const searchParams = useSearchParams();
  const mode = searchParams.get('mode') || 'setup';
  const serverType = searchParams.get('type');
  const templateId = searchParams.get('templateId');
  const stepParam = searchParams.get('step');
  const isEdit = searchParams.get('isEdit') === 'true';
  const toolId = params.id as string;

  const { proxyId, getServerInfo } = useServerInfo();

  // State management
  const [currentServer, setCurrentServer] = useState<MCPServerConfig | null>(null);
  // Ref to solve stale closure issue in OAuth callback
  const performCreateToolRef = useRef<
    | ((
        masterPwd: string,
        code?: string,
        allowUserInputOverride?: number,
        redirectUri?: string,
        pkceVerifier?: string,
      ) => Promise<void>)
    | null
  >(null);

  const [toolTemplate, setToolTemplate] = useState<Tool | null>(null);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [dynamicCredentials, setDynamicCredentials] = useState<Record<string, string>>({});
  const [toolCreated, setToolCreated] = useState(false);
  const [createdToolId, setCreatedToolId] = useState<string | null>(null);
  const [allowUserInput, setAllowUserInput] = useState(false);
  const [lazyStartEnabled, setLazyStartEnabled] = useState(true); // Default to true
  const [publicAccess, setPublicAccess] = useState(false); // Default to false (private)
  const [anonymousAccess, setAnonymousAccess] = useState(false);
  const [anonymousRateLimit, setAnonymousRateLimit] = useState(10);

  // Configuration Mode: owner (allowUserInput=0) or user (allowUserInput=1)
  const [configMode, setConfigMode] = useState<'owner' | 'user'>('owner');
  // App Credentials: kimbap's app or custom
  const [credentialSource, setCredentialSource] = useState<'kimbap' | 'custom'>('kimbap');

  useEffect(() => {
    if (requiresCustomOAuthApp(toolTemplate) && credentialSource === 'kimbap') {
      setCredentialSource('custom');
    }
  }, [toolTemplate, credentialSource]);

  // Use global master password dialog
  const { requestMasterPassword } = useMasterPassword();

  // REST API tool states (for category === 3)
  const [restApiConfig, setRestApiConfig] = useState('');
  const [restApiConfigError, setRestApiConfigError] = useState<string | null>(null);
  const [toolCategory, setToolCategory] = useState<number | undefined>(undefined);

  const [customRemoteConfig, setCustomRemoteConfig] = useState<{
    url?: string;
    headers?: Record<string, string>;
    command?: string;
    args?: string[];
    env?: Record<string, string>;
    cwd?: string;
  } | null>(null);
  const [customRemoteConfigError, setCustomRemoteConfigError] = useState<string | null>(null);
  const [customRemoteTransportType, setCustomRemoteTransportType] = useState<'url' | 'stdio'>(
    'url',
  );
  const [encryptedCustomRemoteConfig, setEncryptedCustomRemoteConfig] = useState<string | null>(
    null,
  );
  const [cacheHealth, setCacheHealth] = useState<CacheHealthState | null>(null);
  const [cachePolicies, setCachePolicies] = useState<CachePolicyState>({
    tools: {},
    prompts: {},
    resources: { exact: {}, patterns: [] },
  });
  const [isLoadingCache, setIsLoadingCache] = useState(false);

  // Access token dialog states
  const [showAccessTokenDialog, setShowAccessTokenDialog] = useState(false);
  const [accessTokenError, setAccessTokenError] = useState<string | null>(null);
  const [isDecrypting, setIsDecrypting] = useState(false);
  const [cachedAccessToken, setCachedAccessToken] = useState<string | null>(null); // Memory-only cache, lost on page refresh

  // Master password state
  const [showMasterPasswordDialog, setShowMasterPasswordDialog] = useState(false);
  const [masterPasswordAction, setMasterPasswordAction] = useState<
    'create' | 'update' | 'delete' | 'saveRestApi' | 'saveCustomRemote' | null
  >(null);
  // Pending REST API config data (used when waiting for master password)
  const [pendingRestApiData, setPendingRestApiData] = useState<{
    configString: string;
    proxyId: number;
  } | null>(null);
  const [isProcessingWithPassword, setIsProcessingWithPassword] = useState(false);
  const [masterPasswordForSession, setMasterPasswordForSession] = useState<string | null>(null);

  // Get current user role
  const getCurrentUserRole = (): string => {
    if (typeof window !== 'undefined') {
      const storedServer = localStorage.getItem('selectedServer');
      if (storedServer) {
        try {
          const parsedServer = JSON.parse(storedServer);
          return parsedServer.role || 'Member';
        } catch (error) {
          // Failed to parse selectedServer:
        }
      }
    }
    return 'Member'; // Default to Member if not found
  };

  const currentUserRole = getCurrentUserRole();
  const isOwner = currentUserRole === 'Owner';

  // Step configuration
  const allSteps: StepConfig[] = useMemo(
    () => [
      {
        id: 'credentials',
        title: 'Credentials',
        description: 'Configure authentication',
        icon: Lock,
        bgColor: 'bg-orange-500',
      },
      {
        id: 'functions',
        title: 'Functions',
        description: 'Enable tool features',
        icon: Zap,
        bgColor: 'bg-yellow-500',
      },
      {
        id: 'permissions',
        title: 'Resource Permissions',
        description: 'Set resource access',
        icon: Shield,
        bgColor: 'bg-red-500',
      },
    ],
    [],
  );

  // Filter steps by mode
  const steps = useMemo(() => {
    switch (mode) {
      case 'credentials':
        return [allSteps[0]];
      case 'functions':
        return [allSteps[1]];
      case 'permissions':
        return [allSteps[2]];
      case 'setup':
      default:
        // If allowUserInput is true, only show credentials step
        if (allowUserInput) {
          return [allSteps[0]];
        }
        return allSteps;
    }
  }, [mode, allSteps, allowUserInput]);

  // Current step calculation
  const getInitialStep = () => {
    if (stepParam) {
      const step = parseInt(stepParam, 10);
      if (!isNaN(step)) return step;
    }

    switch (mode) {
      case 'credentials':
        return 0;
      case 'functions':
        return 1;
      case 'permissions':
        return 2;
      case 'setup':
      default:
        return 0;
    }
  };

  const [currentStep, setCurrentStep] = useState(getInitialStep());
  const safeCurrentStep = Math.min(Math.max(currentStep, 0), steps.length - 1);
  const progress = ((safeCurrentStep + 1) / steps.length) * 100;

  // URL update
  const updateStepInUrl = useCallback(
    (newStep: number) => {
      const current = new URLSearchParams(searchParams.toString());
      current.set('step', newStep.toString());
      router.push(`/dashboard/tool-configure/${toolId}?${current.toString()}`);
      setCurrentStep(newStep);
    },
    [searchParams, router, toolId],
  );

  // Server update handler
  const handleServerUpdate = useCallback((updatedServer: MCPServerConfig) => {
    setCurrentServer(updatedServer);
  }, []);

  // Credential update handler
  const handleCredentialUpdate = useCallback((key: string, value: string) => {
    setDynamicCredentials((prev) => ({ ...prev, [key]: value }));
  }, []);

  const handleToolCachePolicyChange = useCallback((toolName: string, policy: CachePolicyConfig) => {
    setCachePolicies((prev) => ({
      ...prev,
      tools: {
        ...prev.tools,
        [toolName]: policy,
      },
    }));
  }, []);

  const handlePromptCachePolicyChange = useCallback(
    (promptName: string, policy: CachePolicyConfig) => {
      setCachePolicies((prev) => ({
        ...prev,
        prompts: {
          ...prev.prompts,
          [promptName]: policy,
        },
      }));
    },
    [],
  );

  const handleResourceCachePolicyChange = useCallback((uri: string, policy: CachePolicyConfig) => {
    setCachePolicies((prev) => ({
      ...prev,
      resources: {
        ...prev.resources,
        exact: {
          ...prev.resources.exact,
          [uri]: policy,
        },
      },
    }));
  }, []);

  const loadCacheData = useCallback(async () => {
    if (!createdToolId) return;

    try {
      setIsLoadingCache(true);
      const [healthResponse, policyResponse] = await Promise.all([
        api.tools.getCacheHealth(),
        api.tools.getCachePolicy({ serverId: createdToolId }),
      ]);

      if (healthResponse.data?.data) {
        setCacheHealth(healthResponse.data.data as CacheHealthState);
      }

      if (policyResponse.data?.data) {
        const policyData = policyResponse.data.data;
        setCachePolicies({
          tools: (policyData as any).tools || {},
          prompts: (policyData as any).prompts || {},
          resources: {
            exact: (policyData as any).resources?.exact || {},
            patterns: (policyData as any).resources?.patterns || [],
          },
        });
      }
    } catch {
      setCacheHealth(null);
    } finally {
      setIsLoadingCache(false);
    }
  }, [createdToolId]);

  // Tool metadata
  const toolMetadata = useMemo(() => {
    // Use default styles, not template-dependent
    return { icon: 'Bot', bgColor: 'bg-gray-500' };
  }, []);

  // Mode title
  const getModeTitle = () => {
    switch (mode) {
      case 'credentials':
        return 'Credentials';
      case 'functions':
        return 'Functions';
      case 'permissions':
        return 'Resource Permissions';
      case 'setup':
      default:
        return 'Setup';
    }
  };

  // Server configuration creation
  const createServerFromTemplate = useCallback(
    (template: Tool | null, serverType?: string): MCPServerConfig => {
      if (!template && !serverType) {
        return {
          id: String(params.id),
          type: 'unknown' as any,
          name: 'Unknown Tool',
          credentials: {},
          enabledFunctions: [],
          dataPermissions: { type: 'default', allowedResources: [] },
          status: 'connect_failed',
          authType: 1,
          allowUserInput: 0,
          enabled: false,
        };
      }

      if (!template && serverType) {
        return {
          id: String(params.id),
          type: serverType as any,
          name: `${serverType.charAt(0).toUpperCase() + serverType.slice(1)} Tool`,
          credentials: {},
          enabledFunctions: [],
          dataPermissions: { type: 'default', allowedResources: [] },
          status: 'connect_failed',
          authType: 1,
          allowUserInput: 0,
          enabled: false,
        };
      }

      return {
        id: String(params.id),
        type: String(template!.toolType) as any,
        name: template!.name,
        credentials: {},
        enabledFunctions: template!.toolFuncs?.map((f) => f.funcName) || [],
        dataPermissions: {
          type: 'default',
          allowedResources: template!.toolResources?.map((r) => r.uri) || [],
        },
        status: 'connect_failed',
        authType: template!.authType ?? 1,
        allowUserInput: 0,
        enabled: template!.enabled || false,
      };
    },
    [params.id],
  );

  // Template cache management
  const getTemplateFromCache = (templateId: string): Tool | null => {
    try {
      const cached = localStorage.getItem('kimbap-tool-templates-cache');
      const expiry = localStorage.getItem('kimbap-tool-templates-expiry');

      if (cached && expiry && Date.now() < parseInt(expiry) + 20 * 60 * 1000) {
        const templates: Tool[] = JSON.parse(cached);
        return templates.find((t) => t.toolTmplId === templateId) || null;
      }
    } catch (error) {
      // Failed to read template from cache:
    }
    return null;
  };

  // Initialize new tool configuration (based on serverType and cached template)
  useEffect(() => {
    // Only when serverType param exists is it a new tool creation flow
    if (!serverType) {
      return;
    }

    if (currentServer) {
      return;
    }

    // Get templateId from URL params and retrieve from localStorage cache
    const templateId = searchParams.get('templateId');

    let template: Tool | null = null;

    if (templateId) {
      // Get template from localStorage cache
      template = getTemplateFromCache(templateId);
    }

    if (template) {
      // Use template data
      setToolTemplate(template);
      const newServer = createServerFromTemplate(template, serverType);
      setCurrentServer(newServer);

      // Initialize dynamic credentials
      const initialCreds: Record<string, string> = {};
      template.credentials?.forEach((cred, index) => {
        const fieldKey = buildCredentialFieldKey(cred, index);
        initialCreds[fieldKey] = cred.value || '';
      });
      setDynamicCredentials(initialCreds);
    } else {
      // No template cache, use default configuration
      const newServer = createServerFromTemplate(null, serverType);
      setCurrentServer(newServer);
    }
  }, [serverType, createServerFromTemplate, searchParams]);

  // Load existing tool
  useEffect(() => {
    const loadExistingTool = async () => {
      if (toolId !== 'new' && !serverType && !toolCreated) {
        try {
          setLoading(true);

          const proxyId = await getServerInfo();
          if (!proxyId) return;

          const response = await api.tools.getToolList({
            proxyId,
            handleType: 1,
          });

          const tool = response.data?.data?.toolList?.find((t: any) => t.toolId === toolId);
          if (tool) {
            setToolCreated(true);
            setCreatedToolId(toolId);
            // Set allowUserInput flag to determine if only credentials step should be shown
            // allowUserInput: 1 = user configures, 0 = pre-configured
            setAllowUserInput(tool.allowUserInput === 1);

            // Set tool category
            setToolCategory(tool.category);

            // If REST API tool (category === 3), set the restApiConfig
            if (tool.category === 3 && tool.restApiConfig) {
              setRestApiConfig(JSON.stringify(tool.restApiConfig, null, 2));
            }

            if (isCustomMcpCategory(tool.category)) {
              const rawCustomMcpConfig =
                tool.category === 5 ? tool.stdioConfig : tool.customRemoteConfig;

              if (rawCustomMcpConfig) {
                const allowUserInputFlag = tool.allowUserInput === 1;

                if (allowUserInputFlag) {
                  try {
                    const config = JSON.parse(rawCustomMcpConfig);
                    setCustomRemoteConfig(config);
                    setCustomRemoteTransportType(tool.category === 5 ? 'stdio' : 'url');
                  } catch (error) {
                    setCustomRemoteConfigError('Could not parse configuration');
                  }
                } else {
                  setEncryptedCustomRemoteConfig(rawCustomMcpConfig);

                  if (cachedAccessToken) {
                    setTimeout(() => {
                      handleDecryptConfigWithAccessToken(
                        rawCustomMcpConfig,
                        cachedAccessToken,
                        (config) => {
                          setCustomRemoteConfig(config);
                          setCustomRemoteTransportType(tool.category === 5 ? 'stdio' : 'url');
                          setEncryptedCustomRemoteConfig(null);
                        },
                      );
                    }, 0);
                  } else {
                    setShowAccessTokenDialog(true);
                  }
                }
              }
            }

            const serverConfig: MCPServerConfig = {
              id: tool.toolId,
              type: getServerTypeFromToolType(tool.toolType),
              name: tool.name,
              toolTmplId: tool.toolTmplId,
              credentials: tool.credentials || {},
              authType: tool.authType,
              allowUserInput: tool.allowUserInput,
              enabledFunctions:
                tool.toolFuncs?.filter((f: any) => f.enabled).map((f: any) => f.funcName) || [],
              // Add complete function configuration info, including dangerLevel
              allFunctions:
                tool.toolFuncs?.map((f: any) => ({
                  funcName: f.funcName,
                  enabled: f.enabled || false,
                  dangerLevel: f.dangerLevel || 0,
                  description: f.description || '',
                })) || [],
              dataPermissions: {
                type: 'default',
                allowedResources:
                  tool.toolResources?.filter((r: any) => r.enabled).map((r: any) => r.uri) || [],
              },
              // Add complete resource configuration info, including enabled state for all resources
              allResources:
                tool.toolResources?.map((r: any) => ({
                  uri: r.uri,
                  enabled: r.enabled || false,
                })) || [],
              status: getStatusFromRunningState(tool.runningState),
              lastValidated: tool.lastUsed
                ? new Date(tool.lastUsed * 1000).toISOString()
                : undefined,
              enabled: tool.enabled,
              lazyStartEnabled: tool.lazyStartEnabled ?? true,
            };

            setCurrentServer(serverConfig);
            setLazyStartEnabled(tool.lazyStartEnabled ?? true);
            setPublicAccess(tool.publicAccess ?? false);
            setAnonymousAccess(tool.anonymousAccess ?? false);
            setAnonymousRateLimit(tool.anonymousRateLimit ?? 10);

            // Load tool template for edit mode to display credential fields
            if (tool.toolTmplId) {
              const template = getTemplateFromCache(tool.toolTmplId);
              if (template) {
                setToolTemplate(template);

                // Initialize dynamic credentials from tool's authConf
                // Map by key property to ensure correct matching with template credentials
                if (tool.authConf && Array.isArray(tool.authConf) && template.credentials) {
                  const credentials: Record<string, string> = {};
                  template.credentials.forEach((credential: any, index: number) => {
                    const fieldKey = buildCredentialFieldKey(credential, index);
                    const authConfig = tool.authConf.find(
                      (auth: any) => auth.key === credential.key,
                    );
                    credentials[fieldKey] = authConfig?.value || '';
                  });
                  setDynamicCredentials(credentials);
                }
              }
            }
          }
        } catch (error) {
          // Failed to load existing tool:
        } finally {
          setLoading(false);
        }
      }
    };

    loadExistingTool();
  }, [toolId, serverType, toolCreated, getServerInfo]);

  useEffect(() => {
    if (!createdToolId) return;
    loadCacheData();
  }, [createdToolId, loadCacheData]);

  // Tool creation
  const performCreateTool = useCallback(
    async (
      masterPwd: string,
      code?: string,
      allowUserInputOverride?: number,
      redirectUri?: string,
      pkceVerifier?: string,
    ) => {
      if (!currentServer) return;

      try {
        setCreating(true);
        setMasterPasswordForSession(masterPwd);

        const allowUserInputValue = allowUserInputOverride ?? (configMode === 'user' ? 1 : 0);

        let authConf: any[] = [];

        if (toolTemplate?.oAuthConfig) {
          // OAuth tool: build authConf based on credentialSource and allowUserInputValue
          const requiresCustomApp = requiresCustomOAuthApp(toolTemplate);

          if (credentialSource === 'kimbap' && requiresCustomApp) {
            toast.error('This tool requires your own OAuth app credentials');
            setCreating(false);
            return;
          }

          // 1. Determine clientId and clientSecret
          let clientId: string;
          let clientSecret: string | undefined;

          if (credentialSource === 'kimbap') {
            clientId =
              allowUserInputValue === 1
                ? toolTemplate.oAuthConfig.userClientId
                : toolTemplate.oAuthConfig.clientId;
            // In kimbap mode, clientSecret is handled server-side, no need to pass
          } else {
            clientId =
              getCredentialValueByTemplateKey(
                toolTemplate.credentials,
                dynamicCredentials,
                'YOUR_CLIENT_ID',
              ) || '';
            clientSecret = getCredentialValueByTemplateKey(
              toolTemplate.credentials,
              dynamicCredentials,
              'YOUR_CLIENT_SECRET',
            );

            // Validate required fields in custom mode
            if (
              !clientId ||
              !clientSecret ||
              clientId === '' ||
              clientSecret === '' ||
              clientId === 'YOUR_CLIENT_ID' ||
              clientSecret === 'YOUR_CLIENT_SECRET'
            ) {
              // Custom credentials require both Client ID and Client Secret, and they cannot be empty or YOUR_CLIENT_ID or YOUR_CLIENT_SECRET
              setCreating(false);
              return;
            }
          }

          // 2. Add clientId (required)
          if (clientId) {
            authConf.push({
              key: 'YOUR_CLIENT_ID',
              value: clientId,
              dataType: 1,
              name: 'Client ID',
              description: 'OAuth Client ID',
            });
          }

          // 3. Add clientSecret (custom mode only)
          if (credentialSource === 'custom' && clientSecret) {
            authConf.push({
              key: 'YOUR_CLIENT_SECRET',
              value: clientSecret,
              dataType: 1,
              name: 'Client Secret',
              description: 'OAuth Client Secret',
            });
          }

          // 4. Parse and write OAuth endpoint overrides (custom mode only)
          if (credentialSource === 'custom') {
            const endpointOverrides = getOAuthEndpointOverridesFromCredentials(
              toolTemplate,
              dynamicCredentials,
            );

            try {
              const resolvedEndpoints = resolveOAuthEndpoints(
                toolTemplate.oAuthConfig,
                endpointOverrides,
              );

              if (resolvedEndpoints.usingOverride) {
                authConf.push({
                  key: OAUTH_AUTHORIZATION_URL_KEY,
                  value: resolvedEndpoints.authorizationUrl,
                  dataType: 1,
                  name: 'OAuth Authorization URL',
                  description: 'Resolved OAuth authorization endpoint',
                });
                authConf.push({
                  key: OAUTH_TOKEN_URL_KEY,
                  value: resolvedEndpoints.tokenUrl,
                  dataType: 1,
                  name: 'OAuth Token URL',
                  description: 'Resolved OAuth token endpoint',
                });
              }

              if (endpointOverrides.baseUrl) {
                authConf.push({
                  key: OAUTH_BASE_URL_KEY,
                  value: endpointOverrides.baseUrl,
                  dataType: 1,
                  name: 'OAuth Base URL',
                  description: 'OAuth service base URL',
                });
              }
            } catch (error) {
              toast.error(
                error instanceof Error ? error.message : 'Invalid OAuth endpoint configuration',
              );
              setCreating(false);
              return;
            }
          }

          // 5. Add code and redirectUri (Owner mode only, allowUserInputValue === 0)
          if (allowUserInputValue === 0) {
            if (code && code !== '') {
              authConf.push({
                key: 'YOUR_OAUTH_CODE',
                value: code,
                dataType: 1,
                name: 'OAuth Code',
                description: 'OAuth authorization code',
              });
              if (redirectUri) {
                authConf.push({
                  key: 'YOUR_OAUTH_REDIRECT_URL',
                  value: redirectUri,
                  dataType: 1,
                  name: 'Redirect URL',
                  description: 'OAuth redirect URL',
                });

                if (pkceVerifier) {
                  authConf.push({
                    key: OAUTH_CODE_VERIFIER_KEY,
                    value: pkceVerifier,
                    dataType: 1,
                    name: 'PKCE Code Verifier',
                    description: 'OAuth PKCE code verifier',
                  });
                }
              } else {
                // OAuth Code is required
                setCreating(false);
                return;
              }
            }
          }
        } else if (toolTemplate?.credentials && toolTemplate.credentials.length > 0) {
          // Non-OAuth tool: keep original logic
          const credentials = toolTemplate.credentials;

          authConf = Object.entries(dynamicCredentials)
            .filter(([_, value]) => value.trim() !== '')
            .map(([key, value]) => {
              const credIndex = parseInt(key.split('_')[1]);
              const credential = credentials.length > credIndex ? credentials[credIndex] : null;
              return {
                key: credential?.key || key,
                value: value?.trim(),
                dataType: credential?.dataType || 1,
                name: credential?.name || key,
                description: credential?.description || '',
              };
            });
        }

        const proxyId = await getServerInfo();
        const initialResources =
          toolTemplate?.toolResources?.map((res) => ({
            uri: res.uri,
            enabled: res.enabled || true,
          })) || [];

        const response = await api.tools.operateTool({
          handleType: 1,
          proxyId: proxyId || '1',
          toolTmplId: templateId || undefined,
          toolType: toolTemplate ? toolTemplate.toolType : 1,
          authConf,
          masterPwd,
          lazyStartEnabled: lazyStartEnabled,
          publicAccess: publicAccess,
          anonymousAccess: anonymousAccess,
          anonymousRateLimit: anonymousRateLimit,
          allowUserInput: allowUserInputValue,
        });

        if (response.data?.data?.toolId) {
          setToolCreated(true);
          setCreatedToolId(response.data.data.toolId);

          setCurrentServer((prev) =>
            prev
              ? {
                  ...prev,
                  id: response.data.data.toolId,
                  enabledFunctions: [],
                  dataPermissions: {
                    ...prev.dataPermissions,
                    allowedResources: initialResources.map((r) => r.uri),
                  },
                  // Initialize allResources with complete state for all resources
                  allResources: initialResources.map((r) => ({
                    uri: r.uri,
                    enabled: r.enabled || false,
                  })),
                }
              : prev,
          );

          // If user mode (allowUserInput=1), go back to list page
          if (allowUserInputValue === 1) {
            router.push('/dashboard/tool-configure');
            return;
          }

          const newUrl = `/dashboard/tool-configure/${response.data.data.toolId}?mode=setup&step=1`;
          const finalUrl = toolTemplate
            ? `${newUrl}&templateId=${toolTemplate.toolTmplId}`
            : newUrl;
          router.push(finalUrl);
          setCurrentStep(1);
        }
      } catch (error) {
        // Failed to create tool:
      } finally {
        setCreating(false);
      }
    },
    [
      currentServer,
      dynamicCredentials,
      toolTemplate,
      templateId,
      lazyStartEnabled,
      publicAccess,
      anonymousAccess,
      anonymousRateLimit,
      configMode,
      getServerInfo,
      router,
      credentialSource,
    ],
  );

  // Keep ref always pointing to latest performCreateTool, solve stale closure issue in OAuth callback
  useEffect(() => {
    performCreateToolRef.current = performCreateTool;
  }, [performCreateTool]);

  // Handle OAuth callback - must be after performCreateTool definition
  useEffect(() => {
    const handleOAuthCallback = async () => {
      const oauthCallback = searchParams.get('oauthCallback');
      if (oauthCallback !== 'true') return;

      const callbackData = sessionStorage.getItem('oauth_callback');
      if (!callbackData) return;

      try {
        const { code, state } = JSON.parse(callbackData);
        sessionStorage.removeItem('oauth_callback');

        const { getOAuthSession, clearOAuthSession } = await import('@/lib/oauth-authorization');
        const sessionData = getOAuthSession(state);

        if (sessionData && code) {
          // Restore state
          setConfigMode(sessionData.allowUserInput === 1 ? 'user' : 'owner');
          setCredentialSource(sessionData.credentialSource);
          setLazyStartEnabled(sessionData.lazyStartEnabled ?? true);
          setPublicAccess(sessionData.publicAccess ?? false);
          setAnonymousAccess(sessionData.anonymousAccess ?? false);
          setAnonymousRateLimit(sessionData.anonymousRateLimit ?? 10);

          if (sessionData.credentials) {
            setDynamicCredentials(sessionData.credentials);
          }

          // Clear session data
          clearOAuthSession(state);

          // Request master password and create tool
          requestMasterPassword({
            title: 'Create Tool - Master Password Required',
            description: 'Please enter your master password to create the tool.',
            onConfirm: async (password) => {
              // Use ref to get latest performCreateTool, avoid stale closure issue
              if (performCreateToolRef.current) {
                await performCreateToolRef.current(
                  password,
                  code,
                  sessionData.allowUserInput,
                  sessionData.redirectUri,
                  sessionData.pkceVerifier,
                );
              }
            },
          });
        }
      } catch (error) {
        // Failed to handle OAuth callback:
      }
    };

    handleOAuthCallback();
  }, [searchParams, requestMasterPassword, performCreateTool]);

  // Tool update
  const performUpdateTool = useCallback(
    async (
      stepData: 'functions' | 'permissions' | 'complete',
      masterPwd: string,
      shouldNavigate = false,
    ) => {
      if (!currentServer || !createdToolId) return;

      try {
        setCreating(true);

        const proxyId = await getServerInfo();

        let functions: any[] | undefined;
        let resources: any[] | undefined;
        let authConf: any[] | undefined;

        // Functions/permissions step in setup mode does not surface access flags UI.
        // Avoid sending lazyStartEnabled/publicAccess there to prevent overwriting server defaults.
        const shouldIncludeAccessFlags = !(
          (mode === 'setup' && (safeCurrentStep === 1 || safeCurrentStep === 2)) ||
          mode === 'functions' ||
          mode === 'permissions'
        );

        if (stepData === 'functions' || stepData === 'complete') {
          // Only collect functions data in setup mode or when explicitly in functions mode
          // Don't collect when in credentials or permissions mode
          if (mode === 'setup' || mode === 'functions') {
            // Use allFunctions if exists, it contains correct enabled state
            if (currentServer.allFunctions) {
              functions = currentServer.allFunctions;
            } else {
              // If no allFunctions, only create entries for enabled functions
              functions = currentServer.enabledFunctions.map((funcName) => ({
                funcName,
                enabled: true,
              }));
            }
          }
        }

        if (stepData === 'permissions' || stepData === 'complete') {
          // Only collect resources data in setup mode or when explicitly in permissions mode
          // Don't collect when in credentials or functions mode
          if (mode === 'setup' || mode === 'permissions') {
            // Use allResources if exists, it contains complete state for all resources (enabled: true/false)
            if (currentServer.allResources && currentServer.allResources.length > 0) {
              resources = currentServer.allResources;
            } else {
              // If no allResources, only create entries for enabled resources
              resources =
                currentServer.dataPermissions?.allowedResources?.map((uri) => ({
                  uri,
                  enabled: true,
                })) || [];
            }
          }
        }

        const requestParams: any = {
          handleType: 2,
          proxyId: proxyId || '1',
          toolId: createdToolId,
          toolTmplId: templateId || undefined,
          toolType: toolTemplate ? toolTemplate.toolType : 1,
        };

        const hasCachePolicies =
          Object.keys(cachePolicies.tools || {}).length > 0 ||
          Object.keys(cachePolicies.prompts || {}).length > 0 ||
          Object.keys(cachePolicies.resources?.exact || {}).length > 0 ||
          (cachePolicies.resources?.patterns?.length ?? 0) > 0;

        if (hasCachePolicies) {
          requestParams.cachePolicies = cachePolicies;
        }

        if (shouldIncludeAccessFlags) {
          requestParams.lazyStartEnabled = lazyStartEnabled;
          requestParams.publicAccess = publicAccess;
          requestParams.anonymousAccess = anonymousAccess;
          requestParams.anonymousRateLimit = anonymousRateLimit;
        }

        // Only include modified parameters
        if (authConf && authConf.length > 0) {
          requestParams.authConf = authConf;
        }

        if (functions && functions.length > 0) {
          requestParams.functions = functions;
        }

        if (resources && resources.length > 0) {
          requestParams.resources = resources;
        }

        if (masterPwd?.trim()) {
          requestParams.masterPwd = masterPwd;
        }

        await api.tools.operateTool(requestParams);

        if (stepData === 'complete') {
          router.push('/dashboard/tool-configure');
          return;
        } else if (shouldNavigate || stepData !== 'functions') {
          // Navigate to next step if explicitly requested or for non-function updates
          setCurrentStep((prev) => prev + 1);
          updateStepInUrl(safeCurrentStep + 1);
        }
      } catch (error) {
        // Failed to update tool:
      } finally {
        setCreating(false);
      }
    },
    [
      currentServer,
      createdToolId,
      toolTemplate,
      templateId,
      dynamicCredentials,
      cachePolicies,
      mode,
      lazyStartEnabled,
      publicAccess,
      anonymousAccess,
      anonymousRateLimit,
      getServerInfo,
      router,
      updateStepInUrl,
      safeCurrentStep,
    ],
  );

  // Functions auto-save callback with proper dependency management
  const handleFunctionsAutoSave = useCallback(
    async (functions: any[]) => {
      if (!createdToolId) return;
      try {
        const proxyId = await getServerInfo();
        const requestParams: any = {
          handleType: 2,
          proxyId: proxyId || '1',
          toolId: createdToolId,
          toolTmplId: templateId || undefined,
          toolType: toolTemplate ? toolTemplate.toolType : 1,
        };

        const hasCachePolicies =
          Object.keys(cachePolicies.tools || {}).length > 0 ||
          Object.keys(cachePolicies.prompts || {}).length > 0 ||
          Object.keys(cachePolicies.resources?.exact || {}).length > 0 ||
          (cachePolicies.resources?.patterns?.length ?? 0) > 0;

        if (hasCachePolicies) {
          requestParams.cachePolicies = cachePolicies;
        }

        // Only include functions if there are any
        if (functions && functions.length > 0) {
          requestParams.functions = functions;
        }

        await api.tools.operateTool(requestParams);
      } catch (error) {
        // Auto-save failed:
      }
    },
    [
      createdToolId,
      toolTemplate,
      templateId,
      lazyStartEnabled,
      publicAccess,
      anonymousAccess,
      anonymousRateLimit,
      cachePolicies,
      getServerInfo,
    ],
  );

  // Delete tool
  const performDeleteTool = useCallback(
    async (masterPwd: string) => {
      if (!currentServer || !createdToolId || !proxyId) return;

      try {
        setCreating(true);

        // If it's a Skills tool (category === 4), delete the skills directory first
        if (toolCategory === 4) {
          try {
            await api.tools.deleteServerSkills({ serverId: createdToolId });
          } catch (error) {
            // Continue with tool deletion even if skills deletion fails
          }
        }

        const response = await api.tools.operateTool({
          handleType: 5, // 5 = delete
          proxyId: proxyId,
          toolId: createdToolId,
          masterPwd: masterPwd,
        });

        if (response.data?.common?.code === 0) {
          // Success - navigate back to tool list
          router.push('/dashboard/tool-configure');
        } else {
          throw new Error(response.data?.common?.message || 'Could not delete tool');
        }
      } catch (error: any) {
        // Delete tool error:
        throw error;
      } finally {
        setCreating(false);
      }
    },
    [currentServer, createdToolId, proxyId, router, toolCategory],
  );

  const handleDeleteTool = useCallback(() => {
    setMasterPasswordAction('delete');
    setShowMasterPasswordDialog(true);
  }, []);

  // Handle saving Skills configuration (category === 4)
  const handleSaveSkillsConfig = useCallback(async () => {
    try {
      setCreating(true);

      // Get proxyId
      const proxyIdValue = await getServerInfo();
      if (!proxyIdValue) {
        toast.error('Unable to get server information');
        setCreating(false);
        return;
      }

      // Call API to update Skills tool with handleType=2
      const response = await api.tools.operateTool({
        handleType: 2, // 2 = update
        proxyId: proxyIdValue,
        toolId: createdToolId!,
        lazyStartEnabled: lazyStartEnabled,
        publicAccess: publicAccess,
        anonymousAccess: anonymousAccess,
        anonymousRateLimit: anonymousRateLimit,
      });

      if (response.data?.common?.code === 0) {
        toast.success('Skills configuration saved');
      } else {
        toast.error(response.data?.common?.message || 'Could not save Skills configuration');
      }
    } catch (error: any) {
      // Failed to save Skills config:
      toast.error(error.message || 'Could not save Skills configuration');
    } finally {
      setCreating(false);
    }
  }, [
    getServerInfo,
    createdToolId,
    lazyStartEnabled,
    publicAccess,
    anonymousAccess,
    anonymousRateLimit,
  ]);

  // Perform the actual REST API config save (called after password is obtained)
  const performSaveRestApiConfig = useCallback(
    async (masterPwd: string, configString: string, proxyIdValue: number) => {
      try {
        setCreating(true);

        // Call API to update REST API tool with handleType=2
        const response = await api.tools.operateTool({
          handleType: 2, // 2 = update
          proxyId: proxyIdValue,
          toolId: createdToolId!,
          restApiConfig: configString,
          lazyStartEnabled: lazyStartEnabled,
          publicAccess: publicAccess,
          anonymousAccess: anonymousAccess,
          anonymousRateLimit: anonymousRateLimit,
          masterPwd: masterPwd,
        });

        if (response.data?.common?.code === 0) {
          // Success - navigate back to tool list
          router.push('/dashboard/tool-configure');
        } else {
          setRestApiConfigError(response.data?.common?.message || 'Could not update REST API tool');
        }
      } catch (error: any) {
        // Failed to save REST API config:
        const errorMessage =
          error.response?.data?.common?.message ||
          error.response?.data?.message ||
          error.message ||
          'Could not save REST API configuration';
        setRestApiConfigError(errorMessage);
      } finally {
        setCreating(false);
        setPendingRestApiData(null);
      }
    },
    [createdToolId, lazyStartEnabled, publicAccess, anonymousAccess, anonymousRateLimit, router],
  );

  // Generic decrypt config with access token (for customRemoteConfig, restApiConfig, etc.)
  const handleDecryptConfigWithAccessToken = useCallback(
    async (
      encryptedData: string,
      accessToken: string,
      onSuccess: (decryptedConfig: any) => void,
      options?: {
        validateStructure?: (config: any) => boolean;
        successMessage?: string;
      },
    ) => {
      if (!encryptedData) return;

      try {
        setIsDecrypting(true);
        setAccessTokenError(null);

        // Import CryptoUtils
        const { CryptoUtils } = await import('@/lib/crypto');

        // Decrypt the configuration
        const decryptedStr = await CryptoUtils.decryptDataFromString(encryptedData, accessToken);

        // Parse the decrypted JSON
        const config = JSON.parse(decryptedStr);

        // Validate structure if validator provided
        if (options?.validateStructure && !options.validateStructure(config)) {
          throw new Error('Invalid configuration structure');
        }

        // Success callback
        onSuccess(config);

        setShowAccessTokenDialog(false);

        // Cache the access token in memory (current page session only, lost on refresh)
        setCachedAccessToken(accessToken);

        // Show success message
        const { toast } = await import('sonner');
        toast.success(options?.successMessage || 'Configuration decrypted');
      } catch (error) {
        // Failed to decrypt config:
        setAccessTokenError('Could not decrypt. Check your access token.');
      } finally {
        setIsDecrypting(false);
      }
    },
    [],
  );

  // Wrapper for customRemoteConfig decryption (for AccessTokenDialog onConfirm)
  const handleDecryptCustomRemoteConfig = useCallback(
    async (accessToken: string) => {
      if (!encryptedCustomRemoteConfig) return;

      await handleDecryptConfigWithAccessToken(
        encryptedCustomRemoteConfig,
        accessToken,
        (config) => {
          setCustomRemoteConfig(config);
          setCustomRemoteTransportType(
            toolCategory === 5 ? 'stdio' : config.command ? 'stdio' : 'url',
          );
          setEncryptedCustomRemoteConfig(null);
        },
        {
          validateStructure: (config) => !!config.url || !!config.command,
          successMessage: 'Custom MCP configuration decrypted',
        },
      );
    },
    [encryptedCustomRemoteConfig, handleDecryptConfigWithAccessToken, toolCategory],
  );

  // Perform the actual Custom MCP config save (called after password is obtained)
  const performSaveCustomRemoteConfig = useCallback(
    async (masterPassword: string) => {
      try {
        setCreating(true);

        const proxyIdValue = await getServerInfo();
        if (!proxyIdValue) {
          throw new Error('Could not get proxy ID');
        }

        const operateParams: Parameters<typeof api.tools.operateTool>[0] = {
          handleType: 2,
          toolId: createdToolId!,
          proxyId: proxyIdValue.toString(),
          masterPwd: masterPassword,
          category: customRemoteTransportType === 'stdio' ? 5 : 2,
          lazyStartEnabled,
          publicAccess,
          anonymousAccess,
          anonymousRateLimit,
        };

        if (customRemoteTransportType === 'stdio') {
          const trimmedCwd = customRemoteConfig!.cwd?.trim();
          operateParams.stdioConfig = {
            command: customRemoteConfig!.command!,
            ...(customRemoteConfig!.args && { args: customRemoteConfig!.args }),
            ...(customRemoteConfig!.env && { env: customRemoteConfig!.env }),
            ...(trimmedCwd ? { cwd: trimmedCwd } : {}),
          };
        } else {
          operateParams.customRemoteConfig = customRemoteConfig as {
            url: string;
            headers: Record<string, string>;
          };
        }

        const response = await api.tools.operateTool(operateParams);
        if (response.data?.common?.code === 0) {
          const { toast } = await import('sonner');
          toast.success('Configuration saved');
          router.push('/dashboard/tool-configure');
        } else {
          throw new Error(response.data?.common?.message || 'Save failed');
        }
      } catch (error: any) {
        // Failed to save custom remote config:
        const errorMessage =
          error.response?.data?.common?.message ||
          error.message ||
          'Could not save configuration. Try again.';
        setCustomRemoteConfigError(errorMessage);

        const { toast } = await import('sonner');
        toast.error(errorMessage);
      } finally {
        setCreating(false);
      }
    },
    [
      customRemoteConfig,
      customRemoteTransportType,
      createdToolId,
      lazyStartEnabled,
      publicAccess,
      anonymousAccess,
      anonymousRateLimit,
      router,
      getServerInfo,
    ],
  );

  // Save Custom MCP configuration
  const handleSaveCustomRemoteConfig = useCallback(async () => {
    setCustomRemoteConfigError(null);

    if (!customRemoteConfig) {
      setCustomRemoteConfigError('Configuration is required');
      return;
    }

    // Validate based on transport type
    if (customRemoteTransportType === 'stdio') {
      // Stdio config validation
      if (!customRemoteConfig.command?.trim()) {
        setCustomRemoteConfigError('Command is required');
        return;
      }
    } else {
      // URL config validation (existing)
      if (!customRemoteConfig.url?.trim()) {
        setCustomRemoteConfigError('MCP URL is required');
        return;
      }
      try {
        new URL(customRemoteConfig.url!);
      } catch {
        setCustomRemoteConfigError('Invalid URL format');
        return;
      }
    }

    // Master password required regardless of allowUserInput true or false
    const cachedPassword = MasterPasswordManager.getCachedMasterPassword();
    if (cachedPassword) {
      await performSaveCustomRemoteConfig(cachedPassword);
    } else {
      setMasterPasswordAction('saveCustomRemote');
      setShowMasterPasswordDialog(true);
    }
  }, [customRemoteConfig, customRemoteTransportType, performSaveCustomRemoteConfig]);

  // Handle saving REST API configuration (category === 3)
  const handleSaveRestApiConfig = useCallback(async () => {
    // Clear previous error
    setRestApiConfigError(null);

    // Validate input not empty
    if (!restApiConfig.trim()) {
      setRestApiConfigError('REST API configuration is required');
      return;
    }

    try {
      setCreating(true);

      // Import validation utilities
      const {
        detectFormat,
        validateRestApiConfig: validateConfig,
        checkEnvironmentVariables,
        isOpenApiFormat,
        convertOpenApiToRestApiFormat,
      } = await import('@/lib/rest-api-utils');

      // Detect + parse configuration
      const detected = await detectFormat(restApiConfig);
      if (detected.format === 'unknown') {
        setRestApiConfigError(
          detected.error ||
            'Unable to detect configuration format. Please provide valid JSON or YAML.',
        );
        setCreating(false);
        return;
      }

      let parsedConfig: any = detected.data;

      // Check if OpenAPI format and convert
      if (isOpenApiFormat(parsedConfig)) {
        try {
          // Warn if primary security scheme is oauth2/openIdConnect (not supported in AuthConfig mapping)
          const schemes = parsedConfig?.components?.securitySchemes;
          let schemeName: string | undefined;
          const security = parsedConfig?.security;
          if (Array.isArray(security) && security.length > 0) {
            const firstReq = security[0];
            if (firstReq && typeof firstReq === 'object' && !Array.isArray(firstReq)) {
              schemeName = Object.keys(firstReq)[0];
            }
          }
          if (!schemeName && schemes && typeof schemes === 'object') {
            schemeName = Object.keys(schemes)[0];
          }
          const schemeType =
            schemeName && schemes && typeof schemes === 'object'
              ? schemes?.[schemeName]?.type
              : undefined;
          if (schemeType === 'oauth2' || schemeType === 'openIdConnect') {
            window.alert(
              `OpenAPI security scheme "${schemeName}" is "${schemeType}" and is not supported.\n` +
                `Auth will be set to "none"; please update auth manually after conversion.`,
            );
          }

          parsedConfig = convertOpenApiToRestApiFormat(parsedConfig);

          // Show converted JSON back in the input for users to fill placeholders
          setRestApiConfig(JSON.stringify(parsedConfig, null, 2));

          const placeholders = checkEnvironmentVariables(parsedConfig.auth);
          if (
            schemeType === 'oauth2' ||
            schemeType === 'openIdConnect' ||
            placeholders.length > 0
          ) {
            setRestApiConfigError(
              `OpenAPI has been converted to a REST API configuration. Please replace the placeholder values in "auth" (e.g., \${...}) with real credentials before saving.` +
                (schemeType === 'oauth2' || schemeType === 'openIdConnect'
                  ? `\n\nNote: OpenAPI security scheme "${schemeName}" is "${schemeType}" and is not supported for automatic mapping. Auth will be set to "none"; please configure auth manually.`
                  : '') +
                (placeholders.length > 0
                  ? `\n\nDetected placeholders: ${placeholders.join(', ')}`
                  : ''),
            );
          }

          // Stop here so user can review/edit the converted config before saving
          setCreating(false);
          return;
        } catch (error: any) {
          setRestApiConfigError(`Could not convert OpenAPI spec: ${error.message}`);
          setCreating(false);
          return;
        }
      }

      // Validate structure
      const validation = validateConfig(parsedConfig);
      if (!validation.valid) {
        setRestApiConfigError(validation.error || 'Invalid configuration');
        setCreating(false);
        return;
      }

      // Size + tools count limits
      const configString = JSON.stringify(parsedConfig);
      const configBytes = new TextEncoder().encode(configString).length;
      if (configBytes > 100 * 1024) {
        setRestApiConfigError(
          `Configuration is too large (${Math.ceil(configBytes / 1024)} KB). Maximum allowed size is 100 KB.`,
        );
        setCreating(false);
        return;
      }
      const toolCount = Array.isArray(parsedConfig?.tools) ? parsedConfig.tools.length : 0;
      if (toolCount > 50) {
        setRestApiConfigError(`Too many tools (${toolCount}). Maximum allowed tools count is 50.`);
        setCreating(false);
        return;
      }

      // Check for environment variables
      const envVars = checkEnvironmentVariables(parsedConfig);
      if (envVars.length > 0) {
        const confirmed = window.confirm(
          `Configuration contains environment variable placeholders: ${envVars.join(', ')}\n\n` +
            `Please ensure you have replaced these with actual values before proceeding.\n\n` +
            `Do you want to continue?`,
        );
        if (!confirmed) {
          setCreating(false);
          return;
        }
      }

      // Get proxyId
      const proxyIdValue = await getServerInfo();
      if (!proxyIdValue) {
        setRestApiConfigError('Unable to get server information');
        setCreating(false);
        return;
      }

      // Convert parsed config to JSON string
      const restApiConfigString = JSON.stringify(parsedConfig);

      // Check if master password is cached
      const cachedPassword = MasterPasswordManager.getCachedMasterPassword();

      if (!cachedPassword) {
        // No cached password - save pending data and show dialog
        setPendingRestApiData({
          configString: restApiConfigString,
          proxyId: proxyIdValue,
        });
        setCreating(false);
        setMasterPasswordAction('saveRestApi');
        setShowMasterPasswordDialog(true);
        return;
      }

      // Have password - proceed with save
      await performSaveRestApiConfig(cachedPassword, restApiConfigString, proxyIdValue);
    } catch (error: any) {
      // Failed to save REST API config:
      const errorMessage =
        error.response?.data?.common?.message ||
        error.response?.data?.message ||
        error.message ||
        'Could not save REST API configuration';
      setRestApiConfigError(errorMessage);
      setCreating(false);
    }
  }, [restApiConfig, getServerInfo, performSaveRestApiConfig]);

  // Master password handling
  const handleMasterPasswordConfirm = useCallback(
    async (password: string) => {
      try {
        setIsProcessingWithPassword(true);

        if (masterPasswordAction === 'create') {
          await performCreateTool(password);
        } else if (masterPasswordAction === 'update') {
          const stepMap = {
            1: 'functions' as const,
            2: 'permissions' as const,
            default: 'complete' as const,
          };
          const stepData = stepMap[safeCurrentStep as keyof typeof stepMap] || stepMap.default;
          await performUpdateTool(stepData, password, true); // shouldNavigate = true for manual Next clicks
        } else if (masterPasswordAction === 'delete') {
          await performDeleteTool(password);
        } else if (masterPasswordAction === 'saveRestApi') {
          // Save REST API configuration with the provided password
          if (pendingRestApiData) {
            await performSaveRestApiConfig(
              password,
              pendingRestApiData.configString,
              pendingRestApiData.proxyId,
            );
          }
        } else if (masterPasswordAction === 'saveCustomRemote') {
          // Save Custom MCP configuration with the provided password
          await performSaveCustomRemoteConfig(password);
        }

        // Store password in session for subsequent auto-saves
        setMasterPasswordForSession(password);

        setShowMasterPasswordDialog(false);
        setMasterPasswordAction(null);
      } catch (error) {
        // Error with master password:
      } finally {
        setIsProcessingWithPassword(false);
      }
    },
    [
      masterPasswordAction,
      performCreateTool,
      performUpdateTool,
      performDeleteTool,
      performSaveRestApiConfig,
      pendingRestApiData,
      safeCurrentStep,
    ],
  );

  // Tool creation handler
  const handleCreateTool = useCallback(async () => {
    if (!currentServer) return;

    const allowUserInputValue = configMode === 'user' ? 1 : 0;

    // User mode: directly create with master password, then go to list
    if (allowUserInputValue === 1) {
      setMasterPasswordAction('create');
      setShowMasterPasswordDialog(true);
      return;
    }

    // Owner mode with OAuth: initiate OAuth flow
    if (toolTemplate?.oAuthConfig) {
      const {
        buildAuthorizationUrl,
        saveOAuthSession,
        getOAuthPKCEConfig,
        generateOAuthPKCEParams,
      } = await import('@/lib/oauth-authorization');

      const isCustomCredentials = credentialSource === 'custom';
      const requiresCustomApp = requiresCustomOAuthApp(toolTemplate);
      let redirectUri = `${window.location.origin}/dashboard/tool-configure`;
      if (toolTemplate.authType === ServerAuthType.CanvaAuth) {
        if (window.location.hostname === 'localhost') {
          redirectUri = redirectUri.replace('localhost', '127.0.0.1');
        }
      }

      if (!isCustomCredentials && requiresCustomApp) {
        toast.error('This tool requires your own OAuth app credentials');
        return;
      }

      let clientId = undefined;
      let clientSecret = undefined;
      let resolvedOAuthConfig = toolTemplate.oAuthConfig;
      let pkceVerifier: string | undefined;
      if (isCustomCredentials) {
        clientId = getCredentialValueByTemplateKey(
          toolTemplate.credentials,
          dynamicCredentials,
          'YOUR_CLIENT_ID',
        );
        clientSecret = getCredentialValueByTemplateKey(
          toolTemplate.credentials,
          dynamicCredentials,
          'YOUR_CLIENT_SECRET',
        );

        if (
          !clientId ||
          !clientSecret ||
          clientId === '' ||
          clientSecret === '' ||
          clientId === 'YOUR_CLIENT_ID' ||
          clientSecret === 'YOUR_CLIENT_SECRET'
        ) {
          toast.error('Custom credentials require both Client ID and Client Secret');
          return;
        }

        try {
          const endpointOverrides = getOAuthEndpointOverridesFromCredentials(
            toolTemplate,
            dynamicCredentials,
          );
          const resolvedEndpoints = resolveOAuthEndpoints(
            toolTemplate.oAuthConfig,
            endpointOverrides,
          );
          resolvedOAuthConfig = {
            ...toolTemplate.oAuthConfig,
            authorizationUrl: resolvedEndpoints.authorizationUrl,
            tokenUrl: resolvedEndpoints.tokenUrl,
          };
        } catch (error) {
          toast.error(
            error instanceof Error ? error.message : 'Invalid OAuth endpoint configuration',
          );
          return;
        }
      }

      try {
        const pkceConfig = getOAuthPKCEConfig(resolvedOAuthConfig);
        if (pkceConfig.required) {
          const pkceParams = await generateOAuthPKCEParams(pkceConfig.method);
          pkceVerifier = pkceParams.codeVerifier;

          resolvedOAuthConfig = {
            ...resolvedOAuthConfig,
            extraParams: {
              ...(resolvedOAuthConfig.extraParams || {}),
              code_challenge: pkceParams.codeChallenge,
              code_challenge_method: pkceConfig.method,
            },
          };
        }
      } catch (error) {
        toast.error(error instanceof Error ? error.message : 'Invalid PKCE configuration');
        return;
      }

      // Save session data to sessionStorage, returns random state
      const state = saveOAuthSession({
        templateId: toolTemplate.toolTmplId,
        serverType: serverType || '',
        tempId: toolId,
        credentialSource,
        credentials: isCustomCredentials ? dynamicCredentials : undefined,
        allowUserInput: allowUserInputValue,
        redirectUri,
        pkceVerifier,
        lazyStartEnabled,
        publicAccess,
        anonymousAccess,
        anonymousRateLimit,
      });

      const authUrl = buildAuthorizationUrl(resolvedOAuthConfig, clientId, redirectUri, state);

      window.location.href = authUrl;
      return;
    }

    // Non-OAuth: show master password dialog
    setMasterPasswordAction('create');
    setShowMasterPasswordDialog(true);
  }, [
    currentServer,
    configMode,
    toolTemplate,
    credentialSource,
    serverType,
    toolId,
    dynamicCredentials,
    lazyStartEnabled,
    publicAccess,
    anonymousAccess,
    anonymousRateLimit,
  ]);

  const handleUpdateTool = useCallback(
    async (
      stepData: 'functions' | 'permissions' | 'complete',
      isAutoSave = false,
      shouldNavigate = false,
    ) => {
      if (!currentServer || !createdToolId) return;

      // Auto-save mode: call API directly, no master password needed
      if (isAutoSave) {
        await performUpdateTool(stepData, '', shouldNavigate);
        return;
      }

      // Manual save mode: use session password if available, otherwise request input
      if (masterPasswordForSession) {
        await performUpdateTool(stepData, masterPasswordForSession, shouldNavigate);
      } else if (stepData === 'complete' || safeCurrentStep > 0) {
        await performUpdateTool(stepData, '', shouldNavigate);
      } else {
        setMasterPasswordAction('update');
        setShowMasterPasswordDialog(true);
      }
    },
    [currentServer, createdToolId, masterPasswordForSession, performUpdateTool, safeCurrentStep],
  );

  // Render step component
  const renderStepComponent = () => {
    const stepConfig = steps[safeCurrentStep];
    if (!stepConfig) return null;

    const commonProps = {
      server: currentServer!,
      onUpdate: handleServerUpdate,
      toolTemplate: toolTemplate || undefined,
    };

    switch (stepConfig.id) {
      case 'credentials':
        return (
          <CredentialsTab
            {...commonProps}
            dynamicCredentials={dynamicCredentials}
            onCredentialUpdate={handleCredentialUpdate}
            onDelete={isEdit ? handleDeleteTool : undefined}
            isEdit={isEdit}
            allowUserInputMode={allowUserInput}
            lazyStartEnabled={lazyStartEnabled}
            onLazyStartEnabledChange={setLazyStartEnabled}
            publicAccess={publicAccess}
            onPublicAccessChange={setPublicAccess}
            anonymousAccess={anonymousAccess}
            onAnonymousAccessChange={setAnonymousAccess}
            anonymousRateLimit={anonymousRateLimit}
            onAnonymousRateLimitChange={setAnonymousRateLimit}
            configMode={configMode}
            onConfigModeChange={setConfigMode}
            credentialSource={credentialSource}
            onCredentialSourceChange={setCredentialSource}
            isSetupMode={mode === 'setup' && !toolCreated}
          />
        );
      case 'functions':
        return (
          <FunctionsTab
            {...commonProps}
            onAutoSave={isEdit ? handleFunctionsAutoSave : undefined}
            cacheHealth={cacheHealth}
            cachePolicies={cachePolicies}
            isLoadingCache={isLoadingCache}
            onToolCachePolicyChange={handleToolCachePolicyChange}
            onPromptCachePolicyChange={handlePromptCachePolicyChange}
            onResourceCachePolicyChange={handleResourceCachePolicyChange}
            onRefreshCacheData={loadCacheData}
          />
        );
      case 'permissions':
        return <PermissionsTab {...commonProps} />;
      default:
        return null;
    }
  };

  // Loading state
  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 dark:border-blue-400 mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">Loading tool template...</p>
        </div>
      </div>
    );
  }

  // If no current server config, show loading state (usually won't happen)
  if (!currentServer) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 dark:border-blue-400 mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">Initializing tool configuration...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header navigation */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border dark:border-gray-700 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Link
              href="/dashboard/tool-configure"
              className="inline-flex items-center text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
            >
              <ArrowLeft className="w-4 h-4 mr-1" />
              Back
            </Link>
            <div className="h-4 w-px bg-gray-300 dark:bg-gray-600" />
            <div
              className={`w-10 h-10 rounded-lg flex items-center justify-center ${toolMetadata.bgColor}`}
            >
              <DynamicIcon iconName={toolMetadata.icon} className="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 className="text-lg font-bold text-gray-900 dark:text-white">
                {currentServer.name}
              </h1>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {currentServer.name} • {getModeTitle()}
              </p>
            </div>
          </div>
          <div className="text-xs text-gray-500 dark:text-gray-400">
            Progress: {Math.round(progress)}%
          </div>
        </div>
      </div>

      {/* Step navigation */}
      {/* Hide step navigation for allowUserInput tools in edit mode, REST API tools (category === 3), and Skills tools (category === 4) in credentials mode */}
      {mode === 'setup' &&
        !(allowUserInput && isEdit) &&
        !((toolCategory === 3 || toolCategory === 4) && isEdit) && (
          <div className="bg-white dark:bg-gray-800 rounded-lg border dark:border-gray-700 p-3">
            <div className={`grid gap-2 mb-3 grid-cols-${steps.length}`}>
              {steps.map((step, index) => {
                const Icon = step.icon;
                const isActive = index === safeCurrentStep;
                const isCompleted = index < safeCurrentStep;
                const isClickable = isCompleted || isActive;

                return (
                  <button
                    key={step.id}
                    onClick={() => isClickable && updateStepInUrl(index)}
                    disabled={!isClickable}
                    className={`
                    p-3 rounded-md border transition-all duration-200 text-left
                    ${
                      isActive
                        ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30'
                        : isCompleted
                          ? 'border-green-500 bg-green-50 dark:bg-green-900/30 cursor-pointer hover:bg-green-100 dark:hover:bg-green-900/50'
                          : isClickable
                            ? 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500 cursor-pointer'
                            : 'border-gray-200 dark:border-gray-700 opacity-50 cursor-not-allowed'
                    }
                  `}
                  >
                    <div className="flex items-center space-x-2">
                      <div
                        className={`
                      w-6 h-6 rounded-md flex items-center justify-center
                      ${
                        isCompleted
                          ? 'bg-green-500 text-white'
                          : isActive
                            ? `${step.bgColor} text-white`
                            : 'bg-gray-200 dark:bg-gray-700 text-gray-400 dark:text-gray-500'
                      }
                    `}
                      >
                        {isCompleted ? <Check className="w-3 h-3" /> : <Icon className="w-3 h-3" />}
                      </div>
                      <div>
                        <h4
                          className={`text-sm font-medium ${
                            isActive
                              ? 'text-blue-900 dark:text-blue-200'
                              : isCompleted
                                ? 'text-green-900 dark:text-green-200'
                                : 'text-gray-700 dark:text-gray-300'
                          }`}
                        >
                          {step.title}
                        </h4>
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                          {step.description}
                        </p>
                      </div>
                    </div>
                  </button>
                );
              })}
            </div>

            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
              <div
                className="bg-blue-500 h-1.5 rounded-full transition-all duration-500"
                style={{ width: `${progress}%` }}
              />
            </div>
          </div>
        )}

      {/* Main content */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border dark:border-gray-700">
        {/* Hide setup header/buttons for allowUserInput tools in edit mode, REST API tools (category === 3), and Skills tools (category === 4) in setup mode */}
        {mode === 'setup' &&
          !(allowUserInput && isEdit) &&
          !((toolCategory === 3 || toolCategory === 4) && isEdit) && (
            <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-lg font-bold text-gray-900 dark:text-white">
                    {steps[safeCurrentStep]?.title || 'Loading...'}
                  </h2>
                  <p className="text-xs text-gray-600 dark:text-gray-400 mt-0.5">
                    {steps[safeCurrentStep]?.id === 'credentials' &&
                      `Configure authentication for ${currentServer.name}`}
                    {steps[safeCurrentStep]?.id === 'functions' &&
                      `Select functions to enable for ${currentServer.name}`}
                    {steps[safeCurrentStep]?.id === 'permissions' &&
                      `Set data access permissions for ${currentServer.name}`}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {safeCurrentStep > 0 && !(safeCurrentStep === 1 && toolCreated) && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => updateStepInUrl(safeCurrentStep - 1)}
                      disabled={creating}
                    >
                      <ArrowLeft className="w-3 h-3 mr-1" />
                      Previous
                    </Button>
                  )}

                  {/* Hide save/next button for functions step only if it's in editing mode (auto-save mode) */}
                  {!(steps[safeCurrentStep]?.id === 'functions' && isEdit) && (
                    <>
                      {safeCurrentStep === steps.length - 1 ? (
                        <Button
                          size="sm"
                          onClick={() => handleUpdateTool('complete')}
                          disabled={creating}
                          className="bg-green-600 hover:bg-green-700"
                        >
                          {creating ? (
                            <>
                              <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                              Completing...
                            </>
                          ) : (
                            <>
                              <Check className="w-3 h-3 mr-1" />
                              Complete Setup
                            </>
                          )}
                        </Button>
                      ) : (
                        <Button
                          size="sm"
                          onClick={() => {
                            if (safeCurrentStep === 0 && !toolCreated) {
                              handleCreateTool();
                            } else if (safeCurrentStep === 1 && toolCreated) {
                              handleUpdateTool('functions', false, true); // shouldNavigate = true
                            } else if (safeCurrentStep === 2 && toolCreated) {
                              handleUpdateTool('permissions', false, true); // shouldNavigate = true
                            } else {
                              updateStepInUrl(safeCurrentStep + 1);
                            }
                          }}
                          disabled={creating}
                        >
                          {creating && safeCurrentStep === 0 ? (
                            <>
                              <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                              Creating...
                            </>
                          ) : creating ? (
                            <>
                              <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                              Updating...
                            </>
                          ) : (
                            <>
                              Next
                              <ArrowLeft className="w-3 h-3 ml-1 rotate-180" />
                            </>
                          )}
                        </Button>
                      )}
                    </>
                  )}

                  {/* Show auto-save indicator for functions step only if it's in editing mode */}
                  {steps[safeCurrentStep]?.id === 'functions' && isEdit && (
                    <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                      {creating ? (
                        <>
                          <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-muted-foreground"></div>
                          Saving...
                        </>
                      ) : (
                        'Changes are saved automatically'
                      )}
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

        {/* Single-mode save button */}
        {mode !== 'setup' &&
          !(mode === 'functions' && isEdit) &&
          !(
            (isCustomMcpCategory(toolCategory) || toolCategory === 3 || toolCategory === 4) &&
            mode === 'credentials' &&
            isEdit
          ) && (
            <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 flex justify-end">
              <Button
                size="sm"
                onClick={() => {
                  if (!toolCreated) {
                    handleCreateTool();
                  } else {
                    handleUpdateTool('complete');
                  }
                }}
                disabled={creating}
                className="bg-green-600 hover:bg-green-700"
              >
                {creating ? (
                  <>
                    <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                    {!toolCreated ? 'Creating...' : 'Saving...'}
                  </>
                ) : (
                  <>
                    <Check className="w-3 h-3 mr-1" />
                    {!toolCreated ? 'Create Tool' : 'Save Changes'}
                  </>
                )}
              </Button>
            </div>
          )}

        {/* Single-mode auto-save hint */}
        {mode === 'functions' && isEdit && (
          <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 flex justify-end">
            <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
              {creating ? (
                <>
                  <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-muted-foreground"></div>
                  Saving...
                </>
              ) : (
                'Changes are saved automatically'
              )}
            </div>
          </div>
        )}

        {/* REST API tool (category === 3) header with Save Changes button - only for credentials/settings mode */}
        {toolCategory === 3 && mode === 'credentials' && isEdit && (
          <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 flex justify-end">
            <Button
              size="sm"
              onClick={handleSaveRestApiConfig}
              disabled={creating}
              className="bg-green-600 hover:bg-green-700"
            >
              {creating ? (
                <>
                  <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                  Saving...
                </>
              ) : (
                <>
                  <Check className="w-3 h-3 mr-1" />
                  Save Changes
                </>
              )}
            </Button>
          </div>
        )}

        {isCustomMcpCategory(toolCategory) && mode === 'credentials' && isEdit && (
          <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 flex justify-end">
            <Button
              size="sm"
              onClick={handleSaveCustomRemoteConfig}
              disabled={creating || !customRemoteConfig}
              className="bg-slate-900 hover:bg-slate-800 text-white dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900"
            >
              {creating ? (
                <>
                  <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
                  Saving...
                </>
              ) : (
                <>
                  <Check className="w-3 h-3 mr-1" />
                  Save Changes
                </>
              )}
            </Button>
          </div>
        )}

        {/* Skills tool (category === 4) header with Save Changes button - only for credentials/settings mode */}
        {toolCategory === 4 && mode === 'credentials' && isEdit && (
          <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 flex justify-end">
            <Button
              size="sm"
              onClick={handleSaveSkillsConfig}
              disabled={creating}
              className="bg-green-600 hover:bg-green-700"
            >
              {creating ? (
                <>
                  <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                  Saving...
                </>
              ) : (
                <>
                  <Check className="w-3 h-3 mr-1" />
                  Save Changes
                </>
              )}
            </Button>
          </div>
        )}

        {/* Component content */}
        <div className="p-4">
          {isCustomMcpCategory(toolCategory) && mode === 'credentials' && isEdit ? (
            customRemoteConfig ? (
              <CustomRemoteConfigTab
                config={customRemoteConfig}
                transportType={customRemoteTransportType}
                onConfigChange={setCustomRemoteConfig}
                error={customRemoteConfigError}
                onDelete={handleDeleteTool}
                serverName={currentServer?.name}
                lazyStartEnabled={lazyStartEnabled}
                onLazyStartEnabledChange={setLazyStartEnabled}
                publicAccess={publicAccess}
                onPublicAccessChange={setPublicAccess}
                anonymousAccess={anonymousAccess}
                onAnonymousAccessChange={setAnonymousAccess}
                anonymousRateLimit={anonymousRateLimit}
                onAnonymousRateLimitChange={setAnonymousRateLimit}
                allowUserInput={allowUserInput}
                isAdmin={currentUserRole === 'Admin' || currentUserRole === 'Owner'}
              />
            ) : (
              <div className="flex items-center justify-center py-12">
                <div className="text-center space-y-4">
                  <Lock className="h-12 w-12 text-gray-400 dark:text-gray-500 mx-auto" />
                  <div>
                    <p className="text-gray-900 dark:text-gray-100 font-medium mb-2">
                      {encryptedCustomRemoteConfig
                        ? 'Configuration Encrypted'
                        : 'Loading Configuration'}
                    </p>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      {encryptedCustomRemoteConfig
                        ? 'This configuration requires your access token to decrypt.'
                        : 'Please wait while we load the configuration...'}
                    </p>
                  </div>
                  {encryptedCustomRemoteConfig && (
                    <Button
                      onClick={() => setShowAccessTokenDialog(true)}
                      className="bg-slate-900 hover:bg-slate-800 text-white dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900"
                    >
                      Enter Access Token
                    </Button>
                  )}
                </div>
              </div>
            )
          ) : toolCategory === 4 && mode === 'credentials' && isEdit ? (
            <SkillsConfigTab
              serverId={toolId}
              serverName={currentServer?.name}
              lazyStartEnabled={lazyStartEnabled}
              onLazyStartEnabledChange={setLazyStartEnabled}
              publicAccess={publicAccess}
              onPublicAccessChange={setPublicAccess}
              anonymousAccess={anonymousAccess}
              onAnonymousAccessChange={setAnonymousAccess}
              anonymousRateLimit={anonymousRateLimit}
              onAnonymousRateLimitChange={setAnonymousRateLimit}
              onDelete={handleDeleteTool}
              allowUserInput={allowUserInput}
            />
          ) : toolCategory === 3 && mode === 'credentials' && isEdit ? (
            <RestApiConfigTab
              config={restApiConfig}
              onConfigChange={(value) => {
                setRestApiConfig(value);
                // Clear error when user starts typing
                if (restApiConfigError) {
                  setRestApiConfigError(null);
                }
              }}
              error={restApiConfigError}
              onDelete={handleDeleteTool}
              serverName={currentServer?.name}
              lazyStartEnabled={lazyStartEnabled}
              onLazyStartEnabledChange={setLazyStartEnabled}
              publicAccess={publicAccess}
              onPublicAccessChange={setPublicAccess}
              anonymousAccess={anonymousAccess}
              onAnonymousAccessChange={setAnonymousAccess}
              anonymousRateLimit={anonymousRateLimit}
              onAnonymousRateLimitChange={setAnonymousRateLimit}
              allowUserInput={allowUserInput}
            />
          ) : (
            renderStepComponent()
          )}
        </div>
      </div>

      {/* Master Password Dialog */}
      <MasterPasswordDialog
        open={showMasterPasswordDialog}
        onOpenChange={(open) => {
          setShowMasterPasswordDialog(open);
          if (!open) {
            setMasterPasswordAction(null);
            setPendingRestApiData(null);
          }
        }}
        onConfirm={handleMasterPasswordConfirm}
        title={
          masterPasswordAction === 'create'
            ? 'Create Tool - Master Password Required'
            : masterPasswordAction === 'delete'
              ? 'Delete Tool - Master Password Required'
              : masterPasswordAction === 'saveRestApi'
                ? 'Save REST API Configuration - Master Password Required'
                : masterPasswordAction === 'saveCustomRemote'
                  ? 'Save Custom MCP Configuration - Master Password Required'
                  : 'Update Tool - Master Password Required'
        }
        description={
          masterPasswordAction === 'create'
            ? 'Please enter your master password to create the tool with encrypted credentials.'
            : masterPasswordAction === 'delete'
              ? 'Please enter your master password to delete this tool.'
              : masterPasswordAction === 'saveRestApi'
                ? 'Please enter your master password to save the REST API configuration.'
                : masterPasswordAction === 'saveCustomRemote'
                  ? 'Please enter your master password to save the Custom MCP configuration.'
                  : 'Please enter your master password to update the tool configuration.'
        }
        isLoading={isProcessingWithPassword}
        showForgotPassword={isOwner}
      />

      {/* Access Token Dialog */}
      <AccessTokenDialog
        open={showAccessTokenDialog}
        onOpenChange={(open) => {
          setShowAccessTokenDialog(open);
          if (!open) {
            setAccessTokenError(null);
          }
        }}
        onConfirm={handleDecryptCustomRemoteConfig}
        isLoading={isDecrypting}
        error={accessTokenError || undefined}
        title="Access Token Required"
        description="This configuration is encrypted. Please enter your access token to decrypt and view the configuration."
      />
    </div>
  );
}
