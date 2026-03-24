'use client';

import {
  Key,
  Calendar,
  ChevronDown,
  Globe,
  Github,
  Mail,
  Database,
  Shield,
  User,
  Crown,
  AlertTriangle,
  Download,
  Edit,
  Trash2,
  CheckCircle,
  Copy,
  FileDown,
  Search,
  X,
} from 'lucide-react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useState, useEffect, useMemo } from 'react';
import { toast } from 'sonner';
import { getPricingUrl } from '@/lib/plan-config';

import { DestroyTokenDialog } from '@/components/destroy-token-dialog';
import { EditTokenDialog } from '@/components/edit-token-dialog';
import { TagInput } from '@/components/tag-input';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Collapsible, CollapsibleContent } from '@/components/ui/collapsible';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  ScrollableDialogContent,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { cn } from '@/lib/utils';
import { MasterPasswordManager } from '@/lib/crypto';
import { MasterPasswordDialog } from '@/components/master-password-dialog';
import { useUserRole } from '@/hooks/use-user-role';

// Available tools with sub-functions
const availableTools = [
  {
    id: 'web-search',
    name: 'Web Search',
    icon: Globe,
    enabled: true,
    subFunctions: [
      { name: 'Google Search', enabled: true },
      { name: 'Bing Search', enabled: false },
      { name: 'DuckDuckGo', enabled: true },
    ],
  },
  {
    id: 'github',
    name: 'GitHub',
    icon: Github,
    enabled: true,
    subFunctions: [
      { name: 'Repository Access', enabled: true },
      { name: 'Issue Management', enabled: true },
      { name: 'Pull Requests', enabled: false },
      { name: 'Webhooks', enabled: false },
    ],
  },
  {
    id: 'email',
    name: 'Email',
    icon: Mail,
    enabled: false,
    subFunctions: [
      { name: 'Send Email', enabled: false },
      { name: 'Read Email', enabled: false },
      { name: 'Manage Folders', enabled: false },
    ],
  },
  {
    id: 'database',
    name: 'Database',
    icon: Database,
    enabled: true,
    subFunctions: [
      { name: 'Read Operations', enabled: true },
      { name: 'Write Operations', enabled: false },
      { name: 'Schema Management', enabled: true },
      { name: 'Backup & Restore', enabled: false },
    ],
  },
];

// Helper function to convert role number to string
const getRoleFromNumber = (roleNumber: number): string => {
  switch (roleNumber) {
    case 1:
      return 'Owner';
    case 2:
      return 'Admin';
    case 3:
      return 'Member';
    case 4:
      return 'Guest';
    default:
      return 'Member';
  }
};

// Helper function to check if current user can manage a token
const canManageToken = (currentUserRole: string, tokenRole: string | number): boolean => {
  if (currentUserRole === 'Owner') {
    // Owner can manage Admin and Member tokens, but cannot delete Owner tokens
    return true;
  }

  if (currentUserRole === 'Admin') {
    // Admin can only manage Member tokens (role = 3 or 'Member')
    return tokenRole === 'Member' || tokenRole === 3;
  }

  return false; // Member and Guest cannot manage any token
};

// Helper function to check if current user can see a token
const canViewToken = (currentUserRole: string, tokenRole: string | number): boolean => {
  if (currentUserRole === 'Owner') {
    return true; // Owner can see all tokens
  }

  if (currentUserRole === 'Admin') {
    // Admin can see all tokens but can only manage Member tokens
    return true;
  }

  if (currentUserRole === 'Member') {
    // Member can only see Member-level tokens (role = 3 or 'Member')
    return tokenRole === 'Member' || tokenRole === 3;
  }

  return false; // Guest cannot see any tokens
};

export default function MembersPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [tokens, setTokens] = useState<any[]>([]);
  const [showTokenDialog, setShowTokenDialog] = useState(false);
  const [expandedTools, setExpandedTools] = useState<Set<string>>(new Set());
  const [isLoading, setIsLoading] = useState(false);
  const [serverInfo, setServerInfo] = useState<any>(null);
  const [proxyId, setProxyId] = useState<number | null>(null);

  // Check if Owner has tokens for Desk usage
  const hasRegularTokens = tokens.some(
    (token) => token.role === 'Member' || token.role === 'Admin',
  );

  // Prefer user role from localStorage
  const getCurrentUserRole = () => {
    try {
      const selectedServerData = localStorage.getItem('selectedServer');
      if (selectedServerData) {
        const parsedServer = JSON.parse(selectedServerData);
        return parsedServer.role || 'Member';
      }
    } catch {
      // ignore corrupted localStorage
    }
    return serverInfo?.role || 'Member';
  };

  const currentUserRole = getCurrentUserRole(); // Get current user role
  const isOwner = currentUserRole === 'Owner';
  const isAdmin = currentUserRole === 'Admin';
  const isMember = currentUserRole === 'Member';

  // Form state
  const [newTokenName, setNewTokenName] = useState('New API Token');
  const [newTokenRole, setNewTokenRole] = useState<'Admin' | 'Member'>('Member');
  const [newTokenEmail, setNewTokenEmail] = useState('');
  const [newTokenPurpose, setNewTokenPurpose] = useState('');
  const [newTokenNamespace, setNewTokenNamespace] = useState('default');
  const [newTokenTags, setNewTokenTags] = useState<string[]>([]);
  const [newTokenExpiry, setNewTokenExpiry] = useState<'never' | '30d' | '90d' | '1y'>('never');
  const [newTokenRateLimit, setNewTokenRateLimit] = useState(100);
  const [newTokenTools, setNewTokenTools] = useState(availableTools);
  const [showMasterPasswordDialog, setShowMasterPasswordDialog] = useState(false);
  const [masterPasswordAction, setMasterPasswordAction] = useState<{
    type: 'create' | 'edit' | 'delete';
    tokenData?: any;
  } | null>(null);
  const [isProcessingWithPassword, setIsProcessingWithPassword] = useState(false);
  const [showLimitDialog, setShowLimitDialog] = useState(false);
  const [selectedTokenIds, setSelectedTokenIds] = useState<Set<string>>(new Set());
  const [showBulkUpdateDialog, setShowBulkUpdateDialog] = useState(false);
  const [bulkPermissionsSourceTokenId, setBulkPermissionsSourceTokenId] =
    useState<string>('__none__');
  const [bulkPermissionsMode, setBulkPermissionsMode] = useState<'replace' | 'merge'>('replace');
  const [bulkNamespace, setBulkNamespace] = useState('');
  const [bulkTagsMode, setBulkTagsMode] = useState<'replace' | 'add' | 'remove' | 'clear'>('add');
  const [bulkTags, setBulkTags] = useState<string[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [filterNamespace, setFilterNamespace] = useState<string | null>(null);
  const [filterTags, setFilterTags] = useState<string[]>([]);
  const [filterRole, setFilterRole] = useState<string | null>(null);
  const [tagsFilterOpen, setTagsFilterOpen] = useState(false);
  const [isBulkUpdating, setIsBulkUpdating] = useState(false);

  const handleCopyToken = () => {
    if (createdTokenData?.value) {
      navigator.clipboard.writeText(createdTokenData.value);
      toast.success('Token copied to clipboard');
      setTokenActionTaken(true);
    }
  };

  const handleDownloadToken = () => {
    if (createdTokenData?.value && createdTokenData?.name) {
      const tokenContent = `# Kimbap.io Access Token
# Token Name: ${createdTokenData.name}
# Created: ${new Date().toISOString()}
#
# IMPORTANT: Keep this token secure and never commit it to version control
#
${createdTokenData.value}

# Usage Instructions:
# 1. Store this token in your application's environment variables
# 2. Use it as a Bearer token in the Authorization header
# 3. Example: Authorization: Bearer ${createdTokenData.value}
`;
      const blob = new Blob([tokenContent], { type: 'text/plain' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.style.display = 'none';
      a.href = url;
      a.download = `kimbap-token-${createdTokenData.name
        .replace(/[^a-zA-Z0-9]/g, '-')
        .toLowerCase()}.txt`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      setTokenActionTaken(true);
    }
  };
  const [availableScopes, setAvailableScopes] = useState<any[]>([]);
  const [selectedScopes, setSelectedScopes] = useState<Map<string, any>>(new Map());
  const [isLoadingScopes, setIsLoadingScopes] = useState(false);
  const [dialogStep, setDialogStep] = useState<'create' | 'success'>('create');
  const [createdTokenData, setCreatedTokenData] = useState<{
    value: string;
    name: string;
  } | null>(null);
  const [tokenActionTaken, setTokenActionTaken] = useState(false);

  // Load server info and tokens
  // Check if we should open the create token dialog from URL parameter
  useEffect(() => {
    const action = searchParams.get('action');
    if (action === 'create') {
      setShowTokenDialog(true);
      // Remove the action parameter from URL
      router.replace('/dashboard/members');
    }
  }, [searchParams, router]);

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    const selectedServer = localStorage.getItem('selectedServer');
    if (selectedServer) {
      const parsedServerInfo = JSON.parse(selectedServer);
      setServerInfo(parsedServerInfo);
    }
    loadServerInfoAndTokens();
  }, []);

  const loadScopes = async () => {
    if (!proxyId || isLoadingScopes) return;

    try {
      setIsLoadingScopes(true);
      const { api } = await import('@/lib/api-client');

      // Use protocol 10009 to get available scopes
      const response = await api.scopes.getScopes({
        proxyId: Number(proxyId),
      });

      if (response.data?.data?.scopes) {
        setAvailableScopes(response.data.data.scopes);

        // Auto-select scopes that are enabled by default from API
        const newSelectedScopes = new Map();
        response.data.data.scopes.forEach((scope: any) => {
          if (scope.enabled === true) {
            newSelectedScopes.set(scope.toolId, {
              ...scope,
              enabled: true,
              toolFuncs:
                scope.toolFuncs?.map((func: any) => ({
                  ...func,
                  enabled: func.enabled === true,
                })) || [],
              toolResources:
                scope.toolResources?.map((res: any) => ({
                  ...res,
                  enabled: res.enabled === true,
                })) || [],
            });
          }
        });
        setSelectedScopes(newSelectedScopes);
      }
    } catch (error) {
      setAvailableScopes([]);
    } finally {
      setIsLoadingScopes(false);
    }
  };

  const loadServerInfoAndTokens = async () => {
    // Prevent duplicate requests
    if (isLoading) return;

    try {
      setIsLoading(true);
      const { api } = await import('@/lib/api-client');

      // Get server info first to get proxyId
      const serverInfoResponse = await api.servers.getInfo();
      const serverProxyId = serverInfoResponse.data?.data?.proxyId;

      // Use role info from selectedServer, not from API response
      const selectedServerData = localStorage.getItem('selectedServer');
      let currentUserRole = 'Member'; // Default value

      if (selectedServerData) {
        const parsedServer = JSON.parse(selectedServerData);
        currentUserRole = parsedServer.role || 'Member';
      }

      // Update serverInfo, preserve role info from localStorage
      if (serverInfoResponse.data?.data) {
        const updatedServerInfo = {
          ...serverInfo,
          role: currentUserRole, // Use role from localStorage
          proxyId: serverProxyId,
        };
        setServerInfo(updatedServerInfo);
      }

      if (!serverProxyId) {
        setTokens([]);
        return;
      }

      setProxyId(serverProxyId);

      // Use protocol 10007 to get access tokens
      const response = await api.tokens.getAccessTokens({
        proxyId: Number(serverProxyId),
      });

      if (response.data?.data?.tokenList) {
        setTokens(transformTokenList(response.data.data.tokenList));
      } else {
        setTokens([]);
      }
    } catch (error) {
      setTokens([]);
    } finally {
      setIsLoading(false);
    }
  };

  const toggleToolExpansion = (toolId: string) => {
    const newExpanded = new Set(expandedTools);
    if (newExpanded.has(toolId)) {
      newExpanded.delete(toolId);
    } else {
      newExpanded.add(toolId);
    }
    setExpandedTools(newExpanded);
  };

  const handleToolToggle = (toolId: string, enabled: boolean) => {
    setNewTokenTools((prev) =>
      prev.map((tool) => (tool.id === toolId ? { ...tool, enabled } : tool)),
    );
  };

  const handleSubFunctionToggle = (toolId: string, subFunctionName: string, enabled: boolean) => {
    setNewTokenTools((prev) =>
      prev.map((tool) =>
        tool.id === toolId
          ? {
              ...tool,
              subFunctions: tool.subFunctions.map((subFunc) =>
                subFunc.name === subFunctionName ? { ...subFunc, enabled } : subFunc,
              ),
            }
          : tool,
      ),
    );
  };

  const resetForm = () => {
    setNewTokenName('New API Token');
    setNewTokenRole('Member');
    setNewTokenEmail('');
    setNewTokenPurpose('');
    setNewTokenNamespace('default');
    setNewTokenTags([]);
    setNewTokenExpiry('never');
    setNewTokenRateLimit(100);
    setNewTokenTools(availableTools.map((tool) => ({ ...tool, enabled: false })));
    setExpandedTools(new Set());
    setSelectedScopes(new Map());
    setDialogStep('create');
    setCreatedTokenData(null);
    setTokenActionTaken(false);
  };

  const transformTokenList = (tokenList: any[]) => {
    return tokenList.map((token: any) => ({
      id: token.tokenId,
      name: token.name,
      token: `kimbap_${token.tokenId}`,
      displayToken: `kimbap_****...${token.tokenId.slice(-4)}`,
      role: getRoleFromNumber(token.role),
      roleNumber: token.role,
      email: '',
      purpose: token.notes || '',
      namespace: token.namespace || 'default',
      tags: token.tags || [],
      createdAt: new Date(token.createAt * 1000).toLocaleDateString(),
      expiresAt: token.expireAt ? new Date(token.expireAt * 1000).toLocaleDateString() : 'Never',
      rateLimit: token.rateLimit || 100,
      lastUsed: token.lastUsed ? new Date(token.lastUsed * 1000).toLocaleString() : 'Never used',
      tools: token.toolList || [],
      rawData: token,
    }));
  };

  const handleCreateToken = async () => {
    if (!proxyId) return;

    try {
      // Show master password dialog
      setMasterPasswordAction({ type: 'create' });
      setShowMasterPasswordDialog(true);
      return;
    } catch (error: any) {
      // Error handling is done by api-client interceptor
    } finally {
      setIsLoading(false);
    }
    setIsLoading(false);
  };

  const performCreateToken = async (masterPassword: string) => {
    if (!proxyId) return;

    try {
      const { api } = await import('@/lib/api-client');

      const expiresAt =
        newTokenExpiry === 'never'
          ? 0
          : newTokenExpiry === '30d'
            ? Math.floor((Date.now() + 30 * 24 * 60 * 60 * 1000) / 1000)
            : newTokenExpiry === '90d'
              ? Math.floor((Date.now() + 90 * 24 * 60 * 60 * 1000) / 1000)
              : Math.floor((Date.now() + 365 * 24 * 60 * 60 * 1000) / 1000);

      // Transform selected scopes to protocol format
      const permissions = Array.from(selectedScopes.entries()).map(([toolId, scopeData]) => {
        return {
          toolId: scopeData.toolId,
          toolType: scopeData.toolType || 1, // Use actual toolType
          name: scopeData.name,
          enabled: scopeData.enabled,
          toolFuncs: scopeData.toolFuncs || [],
          toolResources: scopeData.toolResources || [],
        };
      });

      // Use protocol 10008 to create token
      const response = await api.tokens.operateAccessToken({
        handleType: 1, // 1-add
        name: newTokenName || `Token ${tokens.length + 1}`,
        role: newTokenRole === 'Admin' ? 2 : 3, // 2-admin, 3-member
        expireAt: expiresAt,
        rateLimit: newTokenRateLimit,
        notes: newTokenPurpose,
        namespace: newTokenNamespace,
        tags: newTokenTags,
        permissions: permissions,
        masterPwd: masterPassword,
        proxyId: Number(proxyId),
      });

      if (response.data?.data?.accessToken) {
        // Show success dialog with token
        setCreatedTokenData({
          value: response.data.data.accessToken,
          name: newTokenName,
        });
        setDialogStep('success');

        // Refresh token list
        const fetchResponse = await api.tokens.getAccessTokens({
          proxyId: Number(proxyId),
        });

        if (fetchResponse.data?.data?.tokenList) {
          setTokens(transformTokenList(fetchResponse.data.data.tokenList));
        }
      }
    } catch (error: any) {
      // Check if it's a token limit error
      const { getApiErrorCode } = await import('@/lib/api-error');
      const errorCode = getApiErrorCode(error);

      if (errorCode === 'ERR_6005') {
        // Token limit exceeded - show limit dialog
        setShowLimitDialog(true);
        // Toast will still be shown by api-client interceptor, but dialog is more prominent
      }
      // Other errors are handled by api-client interceptor
    }
  };

  const handleEditToken = async (updatedToken: any) => {
    if (!proxyId) return;

    try {
      const { api } = await import('@/lib/api-client');

      // Show master password dialog
      setMasterPasswordAction({ type: 'edit', tokenData: updatedToken });
      setShowMasterPasswordDialog(true);
      return;
    } catch (error: any) {
      // Error handling is done by api-client interceptor
    }
  };

  const performEditToken = async (updatedToken: any, masterPassword: string) => {
    if (!proxyId) return;

    try {
      const { api } = await import('@/lib/api-client');

      // Use protocol 10008 to edit token
      await api.tokens.operateAccessToken({
        handleType: 2, // 2-edit
        userid: updatedToken.id,
        name: updatedToken.name,
        role: updatedToken.role === 'Admin' ? 2 : 3,
        expireAt:
          updatedToken.expiresAt === 'Never'
            ? 0
            : Math.floor(new Date(updatedToken.expiresAt).getTime() / 1000),
        rateLimit: updatedToken.rateLimit,
        notes: updatedToken.purpose,
        namespace: updatedToken.namespace,
        tags: updatedToken.tags,
        permissions: updatedToken.tools || [],
        masterPwd: masterPassword,
        proxyId: Number(proxyId),
      });

      // Refresh token list
      const fetchResponse = await api.tokens.getAccessTokens({
        proxyId: Number(proxyId),
      });

      if (fetchResponse.data?.data?.tokenList) {
        setTokens(transformTokenList(fetchResponse.data.data.tokenList));
      }
    } catch (error: any) {
      // Error handling and toast notification are done by api-client interceptor
    }
  };

  const handleDestroyToken = async (tokenId: string) => {
    if (!proxyId) return;

    try {
      const { api } = await import('@/lib/api-client');

      // Show master password dialog
      setMasterPasswordAction({ type: 'delete', tokenData: tokenId });
      setShowMasterPasswordDialog(true);
      return;
    } catch (error: any) {
      // Error handling is done by api-client interceptor
    }
  };

  const performDeleteToken = async (tokenId: string, masterPassword: string) => {
    if (!proxyId) return;

    try {
      const { api } = await import('@/lib/api-client');

      // Use protocol 10008 to delete token
      await api.tokens.operateAccessToken({
        handleType: 3, // 3-delete
        userid: tokenId,
        masterPwd: masterPassword,
        proxyId: Number(proxyId),
      });

      // Remove from local state
      setTokens(tokens.filter((token) => token.id !== tokenId));
      setSelectedTokenIds((prev) => {
        const next = new Set(prev);
        next.delete(tokenId);
        return next;
      });
    } catch (error: any) {
      throw error; // Re-throw to show error in dialog
    }
  };

  const performBulkUpdateTokens = async () => {
    if (!proxyId) return;

    const targetUserIds = Array.from(selectedTokenIds);
    if (targetUserIds.length === 0) return;

    const shouldUpdateNamespace = bulkNamespace.trim().length > 0;
    const shouldUpdateTags = bulkTagsMode === 'clear' || bulkTags.length > 0;
    const shouldUpdatePermissions = bulkPermissionsSourceTokenId !== '__none__';

    if (!shouldUpdateNamespace && !shouldUpdateTags && !shouldUpdatePermissions) {
      toast.error('Select at least one bulk update action');
      return;
    }

    if (shouldUpdateTags && bulkTagsMode !== 'clear' && bulkTags.length === 0) {
      toast.error('Tags are required unless tags mode is clear');
      return;
    }

    if (shouldUpdatePermissions) {
      const sourceToken = tokens.find((t) => t.id === bulkPermissionsSourceTokenId);
      if (!sourceToken) {
        toast.error('Permission source token not found');
        return;
      }
    }

    try {
      setIsBulkUpdating(true);
      const { api } = await import('@/lib/api-client');

      const payload: any = {
        handleType: 4,
        userids: targetUserIds,
        proxyId: Number(proxyId),
        masterPwd: '',
      };

      if (shouldUpdatePermissions) {
        const sourceToken = tokens.find((t) => t.id === bulkPermissionsSourceTokenId);
        payload.permissions = sourceToken?.tools || [];
        payload.permissionsMode = bulkPermissionsMode;
      }

      if (shouldUpdateNamespace) {
        payload.namespace = bulkNamespace.trim();
      }

      if (shouldUpdateTags) {
        payload.tagsMode = bulkTagsMode;
        if (bulkTagsMode !== 'clear') {
          payload.tags = bulkTags;
        }
      }

      const response = await api.tokens.operateAccessToken(payload);
      const result = response.data?.data;
      const updatedCount = result?.updatedCount ?? 0;
      const failedCount = result?.failedCount ?? 0;

      if (failedCount > 0) {
        toast.error(`Bulk update completed with ${failedCount} failures`);
      } else {
        toast.success(`Bulk update applied to ${updatedCount} token(s)`);
      }

      setShowBulkUpdateDialog(false);
      setSelectedTokenIds(new Set());
      setBulkPermissionsSourceTokenId('__none__');
      setBulkPermissionsMode('replace');
      setBulkNamespace('');
      setBulkTagsMode('add');
      setBulkTags([]);

      // Refresh token list
      const fetchResponse = await api.tokens.getAccessTokens({
        proxyId: Number(proxyId),
      });
      if (fetchResponse.data?.data?.tokenList) {
        setTokens(transformTokenList(fetchResponse.data.data.tokenList));
      }
    } catch (error: any) {
      // Error handling is done by api-client interceptor
    } finally {
      setIsBulkUpdating(false);
    }
  };

  // Handle master password confirmation
  const handleMasterPasswordConfirm = async (password: string) => {
    try {
      setIsProcessingWithPassword(true);

      // Execute the pending action
      if (masterPasswordAction?.type === 'create') {
        await performCreateToken(password);
      } else if (masterPasswordAction?.type === 'edit') {
        await performEditToken(masterPasswordAction.tokenData, password);
      } else if (masterPasswordAction?.type === 'delete') {
        await performDeleteToken(masterPasswordAction.tokenData, password);
      }

      // Reset dialog state
      setShowMasterPasswordDialog(false);
      setMasterPasswordAction(null);
    } catch (error) {
    } finally {
      setIsProcessingWithPassword(false);
    }
  };

  const getRoleIcon = (role: string) => {
    if (role === 'Owner') {
      return <Crown className="h-3 w-3" />;
    }
    return role === 'Admin' ? <Shield className="h-3 w-3" /> : <User className="h-3 w-3" />;
  };

  const getRoleBadgeColor = (role: string) => {
    if (role === 'Owner') {
      return 'bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 border-purple-200 dark:border-purple-800';
    }
    return role === 'Admin'
      ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 border-blue-200 dark:border-blue-800'
      : 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 border-green-200 dark:border-green-800';
  };

  const visibleTokens = useMemo(
    () => tokens.filter((token) => canViewToken(currentUserRole, token.roleNumber || token.role)),
    [tokens, currentUserRole],
  );

  const allNamespaces = useMemo(
    () => Array.from(new Set(tokens.map((t) => t.namespace || 'default'))).sort(),
    [tokens],
  );
  const allTags = useMemo(
    () => Array.from(new Set(tokens.flatMap((t) => t.tags || []))).sort(),
    [tokens],
  );

  const filteredTokens = useMemo(() => {
    return visibleTokens.filter((t) => {
      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        const matchesSearch =
          t.name?.toLowerCase().includes(q) ||
          t.displayToken?.toLowerCase().includes(q) ||
          t.namespace?.toLowerCase().includes(q) ||
          t.tags?.some((tag: string) => tag.toLowerCase().includes(q));
        if (!matchesSearch) return false;
      }
      if (filterNamespace && t.namespace !== filterNamespace) return false;
      if (filterRole && t.role !== filterRole) return false;
      if (filterTags.length > 0 && !filterTags.some((ft) => t.tags?.includes(ft))) return false;
      return true;
    });
  }, [visibleTokens, searchQuery, filterNamespace, filterTags, filterRole]);

  const groupedTokens = useMemo(() => {
    const groups = new Map<string, typeof filteredTokens>();
    for (const token of filteredTokens) {
      const ns = token.namespace || 'default';
      if (!groups.has(ns)) groups.set(ns, []);
      groups.get(ns)!.push(token);
    }
    return Array.from(groups.entries()).sort(([a], [b]) => a.localeCompare(b));
  }, [filteredTokens]);

  const hasActiveFilters = searchQuery || filterNamespace || filterTags.length > 0 || filterRole;

  const manageableFilteredTokens = filteredTokens.filter((token) =>
    canManageToken(currentUserRole, token.roleNumber || token.role),
  );

  const selectedVisibleCount = manageableFilteredTokens.filter((token) =>
    selectedTokenIds.has(token.id),
  ).length;

  const isAllVisibleManageableSelected =
    manageableFilteredTokens.length > 0 && selectedVisibleCount === manageableFilteredTokens.length;

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-4 mb-2">
          <p className="text-sm text-amber-800 dark:text-amber-300">
            <strong>Deprecated:</strong> This page has been replaced by{' '}
            <a href="/dashboard/tokens" className="underline font-medium">
              Tokens
            </a>
            . It will be removed in a future release.
          </p>
        </div>
        <div className="space-y-0">
          <h1 className="text-[30px] font-bold">Access Tokens</h1>
          <p className="text-base text-muted-foreground">Create and manage access tokens.</p>
        </div>
        <Card>
          <CardContent className="pt-6">
            <div className="space-y-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="p-4 border rounded-lg space-y-2.5">
                  <div className="flex items-center gap-3">
                    <Skeleton className="h-5 w-5 rounded" />
                    <Skeleton className="h-4 w-40" />
                    <Skeleton className="h-5 w-16 rounded-full" />
                  </div>
                  <div className="flex gap-2 pl-8">
                    <Skeleton className="h-3 w-28" />
                  </div>
                  <div className="flex gap-1.5 pl-8">
                    <Skeleton className="h-5 w-20 rounded-full" />
                    <Skeleton className="h-5 w-14 rounded-full" />
                  </div>
                  <div className="flex gap-3 pl-8">
                    <Skeleton className="h-3 w-24" />
                    <Skeleton className="h-3 w-20" />
                    <Skeleton className="h-3 w-16" />
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-4 mb-2">
        <p className="text-sm text-amber-800 dark:text-amber-300">
          <strong>Deprecated:</strong> This page has been replaced by{' '}
          <a href="/dashboard/tokens" className="underline font-medium">
            Tokens
          </a>
          . It will be removed in a future release.
        </p>
      </div>
      <div className="space-y-0">
        <h1 className="text-[30px] font-bold">Access Tokens</h1>
        <p className="text-base text-muted-foreground">Create and manage access tokens.</p>
      </div>

      {/* Desk Usage Notification for Owner */}
      {isOwner && !hasRegularTokens && (
        <Card className="border-amber-200 bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-950/20 dark:to-orange-950/20">
          <CardHeader>
            <div className="space-y-4">
              <div className="text-sm text-amber-800 dark:text-amber-200">
                <h4 className="font-semibold mb-2">Owner Token vs Access Tokens</h4>
                <p>
                  Your Owner Token has full admin privileges—keep it secure and never share it
                  directly. Instead, create Access Tokens with scoped permissions for users and AI
                  agents.
                </p>
              </div>

              <div className="flex items-center gap-3">
                <Button
                  onClick={() => setShowTokenDialog(true)}
                  className="bg-gradient-to-r from-amber-600 to-orange-600 hover:from-amber-700 hover:to-orange-700"
                >
                  <Key className="mr-2 h-4 w-4" />
                  Create First Token
                </Button>
                <a
                  href="https://www.kimbap.io/quick-start/#install-desk"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button
                    variant="outline"
                    className="border-amber-300 text-amber-700 hover:bg-amber-100"
                  >
                    <Download className="mr-2 h-4 w-4" />
                    Download Kimbap Desk
                  </Button>
                </a>
              </div>
            </div>
          </CardHeader>
        </Card>
      )}

      {/* Token Limit Upgrade Guidance */}
      {tokens.length >= 30 && hasRegularTokens && (
        <Card className="border-blue-200 bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-950/20 dark:to-indigo-950/20">
          <CardHeader>
            <CardTitle className="text-blue-900 dark:text-blue-100">
              Token Limit Information
            </CardTitle>
            <CardDescription className="text-blue-700 dark:text-blue-200">
              You're using {tokens.length} of your plan's token limit. Upgrade to create more access
              tokens.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-blue-800 dark:text-blue-200">
                  Different plans allow different numbers of access tokens. Upgrade to create more
                  tokens for your team.
                </p>
                <ul className="text-xs text-blue-700 dark:text-blue-300 mt-2 space-y-1">
                  <li>• Community Plan: Up to 30 tokens</li>
                  <li>• Business Plan: Up to 100 tokens</li>
                  <li>• Enterprise Plan: Unlimited tokens</li>
                </ul>
              </div>
              <a href={getPricingUrl()} target="_blank" rel="noopener noreferrer">
                <Button className="bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700">
                  <Crown className="mr-2 h-4 w-4" />
                  Get License
                </Button>
              </a>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Access Tokens Section */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <span className="text-sm text-muted-foreground">
            {visibleTokens.length} token{visibleTokens.length !== 1 ? 's' : ''}
          </span>
          {(currentUserRole === 'Owner' || currentUserRole === 'Admin') && (
            <Button
              onClick={() => {
                setShowTokenDialog(true);
                if (proxyId) {
                  loadScopes();
                }
              }}
            >
              Create Token
            </Button>
          )}
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {(currentUserRole === 'Owner' || currentUserRole === 'Admin') && (
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Checkbox
                    checked={isAllVisibleManageableSelected}
                    onCheckedChange={(checked) => {
                      const isChecked = checked === true;
                      setSelectedTokenIds((prev) => {
                        const next = new Set(prev);
                        manageableFilteredTokens.forEach((t) => {
                          if (isChecked) {
                            next.add(t.id);
                          } else {
                            next.delete(t.id);
                          }
                        });
                        return next;
                      });
                    }}
                    disabled={manageableFilteredTokens.length === 0}
                  />
                  <span className="text-sm text-muted-foreground">
                    Select all{' '}
                    <span className="text-muted-foreground/70">
                      · {selectedVisibleCount} of {manageableFilteredTokens.length} selected
                    </span>
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowBulkUpdateDialog(true)}
                    disabled={selectedTokenIds.size === 0}
                  >
                    Bulk Update
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setSelectedTokenIds(new Set())}
                    disabled={selectedTokenIds.size === 0}
                  >
                    Clear
                  </Button>
                </div>
              </div>
            )}

            {/* Filter Toolbar */}
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <div className="relative flex-1">
                  <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search tokens..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-9 h-9"
                  />
                </div>
                <Select
                  value={filterNamespace || '__all__'}
                  onValueChange={(v) => setFilterNamespace(v === '__all__' ? null : v)}
                >
                  <SelectTrigger className="w-[160px] h-9">
                    <SelectValue placeholder="All namespaces" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__all__">All namespaces</SelectItem>
                    {allNamespaces.map((ns) => (
                      <SelectItem key={ns} value={ns}>
                        {ns}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>

                <Popover open={tagsFilterOpen} onOpenChange={setTagsFilterOpen}>
                  <PopoverTrigger asChild>
                    <Button variant="outline" size="sm" className="h-9 gap-1.5">
                      Tags
                      {filterTags.length > 0 && (
                        <Badge variant="secondary" className="ml-1 h-5 px-1.5 text-xs">
                          {filterTags.length}
                        </Badge>
                      )}
                      <ChevronDown className="h-3.5 w-3.5 opacity-50" />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-[200px] p-0" align="start">
                    <Command>
                      <CommandInput placeholder="Filter tags..." />
                      <CommandList>
                        <CommandEmpty>No tags found</CommandEmpty>
                        <CommandGroup>
                          {allTags.map((tag) => (
                            <CommandItem
                              key={tag}
                              onSelect={() => {
                                setFilterTags((prev) =>
                                  prev.includes(tag)
                                    ? prev.filter((t) => t !== tag)
                                    : [...prev, tag],
                                );
                              }}
                            >
                              <div
                                className={cn(
                                  'mr-2 flex h-4 w-4 items-center justify-center rounded-sm border border-primary',
                                  filterTags.includes(tag)
                                    ? 'bg-primary text-primary-foreground'
                                    : 'opacity-50',
                                )}
                              >
                                {filterTags.includes(tag) && <CheckCircle className="h-3 w-3" />}
                              </div>
                              <span className="font-mono text-sm">{tag}</span>
                            </CommandItem>
                          ))}
                        </CommandGroup>
                      </CommandList>
                    </Command>
                  </PopoverContent>
                </Popover>

                <Select
                  value={filterRole || '__all__'}
                  onValueChange={(v) => setFilterRole(v === '__all__' ? null : v)}
                >
                  <SelectTrigger className="w-[130px] h-9">
                    <SelectValue placeholder="All roles" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__all__">All roles</SelectItem>
                    <SelectItem value="Owner">Owner</SelectItem>
                    <SelectItem value="Admin">Admin</SelectItem>
                    <SelectItem value="Member">Member</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {hasActiveFilters && (
                <div className="flex items-center gap-1.5 flex-wrap">
                  {filterNamespace && (
                    <Badge variant="secondary" className="gap-1 pr-1">
                      ns:{filterNamespace}
                      <button
                        type="button"
                        onClick={() => setFilterNamespace(null)}
                        aria-label={`Remove namespace filter ${filterNamespace}`}
                        className="ml-0.5 rounded-full p-0.5 hover:bg-muted-foreground/20 cursor-pointer"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  )}
                  {filterTags.map((tag) => (
                    <Badge key={tag} variant="secondary" className="gap-1 pr-1">
                      tag:{tag}
                      <button
                        type="button"
                        onClick={() => setFilterTags((prev) => prev.filter((t) => t !== tag))}
                        aria-label={`Remove tag filter ${tag}`}
                        className="ml-0.5 rounded-full p-0.5 hover:bg-muted-foreground/20 cursor-pointer"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  ))}
                  {filterRole && (
                    <Badge variant="secondary" className="gap-1 pr-1">
                      role:{filterRole}
                      <button
                        type="button"
                        onClick={() => setFilterRole(null)}
                        aria-label={`Remove role filter ${filterRole}`}
                        className="ml-0.5 rounded-full p-0.5 hover:bg-muted-foreground/20 cursor-pointer"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  )}
                  <button
                    type="button"
                    onClick={() => {
                      setSearchQuery('');
                      setFilterNamespace(null);
                      setFilterTags([]);
                      setFilterRole(null);
                    }}
                    className="text-xs text-muted-foreground hover:text-foreground ml-1 cursor-pointer"
                  >
                    Clear all
                  </button>
                  <span className="text-xs text-muted-foreground ml-auto">
                    Showing {filteredTokens.length} of {visibleTokens.length} tokens
                  </span>
                </div>
              )}
            </div>

            {visibleTokens.length === 0 ? (
              <div className="text-center py-8">
                <Key className="h-12 w-12 text-muted-foreground/30 mx-auto mb-4" />
                <h3 className="text-lg font-medium text-foreground mb-2">
                  {currentUserRole === 'Member' ? 'No tokens available' : 'No tokens found'}
                </h3>
                <p className="text-sm text-muted-foreground">
                  {currentUserRole === 'Member'
                    ? 'You can only view Member-level tokens that you have access to.'
                    : 'Create a token to get started.'}
                </p>
              </div>
            ) : filteredTokens.length === 0 && hasActiveFilters ? (
              <div className="text-center py-8">
                <Search className="h-10 w-10 text-muted-foreground/30 mx-auto mb-3" />
                <h3 className="text-sm font-medium text-foreground mb-1">
                  No tokens match your filters
                </h3>
                <button
                  type="button"
                  onClick={() => {
                    setSearchQuery('');
                    setFilterNamespace(null);
                    setFilterTags([]);
                    setFilterRole(null);
                  }}
                  className="text-sm text-primary hover:underline cursor-pointer"
                >
                  Clear filters
                </button>
              </div>
            ) : (
              groupedTokens.map(([namespace, nsTokens]) => (
                <div key={namespace}>
                  {groupedTokens.length > 1 && (
                    <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide py-2 mt-2 first:mt-0">
                      {namespace} ({nsTokens.length})
                    </div>
                  )}
                  {nsTokens.map((token) => {
                    const canManage = canManageToken(
                      currentUserRole,
                      token.roleNumber || token.role,
                    );
                    const showCheckbox = currentUserRole === 'Owner' || currentUserRole === 'Admin';

                    return (
                      <div
                        key={token.id}
                        className={cn(
                          'p-4 border rounded-lg hover:border-foreground/20 transition-colors mb-2 last:mb-0',
                          !canManage && 'bg-muted/30 opacity-75',
                        )}
                      >
                        <div className="flex items-start justify-between gap-3">
                          <div className="flex items-start gap-3 min-w-0 flex-1">
                            {showCheckbox && (
                              <Checkbox
                                className="mt-0.5"
                                checked={selectedTokenIds.has(token.id)}
                                onCheckedChange={(checked) => {
                                  const isChecked = checked === true;
                                  setSelectedTokenIds((prev) => {
                                    const next = new Set(prev);
                                    if (isChecked) {
                                      next.add(token.id);
                                    } else {
                                      next.delete(token.id);
                                    }
                                    return next;
                                  });
                                }}
                                disabled={!canManage}
                              />
                            )}
                            <Key className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                            <div className="min-w-0 flex-1 space-y-1">
                              <div className="flex items-center gap-2 flex-wrap">
                                <span className="font-semibold text-sm truncate">
                                  {token.name || 'Unnamed Token'}
                                </span>
                                <Badge
                                  variant="outline"
                                  className={cn('text-xs shrink-0', getRoleBadgeColor(token.role))}
                                >
                                  {getRoleIcon(token.role)}
                                  {token.role}
                                </Badge>
                                {!canManage && (
                                  <Badge
                                    variant="outline"
                                    className="text-xs bg-yellow-50 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-400 border-yellow-300 dark:border-yellow-700"
                                  >
                                    Read-only
                                  </Badge>
                                )}
                              </div>
                              <div className="font-mono text-xs text-muted-foreground">
                                {token.displayToken}
                              </div>
                              <div className="flex items-center gap-1.5 flex-wrap">
                                <Badge variant="secondary" className="text-xs">
                                  ns:{token.namespace || 'default'}
                                </Badge>
                                {Array.isArray(token.tags) &&
                                  token.tags.map((tag: string) => (
                                    <Badge key={tag} variant="outline" className="text-xs">
                                      {tag}
                                    </Badge>
                                  ))}
                              </div>
                              <div className="flex items-center gap-3 text-xs text-muted-foreground flex-wrap">
                                <span className="flex items-center gap-1">
                                  <Calendar className="h-3 w-3" />
                                  Created {token.createdAt}
                                </span>
                                <span>· Expires {token.expiresAt}</span>
                                <span>· {token.rateLimit}/min</span>
                                <span>
                                  · {token.tools.length} tool{token.tools.length !== 1 ? 's' : ''}
                                </span>
                                <span>
                                  · Last used:{' '}
                                  {token.lastUsed === 'Never used' ? 'never' : token.lastUsed}
                                </span>
                              </div>
                            </div>
                          </div>
                          <div className="flex items-center gap-1 shrink-0">
                            {canManage ? (
                              <>
                                <EditTokenDialog
                                  token={token}
                                  onSave={handleEditToken}
                                  allTags={allTags}
                                />
                                {token.roleNumber !== 1 && (
                                  <DestroyTokenDialog
                                    tokenName={token.name}
                                    tokenId={token.id}
                                    onDestroy={() => handleDestroyToken(token.id)}
                                  />
                                )}
                              </>
                            ) : (
                              <>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  disabled
                                  title="No permission to edit this token"
                                >
                                  <Edit className="h-4 w-4 text-muted-foreground/50" />
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  disabled
                                  title="No permission to delete this token"
                                >
                                  <Trash2 className="h-4 w-4 text-muted-foreground/50" />
                                </Button>
                              </>
                            )}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>

      {/* Create Token Dialog */}
      <Dialog open={showTokenDialog} onOpenChange={setShowTokenDialog}>
        <DialogContent
          className="sm:max-w-2xl max-h-[80vh] flex flex-col"
          hideCloseButton={dialogStep === 'success'}
        >
          <DialogHeader>
            <DialogTitle>
              {dialogStep === 'create' ? 'Create Access Token' : 'Access Token Created'}
            </DialogTitle>
            <DialogDescription>
              {dialogStep === 'create'
                ? 'Create a token and choose its access.'
                : 'Copy the token now. It is shown only once.'}
            </DialogDescription>
          </DialogHeader>

          {dialogStep === 'create' ? (
            <div className="space-y-6 overflow-y-auto flex-1 pr-2 px-1 pt-1">
              {/* Basic Information */}
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <Label htmlFor="token-name">
                      Label <span className="text-red-500">*</span>
                    </Label>
                    <Input
                      id="token-name"
                      placeholder="Production API"
                      value={newTokenName}
                      onChange={(e) => setNewTokenName(e.target.value)}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="token-role">
                      Role <span className="text-red-500">*</span>
                    </Label>
                    <Select
                      value={newTokenRole}
                      onValueChange={(value: 'Admin' | 'Member') => setNewTokenRole(value)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {/* Owner can create Admin and Member tokens */}
                        {currentUserRole === 'Owner' && (
                          <SelectItem value="Admin">
                            <div className="flex items-center gap-2">
                              <Shield className="h-4 w-4" />
                              Admin
                            </div>
                          </SelectItem>
                        )}
                        {/* Admin cannot create Admin tokens, only Member */}
                        {currentUserRole === 'Admin' && (
                          <SelectItem value="Admin" disabled>
                            <div className="flex items-center gap-2">
                              <Shield className="h-4 w-4 text-muted-foreground" />
                              Admin (Not allowed)
                            </div>
                          </SelectItem>
                        )}
                        <SelectItem value="Member">
                          <div className="flex items-center gap-2">
                            <User className="h-4 w-4" />
                            Member
                          </div>
                        </SelectItem>
                      </SelectContent>
                    </Select>
                    {currentUserRole === 'Admin' && (
                      <p className="text-xs text-yellow-600">
                        Admin can create Member tokens only.
                      </p>
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <Label htmlFor="token-expiry">Token Expiry</Label>
                    <Select
                      value={newTokenExpiry}
                      onValueChange={(value: 'never' | '30d' | '90d' | '1y') =>
                        setNewTokenExpiry(value)
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="never">Never expires</SelectItem>
                        <SelectItem value="30d">30 days</SelectItem>
                        <SelectItem value="90d">90 days</SelectItem>
                        <SelectItem value="1y">1 year</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="token-rate-limit">Rate Limit(Requests Per Minute)</Label>
                    <Input
                      id="token-rate-limit"
                      type="number"
                      min="1"
                      max="10000"
                      value={newTokenRateLimit}
                      onChange={(e) => setNewTokenRateLimit(Number(e.target.value))}
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="token-notes">Notes</Label>
                  <Textarea
                    id="token-notes"
                    placeholder="Used for CI/CD"
                    value={newTokenPurpose}
                    onChange={(e) => setNewTokenPurpose(e.target.value)}
                    rows={2}
                  />
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <Label htmlFor="token-namespace">Namespace</Label>
                    <Input
                      id="token-namespace"
                      placeholder="default"
                      value={newTokenNamespace}
                      onChange={(e) => setNewTokenNamespace(e.target.value)}
                    />
                    <p className="text-xs text-muted-foreground">
                      Use letters, numbers, and <span className="font-mono">- _ . / :</span>
                    </p>
                  </div>
                  <div className="space-y-2">
                    <Label>Tags</Label>
                    <TagInput
                      value={newTokenTags}
                      onChange={setNewTokenTags}
                      suggestions={allTags}
                      placeholder="Add tags..."
                    />
                  </div>
                </div>
              </div>

              {/* Tool Permissions */}
              <div className="space-y-4">
                <div>
                  <h3 className="text-lg font-medium">Scopes</h3>
                  <p className="text-sm text-muted-foreground">
                    Choose which resources and operations this token can access.
                  </p>
                </div>

                {isLoadingScopes ? (
                  <div className="flex items-center justify-center p-8">
                    <div className="text-center">
                      <div
                        className="w-6 h-6 border-2 border-muted-foreground border-t-foreground rounded-full animate-spin mx-auto mb-2"
                        aria-hidden="true"
                      />
                      <p className="text-sm text-muted-foreground">Loading scopes...</p>
                    </div>
                  </div>
                ) : availableScopes.length === 0 ? (
                  <div className="text-center py-8 border-2 border-dashed rounded-lg">
                    <Shield className="h-12 w-12 text-muted-foreground/30 mx-auto mb-2" />
                    <p className="text-sm text-muted-foreground">No scopes available</p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {availableScopes.map((scope, index) => {
                      const selectedScope = selectedScopes.get(scope.toolId);
                      const isScopeEnabled = selectedScope?.enabled ?? false;

                      return (
                        <Card key={`${scope.toolId}-${index}`} className="bg-muted/20">
                          <CardHeader className="pb-3">
                            <div className="flex items-center justify-between">
                              <div className="flex items-center gap-3">
                                <Shield className="h-5 w-5" />
                                <CardTitle className="text-base">
                                  {scope.name || scope.description || scope.toolId}
                                </CardTitle>
                              </div>
                              <div className="flex items-center gap-2">
                                <Switch
                                  checked={isScopeEnabled}
                                  onCheckedChange={(checked) => {
                                    const newSelectedScopes = new Map(selectedScopes);
                                    if (checked) {
                                      // When enabling a scope, copy the scope with all functions enabled by default
                                      const scopeCopy = {
                                        ...scope,
                                        enabled: true,
                                        toolFuncs:
                                          scope.toolFuncs?.map((func: any) => ({
                                            ...func,
                                            enabled: func.enabled === true, // Use the actual enabled state from API
                                          })) || [],
                                        toolResources:
                                          scope.toolResources?.map((res: any) => ({
                                            ...res,
                                            enabled: res.enabled === true, // Use the actual enabled state from API
                                          })) || [],
                                      };
                                      newSelectedScopes.set(scope.toolId, scopeCopy);
                                    } else {
                                      newSelectedScopes.delete(scope.toolId);
                                    }
                                    setSelectedScopes(newSelectedScopes);
                                  }}
                                />
                                {(scope.toolFuncs?.length > 0 ||
                                  scope.toolResources?.length > 0) && (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="h-6 w-6 p-0"
                                    onClick={() => toggleToolExpansion(scope.toolId)}
                                  >
                                    <ChevronDown
                                      className={cn(
                                        'h-4 w-4 transition-transform',
                                        expandedTools.has(scope.toolId) && 'rotate-180',
                                      )}
                                    />
                                  </Button>
                                )}
                              </div>
                            </div>
                          </CardHeader>

                          <Collapsible open={expandedTools.has(scope.toolId)}>
                            <CollapsibleContent>
                              <CardContent className="pt-0">
                                <div className="space-y-4">
                                  {/* Functions */}
                                  {scope.toolFuncs && scope.toolFuncs.length > 0 && (
                                    <div>
                                      <p className="text-sm font-medium text-muted-foreground mb-2">
                                        Functions:
                                      </p>
                                      <div className="space-y-1">
                                        {scope.toolFuncs.map((func: any) => {
                                          const isFuncEnabled =
                                            selectedScope?.toolFuncs?.find(
                                              (item: any) => item.funcName === func.funcName,
                                            )?.enabled ?? func.enabled;

                                          return (
                                            <div
                                              key={`${scope.toolId}-${func.funcName}`}
                                              className="flex items-center justify-between py-2 px-2 rounded hover:bg-muted/50"
                                            >
                                              <div className="flex flex-col">
                                                <span className="text-sm font-mono">
                                                  {func.funcName}
                                                </span>
                                                {func.description && (
                                                  <span className="text-xs text-muted-foreground">
                                                    {func.description}
                                                  </span>
                                                )}
                                              </div>
                                              <div className="flex items-center gap-2">
                                                <Switch
                                                  size="sm"
                                                  checked={isFuncEnabled && isScopeEnabled}
                                                  disabled={!isScopeEnabled}
                                                  onCheckedChange={(checked) => {
                                                    const newSelectedScopes = new Map(
                                                      selectedScopes,
                                                    );
                                                    const currentScope = newSelectedScopes.get(
                                                      scope.toolId,
                                                    );
                                                    if (currentScope) {
                                                      const updatedFuncs = (
                                                        currentScope.toolFuncs || []
                                                      ).map((item: any) =>
                                                        item.funcName === func.funcName
                                                          ? {
                                                              ...item,
                                                              enabled: checked,
                                                            }
                                                          : item,
                                                      );
                                                      newSelectedScopes.set(scope.toolId, {
                                                        ...currentScope,
                                                        toolFuncs: updatedFuncs,
                                                      });
                                                      setSelectedScopes(newSelectedScopes);
                                                    }
                                                  }}
                                                />
                                                <Badge variant="outline" className="text-xs">
                                                  {isFuncEnabled && isScopeEnabled
                                                    ? 'Enabled'
                                                    : 'Disabled'}
                                                </Badge>
                                              </div>
                                            </div>
                                          );
                                        })}
                                      </div>
                                    </div>
                                  )}

                                  {/* Resources */}
                                  {scope.toolResources && scope.toolResources.length > 0 && (
                                    <div>
                                      <p className="text-sm font-medium text-muted-foreground mb-2">
                                        Resources:
                                      </p>
                                      <div className="space-y-1">
                                        {scope.toolResources.map((resource: any) => {
                                          const resourceKey = resource.uri || resource.name;
                                          const isResourceEnabled =
                                            selectedScope?.toolResources?.find(
                                              (item: any) =>
                                                (item.uri || item.name) === resourceKey,
                                            )?.enabled ?? resource.enabled;

                                          return (
                                            <div
                                              key={`${scope.toolId}-${resource.uri || resource.name}`}
                                              className="flex items-center justify-between py-2 px-2 rounded hover:bg-muted/50"
                                            >
                                              <span className="text-sm font-mono">
                                                {resource.uri || resource.name}
                                              </span>
                                              <div className="flex items-center gap-2">
                                                <Switch
                                                  size="sm"
                                                  checked={isResourceEnabled && isScopeEnabled}
                                                  disabled={!isScopeEnabled}
                                                  onCheckedChange={(checked) => {
                                                    const newSelectedScopes = new Map(
                                                      selectedScopes,
                                                    );
                                                    const currentScope = newSelectedScopes.get(
                                                      scope.toolId,
                                                    );
                                                    if (currentScope) {
                                                      const updatedResources = (
                                                        currentScope.toolResources || []
                                                      ).map((item: any) =>
                                                        (item.uri || item.name) === resourceKey
                                                          ? {
                                                              ...item,
                                                              enabled: checked,
                                                            }
                                                          : item,
                                                      );
                                                      newSelectedScopes.set(scope.toolId, {
                                                        ...currentScope,
                                                        toolResources: updatedResources,
                                                      });
                                                      setSelectedScopes(newSelectedScopes);
                                                    }
                                                  }}
                                                />
                                                <Badge variant="outline" className="text-xs">
                                                  {isResourceEnabled && isScopeEnabled
                                                    ? 'Allowed'
                                                    : 'Denied'}
                                                </Badge>
                                              </div>
                                            </div>
                                          );
                                        })}
                                      </div>
                                    </div>
                                  )}

                                  {/* No permissions message */}
                                  {(!scope.toolFuncs || scope.toolFuncs.length === 0) &&
                                    (!scope.toolResources || scope.toolResources.length === 0) && (
                                      <p className="text-sm text-muted-foreground text-center py-2">
                                        No permissions configured for this tool.
                                      </p>
                                    )}
                                </div>
                              </CardContent>
                            </CollapsibleContent>
                          </Collapsible>
                        </Card>
                      );
                    })}
                  </div>
                )}
              </div>
            </div>
          ) : (
            // Success view with token display
            <div className="space-y-4 overflow-y-auto flex-1 pr-2">
              <div className="text-center space-y-4">
                <div className="w-12 h-12 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto">
                  <CheckCircle className="h-6 w-6 text-green-600 dark:text-green-400" />
                </div>
                <div>
                  <h3 className="text-lg font-medium text-foreground">
                    Token "{createdTokenData?.name}" Created
                  </h3>
                  <p className="text-sm text-muted-foreground mt-1">
                    Copy this token now. You will not see it again.
                  </p>
                </div>
              </div>

              <Alert className="border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-950/20">
                <Key className="h-4 w-4" />
                <AlertDescription>
                  <div className="space-y-3">
                    <p className="font-medium">Access token:</p>
                    <div className="p-3 bg-background border rounded">
                      <div className="font-mono text-sm break-all mb-3">
                        {createdTokenData?.value}
                      </div>
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          className="flex-1"
                          onClick={handleCopyToken}
                        >
                          <Copy className="h-4 w-4 mr-2" />
                          Copy Token
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          className="flex-1"
                          onClick={handleDownloadToken}
                        >
                          <FileDown className="h-4 w-4 mr-2" />
                          Download
                        </Button>
                      </div>
                    </div>
                  </div>
                </AlertDescription>
              </Alert>

              <Alert variant="destructive">
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  <div className="space-y-2">
                    <p className="font-medium">Security Notice</p>
                    <p className="text-sm">
                      This is the only time this token is shown. Store it securely.
                    </p>
                    <ul className="text-xs mt-2 space-y-1">
                      <li>• Store this key in environment variables</li>
                      <li>• Do not hard-code it in your source code</li>
                      <li>• If lost, create a new token</li>
                    </ul>
                  </div>
                </AlertDescription>
              </Alert>

              {!tokenActionTaken && (
                <Alert className="border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-950/20">
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription>
                    <p className="text-sm font-medium">
                      Copy or download the token before closing.
                    </p>
                  </AlertDescription>
                </Alert>
              )}
            </div>
          )}

          <DialogFooter className="border-t pt-4 mt-4 flex-shrink-0">
            {dialogStep === 'create' ? (
              <>
                <Button
                  variant="outline"
                  onClick={() => {
                    resetForm();
                    setShowTokenDialog(false);
                  }}
                >
                  Cancel
                </Button>
                <Button onClick={handleCreateToken} disabled={!newTokenName.trim()}>
                  Create Token
                </Button>
              </>
            ) : (
              <Button
                onClick={() => {
                  resetForm();
                  setShowTokenDialog(false);
                }}
                className="w-full"
                disabled={!tokenActionTaken}
                variant={tokenActionTaken ? 'default' : 'outline'}
              >
                I Stored the Token
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Bulk Update Dialog */}
      <Dialog open={showBulkUpdateDialog} onOpenChange={setShowBulkUpdateDialog}>
        <ScrollableDialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Bulk Update Tokens</DialogTitle>
            <DialogDescription>
              Apply updates to {selectedTokenIds.size} token
              {selectedTokenIds.size !== 1 ? 's' : ''}.
            </DialogDescription>
          </DialogHeader>

          <div>
            <div className="space-y-6">
              <div className="space-y-4">
                <div>
                  <h3 className="text-sm font-semibold">Permissions</h3>
                  <p className="text-xs text-muted-foreground">
                    Copy permissions from one token to selected tokens.
                  </p>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <Label>Copy from token</Label>
                    <Select
                      value={bulkPermissionsSourceTokenId}
                      onValueChange={(value) => setBulkPermissionsSourceTokenId(value)}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="None" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="__none__">None</SelectItem>
                        {visibleTokens.map((t) => (
                          <SelectItem key={t.id} value={t.id}>
                            {t.name || t.displayToken}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label>Mode</Label>
                    <Select
                      value={bulkPermissionsMode}
                      onValueChange={(value: 'replace' | 'merge') => setBulkPermissionsMode(value)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="replace">Replace</SelectItem>
                        <SelectItem value="merge">Merge</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <h3 className="text-sm font-semibold">Metadata</h3>
                  <p className="text-xs text-muted-foreground">
                    Leave blank to keep current values.
                  </p>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <Label htmlFor="bulk-namespace">Namespace</Label>
                    <Input
                      id="bulk-namespace"
                      placeholder="(no change)"
                      value={bulkNamespace}
                      onChange={(e) => setBulkNamespace(e.target.value)}
                    />
                  </div>

                  <div className="space-y-2">
                    <Label>Tags Mode</Label>
                    <Select
                      value={bulkTagsMode}
                      onValueChange={(value: 'replace' | 'add' | 'remove' | 'clear') =>
                        setBulkTagsMode(value)
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="add">Add</SelectItem>
                        <SelectItem value="remove">Remove</SelectItem>
                        <SelectItem value="replace">Replace</SelectItem>
                        <SelectItem value="clear">Clear</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="space-y-2">
                  <Label>Tags</Label>
                  <TagInput
                    value={bulkTags}
                    onChange={setBulkTags}
                    suggestions={allTags}
                    placeholder="Add tags..."
                    disabled={bulkTagsMode === 'clear'}
                  />
                </div>
              </div>
            </div>
          </div>
          <DialogFooter className="border-t pt-4">
            <Button
              variant="outline"
              onClick={() => setShowBulkUpdateDialog(false)}
              disabled={isBulkUpdating}
            >
              Cancel
            </Button>
            <Button
              onClick={performBulkUpdateTokens}
              disabled={isBulkUpdating || selectedTokenIds.size === 0}
            >
              Apply Updates
            </Button>
          </DialogFooter>
        </ScrollableDialogContent>
      </Dialog>

      {/* Master Password Dialog */}
      <MasterPasswordDialog
        open={showMasterPasswordDialog}
        onOpenChange={(open) => {
          setShowMasterPasswordDialog(open);
          if (!open) {
            setMasterPasswordAction(null);
          }
        }}
        onConfirm={handleMasterPasswordConfirm}
        title={
          masterPasswordAction?.type === 'create'
            ? 'Create Token'
            : masterPasswordAction?.type === 'edit'
              ? 'Edit Token'
              : 'Delete Token'
        }
        description={
          masterPasswordAction?.type === 'create'
            ? 'Enter the master password to create this token.'
            : masterPasswordAction?.type === 'edit'
              ? 'Enter the master password to edit this token.'
              : 'Enter the master password to delete this token.'
        }
        isLoading={isProcessingWithPassword}
        showForgotPassword={isOwner}
      />

      {/* Token Limit Reached Dialog */}
      <Dialog open={showLimitDialog} onOpenChange={setShowLimitDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-amber-600" />
              Usage Limit Reached
            </DialogTitle>
            <DialogDescription>
              You reached your plan token limit. Upgrade to create more tokens.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowLimitDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => {
                setShowLimitDialog(false);
                router.push('/dashboard/billing');
              }}
            >
              Upgrade
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
