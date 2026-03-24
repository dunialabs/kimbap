'use client';

import {
  Plus,
  CirclePlus,
  Trash2,
  AlertTriangle,
  CheckCircle2,
  FileText,
  Settings,
  Zap,
  Shield,
  Loader2,
  Moon,
  Search,
  RefreshCw,
  X,
  Upload,
  FolderArchive,
  Info,
  Eye,
  EyeOff,
} from 'lucide-react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useState, useEffect, useRef, useCallback } from 'react';
import { toast } from 'sonner';

import { RestApiValidationPanel } from '@/components/rest-api/rest-api-validation-panel';
import { AllowUserInputConfiguration } from '@/components/tool-configure/AllowUserInputConfiguration';
import { LazyStartConfiguration } from '@/components/tool-configure/LazyStartConfiguration';
import { PublicAccessConfiguration } from '@/components/tool-configure/PublicAccessConfiguration';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Textarea } from '@/components/ui/textarea';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';
import { MasterPasswordManager } from '@/lib/crypto';
import { SKILLS_MAX_FILE_SIZE } from '@/lib/mcp-constants';
import { useMasterPassword } from '@/contexts/master-password-context';
import { OAuthConfig, ServerAuthType } from '@/types/api';

// Generate consistent ID for temporary use
const generateConsistentId = (seed: string) => {
  let hash = 0;
  for (let i = 0; i < seed.length; i++) {
    const char = seed.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash = hash & hash; // Convert to 32-bit integer
  }
  return Math.abs(hash).toString(36).substring(0, 9);
};

const TEMPLATE_CACHE_KEY = 'kimbap-tool-templates-cache';
const CACHE_EXPIRY_KEY = 'kimbap-tool-templates-expiry';
const CACHE_DURATION = 5 * 60 * 1000;
const SENSITIVE_ENV_KEY_PATTERN =
  /(?:SECRET|TOKEN|PASSWORD|PASSPHRASE|API_KEY|ACCESS_KEY|SECRET_KEY|PRIVATE_KEY|CLIENT_SECRET|REFRESH_TOKEN|SESSION_SECRET|CREDENTIALS?)/i;
const isSensitiveEnvKey = (key: string) => SENSITIVE_ENV_KEY_PATTERN.test(key);
type EditableKeyValueRow = { id: string; key: string; value: string };

let editableKeyValueRowSequence = 0;

const createEditableKeyValueRow = (key = '', value = ''): EditableKeyValueRow => ({
  id: `editable-row-${editableKeyValueRowSequence++}`,
  key,
  value,
});

const getTemplatesFromCache = (): Tool[] | null => {
  try {
    const cached = localStorage.getItem(TEMPLATE_CACHE_KEY);
    const expiry = localStorage.getItem(CACHE_EXPIRY_KEY);

    if (cached && expiry && Date.now() < parseInt(expiry)) {
      return JSON.parse(cached);
    }
  } catch (error) {
    // Failed to read templates from cache:
  }
  return null;
};

const saveTemplatesToCache = (templates: Tool[]) => {
  try {
    localStorage.setItem(TEMPLATE_CACHE_KEY, JSON.stringify(templates));
    localStorage.setItem(CACHE_EXPIRY_KEY, (Date.now() + CACHE_DURATION).toString());
  } catch (error) {
    // Failed to save templates to cache:
  }
};

const getServerTypeFromToolType = (toolType: number): string => {
  const typeMap: Record<number, string> = {
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
  return typeMap[toolType] || 'github';
};

const getStatusFromRunningState = (runningState: number): string => {
  switch (runningState) {
    case 0:
      return 'connected';
    case 1:
      return 'connect_failed';
    case 2:
      return 'connecting';
    case 3:
      return 'error';
    case 4:
      return 'sleeping';
    default:
      return 'connect_failed';
  }
};

interface Tool {
  toolId: string;
  toolTmplId: string;
  toolType: number;
  name: string;
  description: string;
  tags: string[];
  authtags: string[];
  // Present on template tool objects returned by getTemplates()
  oAuthConfig?: OAuthConfig;
  credentials: Array<{
    name: string;
    description: string;
    dataType: number;
    key: string;
    value?: string;
  }>;
  toolFuncs?: Array<{
    funcName: string;
    enabled: boolean;
  }>;
  toolResources?: Array<{
    uri: string;
    enabled: boolean;
  }>;
  enabled: boolean;
  runningState: number;
  lastUsed?: number;
}

interface MCPServerConfig {
  id: string;
  type: string;
  name: string;
  credentials: any;
  enabledFunctions: string[];
  totalFunctions: number;
  dataPermissions: {
    type: string;
    allowedResources: string[];
  };
  totalResources: number;
  status: string;
  lastValidated?: string;
  enabled: boolean;
  allowUserInput?: number;
  category?: number; // 1-Template tool, 2-Custom MCP tool, 3-REST API tool
}

export default function ToolConfigurePage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [servers, setServers] = useState<MCPServerConfig[]>([]);
  const [toolTemplates, setToolTemplates] = useState<any[]>([]);
  const [isLoadingTemplates, setIsLoadingTemplates] = useState(false);
  const [isLoadingTools, setIsLoadingTools] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [serverToDelete, setServerToDelete] = useState<MCPServerConfig | null>(null);
  const [operatingServers, setOperatingServers] = useState<Set<string>>(new Set());

  // Use global master password dialog
  const { requestMasterPassword } = useMasterPassword();
  const skillsFileInputRef = useRef<HTMLInputElement>(null);
  const hasLoadedInitialDataRef = useRef(false);

  // Custom tool form states
  const [customMcpUrl, setCustomMcpUrl] = useState('');
  const [customHeaders, setCustomHeaders] = useState<EditableKeyValueRow[]>(() => [
    createEditableKeyValueRow(),
  ]);
  const [customLazyStartEnabled, setCustomLazyStartEnabled] = useState(true); // Default to true
  const [customTransportType, setCustomTransportType] = useState<'url' | 'stdio'>('url');
  const [customStdioCommand, setCustomStdioCommand] = useState('');
  const [customStdioCwd, setCustomStdioCwd] = useState('');
  const [customStdioArgs, setCustomStdioArgs] = useState('');
  const [customStdioEnvVars, setCustomStdioEnvVars] = useState<EditableKeyValueRow[]>(() => [
    createEditableKeyValueRow(),
  ]);
  const [customStdioHiddenEnvRowIds, setCustomStdioHiddenEnvRowIds] = useState<Set<string>>(
    new Set(),
  );

  // REST API tool form states
  const [restApiConfig, setRestApiConfig] = useState('');
  const [restApiConfigError, setRestApiConfigError] = useState<string | null>(null);
  const hasRestApiInput = restApiConfig.trim().length > 0;
  const [restApiLazyStartEnabled, setRestApiLazyStartEnabled] = useState(true); // Default to true

  // PublicAccess states for each tool type
  const [customPublicAccess, setCustomPublicAccess] = useState(false); // Default to false (private)
  const [restApiPublicAccess, setRestApiPublicAccess] = useState(false); // Default to false (private)

  // AllowUserInput state for REST API tool
  const [restApiAllowUserInput, setRestApiAllowUserInput] = useState<number>(0); // Default to 0 (admin configured)

  // AllowUserInput state for Custom MCP tool
  const [customAllowUserInput, setCustomAllowUserInput] = useState<number>(0); // Default to 0 (admin configured)

  // Skills tool form states
  const [skillsServerName, setSkillsServerName] = useState('Skills MCP Server');
  const [skillsZipFile, setSkillsZipFile] = useState<File | null>(null);
  const [skillsDragActive, setSkillsDragActive] = useState(false);
  const [isAddingSkillsServer, setIsAddingSkillsServer] = useState(false);
  const [skillsLazyStartEnabled, setSkillsLazyStartEnabled] = useState(true); // Default to true
  const [skillsPublicAccess, setSkillsPublicAccess] = useState(false); // Default to false (private)

  // Check if we should open the add dialog from URL parameter
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    const action = searchParams.get('action');
    if (action === 'add') {
      setDialogOpen(true);
      // Remove the action parameter from URL
      router.replace('/dashboard/tool-configure');
    }
  }, [searchParams, router]);

  // Handle OAuth callback entry - redirect to tool configuration page
  useEffect(() => {
    const handleOAuthCallback = async () => {
      const code = searchParams.get('code');
      const state = searchParams.get('state');
      const error = searchParams.get('error');

      if (error) {
        toast.error(`Authorization failed: ${error}`);
        router.replace('/dashboard/tool-configure');
        return;
      }

      if (!code || !state) {
        return;
      }

      try {
        // Import OAuth session utilities
        const { getOAuthSession } = await import('@/lib/oauth-authorization');
        const sessionData = getOAuthSession(state);

        if (sessionData) {
          // Store code and state for configuration page to use
          sessionStorage.setItem('oauth_callback', JSON.stringify({ code, state }));

          // Redirect to tool configuration page
          router.push(
            `/dashboard/tool-configure/${sessionData.tempId}?mode=setup&type=${sessionData.serverType}&templateId=${sessionData.templateId}&oauthCallback=true`,
          );
        } else {
          // No session data found, clear URL params
          router.replace('/dashboard/tool-configure');
        }
      } catch (error: any) {
        // Failed to handle OAuth callback:
        toast.error(error.message || 'Could not handle authorization callback');
        router.replace('/dashboard/tool-configure');
      }
    };

    handleOAuthCallback();
  }, [searchParams, router]);

  const loadToolData = useCallback(async () => {
    // Prevent duplicate requests
    if (isLoadingTools) return;

    try {
      setIsLoadingTools(true);
      const { api } = await import('@/lib/api-client');

      // Get server info first to get proxyId
      const serverInfoResponse = await api.servers.getInfo();
      const proxyId = serverInfoResponse.data?.data?.proxyId;

      if (!proxyId) {
        return;
      }

      // Get all tools using protocol 10006
      const response = await api.tools.getToolList({
        proxyId: Number(proxyId),
        handleType: 1, // 1-all
      });

      if (response.data?.data?.toolList.length) {
        const toolList = response.data.data.toolList.map((tool: any) => ({
          id: tool.toolId || generateConsistentId(`${tool.toolType}-${tool.name}`),
          type: getServerTypeFromToolType(tool.toolType),
          name: tool.name,
          credentials: tool.credentials || {},
          enabledFunctions:
            tool.toolFuncs?.filter((f: any) => f.enabled).map((f: any) => f.funcName) || [],
          totalFunctions: tool.toolFuncs?.length || 0,
          dataPermissions: {
            type: 'default',
            allowedResources:
              tool.toolResources?.filter((r: any) => r.enabled).map((r: any) => r.uri) || [],
          },
          totalResources: tool.toolResources?.length || 0,
          status: getStatusFromRunningState(tool.runningState),
          lastValidated: tool.lastUsed ? new Date(tool.lastUsed * 1000).toISOString() : undefined,
          enabled: tool.enabled,
          allowUserInput: tool.allowUserInput,
          category: tool.category, // 1-Template, 2-Custom MCP, 3-REST API
        }));
        setServers(toolList);
      }
    } catch (error: any) {
      // Failed to load tools:
      toast.error('Could not load tools');
    } finally {
      setIsLoadingTools(false);
    }
  }, [isLoadingTools]);

  const loadToolTemplates = useCallback(async () => {
    // Prevent duplicate requests
    if (isLoadingTemplates) return;

    // First try to get from cache
    const cachedTemplates = getTemplatesFromCache();
    if (cachedTemplates && cachedTemplates.length > 0) {
      setToolTemplates(cachedTemplates);
      return;
    }

    try {
      setIsLoadingTemplates(true);
      const { api } = await import('@/lib/api-client');

      const response = await api.tools.getTemplates();

      // Check different data paths
      const templates = response.data?.data?.toolTmplList || response.data?.toolTmplList || [];
      if (templates.length > 0) {
        setToolTemplates(templates);
        saveTemplatesToCache(templates);
      }
    } catch (error: any) {
      // Failed to load tool templates:
      toast.error('Could not load tool templates');
    } finally {
      setIsLoadingTemplates(false);
    }
  }, [isLoadingTemplates]);

  useEffect(() => {
    if (hasLoadedInitialDataRef.current) {
      return;
    }
    if (isLoadingTools || isLoadingTemplates) {
      return;
    }

    hasLoadedInitialDataRef.current = true;
    void Promise.all([loadToolData(), loadToolTemplates()]);
  }, [isLoadingTools, isLoadingTemplates, loadToolData, loadToolTemplates]);

  const handleToggleServer = async (serverId: string, enabled: boolean) => {
    if (operatingServers.has(serverId)) return;

    const { api } = await import('@/lib/api-client');

    // Get server info first to get proxyId
    const serverInfoResponse = await api.servers.getInfo();
    const proxyId = serverInfoResponse.data?.data?.proxyId;

    if (!proxyId) {
      toast.error('No server found. Unable to toggle tool status.');
      return;
    }

    // Always show master password dialog for toggle
    requestMasterPassword({
      title: 'Toggle Tool - Master Password Required',
      description: 'Please enter your master password to change the tool status.',
      onConfirm: async (password) => {
        await performToggleServer(serverId, enabled, password, proxyId);
      },
    });
  };

  const performToggleServer = async (
    serverId: string,
    enabled: boolean,
    masterPwd: string,
    proxyId: number,
  ) => {
    if (operatingServers.has(serverId)) return;

    try {
      setOperatingServers((prev) => new Set(prev).add(serverId));
      const { api } = await import('@/lib/api-client');

      // Call protocol 10005 to enable/disable tool
      const handleType = enabled ? 3 : 4; // 3-enable, 4-disable
      await api.tools.operateTool({
        handleType,
        proxyId: Number(proxyId),
        toolId: serverId,
        masterPwd,
      });

      // Update local state
      setServers((prev) =>
        prev.map((server) => (server.id === serverId ? { ...server, enabled: enabled } : server)),
      );

      toast.success(`Tool ${enabled ? 'enabled' : 'disabled'}`);
    } catch (error: any) {
      // Failed to toggle tool:
      toast.error(`Could not ${enabled ? 'enable' : 'disable'} tool`);
      throw error; // Re-throw to keep dialog open for retry
    } finally {
      setOperatingServers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(serverId);
        return newSet;
      });
      loadToolData();
    }
  };

  const handleDeleteTool = async (server: MCPServerConfig) => {
    if (operatingServers.has(server.id)) return;

    try {
      setOperatingServers((prev) => new Set(prev).add(server.id));
      const { api } = await import('@/lib/api-client');

      // Get server info first to get proxyId
      const serverInfoResponse = await api.servers.getInfo();
      const proxyId = serverInfoResponse.data?.data?.proxyId;

      if (!proxyId) {
        toast.error('No server found. Unable to delete tool.');
        return;
      }

      // Check if master password is cached
      const cachedPassword = MasterPasswordManager.getCachedMasterPassword();
      if (cachedPassword) {
        // Use cached password directly
        await performDeleteTool(server, cachedPassword);
        return;
      }

      // Show master password dialog if not cached
      requestMasterPassword({
        title: 'Delete Tool - Master Password Required',
        description: 'Please enter your master password to delete the tool.',
        onConfirm: async (password) => {
          await performDeleteTool(server, password);
        },
      });
      return;
    } finally {
      setOperatingServers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(server.id);
        return newSet;
      });
      setDeleteDialogOpen(false);
      setServerToDelete(null);
      loadToolData();
    }
  };

  const performDeleteTool = async (server: MCPServerConfig, masterPwd: string) => {
    if (operatingServers.has(server.id)) return;

    try {
      setOperatingServers((prev) => new Set(prev).add(server.id));
      const { api } = await import('@/lib/api-client');

      // Get server info first to get proxyId
      const serverInfoResponse = await api.servers.getInfo();
      const proxyId = serverInfoResponse.data?.data?.proxyId;

      if (!proxyId) {
        toast.error('No server found. Unable to delete tool.');
        return;
      }

      // Call protocol 10005 to delete tool
      await api.tools.operateTool({
        handleType: 5, // 5-delete
        proxyId: Number(proxyId),
        toolId: server.id,
        masterPwd,
      });

      // Remove from local state
      setServers((prev) => prev.filter((s) => s.id !== server.id));
      toast.success('Tool deleted');
    } catch (error: any) {
      // Failed to delete tool:
      toast.error('Could not delete tool');
    } finally {
      setOperatingServers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(server.id);
        return newSet;
      });
      setDeleteDialogOpen(false);
      setServerToDelete(null);
    }
  };

  const handleRestartTool = async (serverId: string) => {
    if (operatingServers.has(serverId)) return;

    try {
      setOperatingServers((prev) => new Set(prev).add(serverId));
      const { api } = await import('@/lib/api-client');

      // Get server info first to get proxyId
      const serverInfoResponse = await api.servers.getInfo();
      const proxyId = serverInfoResponse.data?.data?.proxyId;

      if (!proxyId) {
        toast.error('No server found. Unable to restart tool.');
        return;
      }

      // Check if master password is cached
      const cachedPassword = MasterPasswordManager.getCachedMasterPassword();
      if (cachedPassword) {
        // Use cached password directly
        await performRestartTool(serverId, cachedPassword);
        return;
      }

      // Show master password dialog if not cached
      requestMasterPassword({
        title: 'Restart Tool - Master Password Required',
        description: 'Please enter your master password to restart the tool.',
        onConfirm: async (password) => {
          await performRestartTool(serverId, password);
        },
      });
      return;
    } finally {
      setOperatingServers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(serverId);
        return newSet;
      });
    }
  };

  const performRestartTool = async (serverId: string, masterPwd: string) => {
    if (operatingServers.has(serverId)) return;

    try {
      setOperatingServers((prev) => new Set(prev).add(serverId));
      const { api } = await import('@/lib/api-client');

      // Get server info first to get proxyId
      const serverInfoResponse = await api.servers.getInfo();
      const proxyId = serverInfoResponse.data?.data?.proxyId;

      if (!proxyId) {
        toast.error('No server found. Unable to restart tool.');
        return;
      }

      // Call protocol 10005 with handleType 6 to restart tool
      const response = await api.tools.operateTool({
        handleType: 6, // 6-restart/start server
        proxyId: Number(proxyId),
        toolId: serverId,
        masterPwd,
      });

      // Check response
      if (response.data?.data?.isStartServer === 1) {
        toast.success('Tool restarted');
      } else {
        toast.error('Could not restart tool');
      }
    } catch (error: any) {
      // Failed to restart tool:
      const errorMessage = (error as any).userMessage || 'Could not restart tool';
      toast.error(errorMessage);
    } finally {
      setOperatingServers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(serverId);
        return newSet;
      });
      // Refresh tool list after restart
      loadToolData();
    }
  };

  const openDeleteDialog = (server: MCPServerConfig) => {
    setServerToDelete(server);
    setDeleteDialogOpen(true);
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'connected':
        return <CheckCircle2 className="w-5 h-5 text-green-500 dark:text-green-400" />;
      case 'error':
        return <AlertTriangle className="w-5 h-5 text-red-500 dark:text-red-400" />;
      case 'connect_failed':
        return <AlertTriangle className="w-5 h-5 text-red-500 dark:text-red-400" />;
      case 'connecting':
        return <Loader2 className="w-5 h-5 text-blue-500 dark:text-blue-400 animate-spin" />;
      case 'sleeping':
        return <Moon className="w-5 h-5 text-muted-foreground" />;
      default:
        return <AlertTriangle className="w-5 h-5 text-yellow-500 dark:text-yellow-400" />;
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'connected':
        return 'Connected';
      case 'error':
        return 'Error';
      case 'connect_failed':
        return 'Connect Failed';
      case 'connecting':
        return 'Connecting';
      case 'sleeping':
        return 'Sleeping';
      default:
        return 'Not Configured';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'connected':
        return 'text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-950/20';
      case 'error':
        return 'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-950/20';
      case 'connect_failed':
        return 'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-950/20';
      case 'connecting':
        return 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950/20';
      case 'sleeping':
        return 'text-muted-foreground bg-muted';
      default:
        return 'text-yellow-600 dark:text-yellow-400 bg-yellow-50 dark:bg-yellow-950/20';
    }
  };

  // Handle adding custom tool
  const handleAddCustomTool = async () => {
    // Validate inputs based on transport type
    if (customTransportType === 'stdio') {
      if (!customStdioCommand.trim()) {
        toast.error('Command is required for stdio tools');
        return;
      }
    } else {
      if (!customMcpUrl.trim()) {
        toast.error('MCP URL is required');
        return;
      }
    }

    try {
      setDialogOpen(false);

      // Check if master password is cached
      const cachedPassword = MasterPasswordManager.getCachedMasterPassword();
      if (cachedPassword) {
        // Use cached password directly
        await performAddCustomTool(cachedPassword);
        return;
      }

      // Show master password dialog if not cached
      requestMasterPassword({
        title: 'Add Custom Tool - Master Password Required',
        description: 'Please enter your master password to add the custom tool.',
        onConfirm: async (password) => {
          await performAddCustomTool(password);
        },
      });
    } catch (error) {
      // Failed to add custom tool:
      toast.error('Could not add custom tool');
    }
  };

  // Add new header row
  const handleAddHeader = () => {
    setCustomHeaders((prev) => [...prev, createEditableKeyValueRow()]);
  };

  // Remove header row
  const handleRemoveHeader = (index: number) => {
    setCustomHeaders((prev) => (prev.length > 1 ? prev.filter((_, i) => i !== index) : prev));
  };

  // Update header key
  const handleHeaderKeyChange = (index: number, key: string) => {
    setCustomHeaders((prev) =>
      prev.map((header, i) => (i === index ? { ...header, key } : header)),
    );
  };

  // Update header value
  const handleHeaderValueChange = (index: number, value: string) => {
    setCustomHeaders((prev) =>
      prev.map((header, i) => (i === index ? { ...header, value } : header)),
    );
  };

  const performAddCustomTool = async (masterPwd: string) => {
    try {
      const { api } = await import('@/lib/api-client');
      const serverInfoResponse = await api.servers.getInfo();
      const proxyId = serverInfoResponse.data?.data?.proxyId;

      if (!proxyId) {
        toast.error('Unable to get server information');
        return;
      }

      // Build config based on transport type
      const headers: Record<string, string> = {};
      const validHeaders = customHeaders.filter(
        (header) => header.key.trim() && header.value.trim(),
      );
      validHeaders.forEach((header) => {
        headers[header.key.trim()] = header.value.trim();
      });

      const response = await api.tools.operateTool({
        handleType: 1, // 1 = add
        proxyId: proxyId,
        category: customTransportType === 'stdio' ? 5 : 2,
        ...(customTransportType === 'stdio'
          ? {
              stdioConfig: {
                command: customStdioCommand.trim(),
                ...(customStdioCwd.trim() ? { cwd: customStdioCwd.trim() } : {}),
                args: customStdioArgs.trim() ? customStdioArgs.split('\n') : [],
                env: Object.fromEntries(
                  customStdioEnvVars
                    .filter((e) => e.key.trim())
                    .map((e) => [e.key.trim(), e.value]),
                ),
              },
            }
          : {
              customRemoteConfig: {
                url: customMcpUrl.trim(),
                headers: headers,
              },
            }),
        allowUserInput: customAllowUserInput,
        lazyStartEnabled: customLazyStartEnabled,
        publicAccess: customPublicAccess,
        masterPwd,
      });

      if (response.data?.common?.code === 0) {
        toast.success('Custom tool added');
        // Clear form
        setCustomMcpUrl('');
        setCustomHeaders([createEditableKeyValueRow()]);
        setCustomAllowUserInput(0);
        setCustomPublicAccess(false);
        setCustomStdioCommand('');
        setCustomStdioCwd('');
        setCustomStdioArgs('');
        setCustomStdioEnvVars([createEditableKeyValueRow()]);
        setCustomLazyStartEnabled(true);
        setCustomStdioHiddenEnvRowIds(new Set());
        setCustomTransportType('url');
        // Reload tool list
        await loadToolData();
      } else {
        toast.error(response.data?.common?.message || 'Could not add custom tool');
      }
    } catch (error: any) {
      // Failed to add custom tool:
      const errorMessage =
        error.response?.data?.common?.message ||
        error.response?.data?.message ||
        'Could not add custom tool';
      toast.error(errorMessage);
    }
  };

  // Handle adding REST API tool
  const handleAddRestApiTool = async () => {
    // Clear previous error
    setRestApiConfigError(null);

    // Validate input not empty
    if (!restApiConfig.trim()) {
      setRestApiConfigError('REST API configuration is required');
      return;
    }

    try {
      // Import validation utilities
      const {
        detectFormat,
        validateRestApiConfig,
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
            toast.warning(
              `OpenAPI security scheme "${schemeName}" is "${schemeType}" and is not supported. ` +
                `Auth will be set to "none"; please update auth manually after conversion.`,
            );
          }

          parsedConfig = convertOpenApiToRestApiFormat(parsedConfig);
          toast.info('Converted OpenAPI specification to REST API format');

          // Show converted JSON back in the input for users to fill placeholders
          setRestApiConfig(JSON.stringify(parsedConfig, null, 2));

          // When allowUserInput = 1, skip checking auth placeholders
          const placeholders =
            restApiAllowUserInput === 1 ? [] : checkEnvironmentVariables(parsedConfig.auth);
          if (
            schemeType === 'oauth2' ||
            schemeType === 'openIdConnect' ||
            placeholders.length > 0
          ) {
            setRestApiConfigError(
              `OpenAPI has been converted to a REST API configuration. Please replace the placeholder values in "auth" (e.g., \${...}) with real credentials before submitting.` +
                (schemeType === 'oauth2' || schemeType === 'openIdConnect'
                  ? `\n\nNote: OpenAPI security scheme "${schemeName}" is "${schemeType}" and is not supported for automatic mapping. Auth will be set to "none"; please configure auth manually.`
                  : '') +
                (placeholders.length > 0
                  ? `\n\nDetected placeholders: ${placeholders.join(', ')}`
                  : ''),
            );
          }

          // Stop here so user can review/edit the converted config before submitting
          return;
        } catch (error: any) {
          setRestApiConfigError(`Could not convert OpenAPI spec: ${error.message}`);
          return;
        }
      }

      // Validate structure
      const validation = validateRestApiConfig(parsedConfig);
      if (!validation.valid) {
        setRestApiConfigError(validation.error || 'Invalid configuration');
        return;
      }

      // Size + tools count limits
      const configString = JSON.stringify(parsedConfig);
      const configBytes = new TextEncoder().encode(configString).length;
      if (configBytes > 100 * 1024) {
        setRestApiConfigError(
          `Configuration is too large (${Math.ceil(configBytes / 1024)} KB). Maximum allowed size is 100 KB.`,
        );
        return;
      }
      const toolCount = Array.isArray(parsedConfig?.tools) ? parsedConfig.tools.length : 0;
      if (toolCount > 50) {
        setRestApiConfigError(`Too many tools (${toolCount}). Maximum allowed tools count is 50.`);
        return;
      }

      // Check for environment variables
      // When allowUserInput = 1, exclude auth section before checking
      let envVars: string[] = [];
      if (restApiAllowUserInput === 1) {
        // Only check parts other than auth
        const { auth, ...configWithoutAuth } = parsedConfig;
        envVars = checkEnvironmentVariables(configWithoutAuth);
      } else {
        // Check entire configuration
        envVars = checkEnvironmentVariables(parsedConfig);
      }

      if (envVars.length > 0) {
        const confirmed = window.confirm(
          `Configuration contains environment variable placeholders: ${envVars.join(', ')}\n\n` +
            `Please ensure you have replaced these with actual values before proceeding.\n\n` +
            `Do you want to continue?`,
        );
        if (!confirmed) {
          return;
        }
      }

      // Close dialog
      setDialogOpen(false);

      // Check if master password is cached
      const cachedPassword = MasterPasswordManager.getCachedMasterPassword();
      if (cachedPassword) {
        await performAddRestApiTool(parsedConfig, cachedPassword);
        return;
      }

      // Show master password dialog if not cached
      requestMasterPassword({
        title: 'Add REST API Tool - Master Password Required',
        description: 'Please enter your master password to add the REST API tool.',
        onConfirm: async (password) => {
          await performAddRestApiTool(parsedConfig, password);
        },
      });
    } catch (error: any) {
      // Failed to add REST API tool:
      setRestApiConfigError(error.message || 'Could not process configuration');
    }
  };

  const performAddRestApiTool = async (parsedConfig: any, masterPwd: string) => {
    try {
      const { api } = await import('@/lib/api-client');
      const serverInfoResponse = await api.servers.getInfo();
      const proxyId = serverInfoResponse.data?.data?.proxyId;

      if (!proxyId) {
        toast.error('Unable to get server information');
        return;
      }

      // Convert parsed config to JSON string
      const restApiConfigString = JSON.stringify(parsedConfig);

      // Call protocol 10005 to add REST API tool with category=3
      const response = await api.tools.operateTool({
        handleType: 1, // 1 = add
        proxyId: proxyId,
        category: 3, // 3 = REST API tool
        restApiConfig: restApiConfigString,
        allowUserInput: restApiAllowUserInput,
        lazyStartEnabled: restApiLazyStartEnabled,
        publicAccess: restApiPublicAccess,
        masterPwd,
      });

      if (response.data?.common?.code === 0) {
        toast.success('REST API tool added');
        // Clear form
        setRestApiConfig('');
        setRestApiConfigError(null);
        setRestApiAllowUserInput(0);
        setRestApiPublicAccess(false);
        // Reload tool list
        await loadToolData();
      } else {
        toast.error(response.data?.common?.message || 'Could not add REST API tool');
      }
    } catch (error: any) {
      // Failed to add REST API tool:
      const errorMessage =
        error.response?.data?.common?.message ||
        error.response?.data?.message ||
        'Could not add REST API tool';
      toast.error(errorMessage);
    }
  };

  // Handle adding Skills MCP Server
  const handleAddSkillsServer = async () => {
    if (!skillsServerName.trim()) {
      toast.error('Server name is required');
      return;
    }

    // Close dialog
    setDialogOpen(false);

    // Check if master password is cached
    const cachedPassword = MasterPasswordManager.getCachedMasterPassword();
    if (cachedPassword) {
      await performAddSkillsServer(cachedPassword);
      return;
    }

    // Show master password dialog if not cached
    requestMasterPassword({
      title: 'Add Skills MCP Server - Master Password Required',
      description: 'Please enter your master password to add the Skills MCP Server.',
      onConfirm: async (password) => {
        await performAddSkillsServer(password);
      },
    });
  };

  const performAddSkillsServer = async (masterPwd: string) => {
    setIsAddingSkillsServer(true);
    try {
      const { api } = await import('@/lib/api-client');
      const serverInfoResponse = await api.servers.getInfo();
      const proxyId = serverInfoResponse.data?.data?.proxyId;

      if (!proxyId) {
        toast.error('Unable to get server information');
        return;
      }

      // Step 1: Create Skills MCP Server using category=4
      const response = await api.tools.operateTool({
        handleType: 1, // 1 = add
        proxyId: proxyId,
        category: 4, // 4 = Skills tool
        serverName: skillsServerName.trim(),
        lazyStartEnabled: skillsLazyStartEnabled,
        publicAccess: skillsPublicAccess,
        masterPwd,
      });

      if (response.data?.common?.code !== 0) {
        toast.error(response.data?.common?.message || 'Could not add Skills MCP Server');
        return;
      }

      const toolId = response.data?.data?.toolId;
      if (!toolId) {
        toast.error('Could not get server ID');
        return;
      }

      toast.success('Skills MCP Server created');

      // Step 2: If ZIP file is provided, upload skills
      if (skillsZipFile) {
        try {
          const zipBuffer = await skillsZipFile.arrayBuffer();
          // Convert to base64 (much more efficient than JSON number array)
          const uint8Array = new Uint8Array(zipBuffer);
          let binary = '';
          for (let i = 0; i < uint8Array.length; i++) {
            binary += String.fromCharCode(uint8Array[i]);
          }
          const base64Data = btoa(binary);

          const uploadResponse = await api.tools.uploadSkills({
            serverId: toolId,
            data: base64Data,
          });

          if (uploadResponse.data?.common?.code === 0) {
            toast.success('Skills uploaded');
          } else {
            toast.warning(
              `Server created but skills upload failed: ${uploadResponse.data?.common?.message || 'Unknown error'}`,
            );
          }
        } catch (uploadError: any) {
          // Failed to upload skills:
          toast.warning(
            `Server created but skills upload failed: ${uploadError.message || 'Unknown error'}`,
          );
        }
      }

      // Clear form
      setSkillsServerName('Skills MCP Server');
      setSkillsZipFile(null);
      setSkillsLazyStartEnabled(true);
      setSkillsPublicAccess(false);

      // Reload tool list
      await loadToolData();
    } catch (error: any) {
      // Failed to add Skills MCP Server:
      const errorMessage =
        error.response?.data?.common?.message ||
        error.response?.data?.message ||
        'Could not add Skills MCP Server';
      toast.error(errorMessage);
    } finally {
      setIsAddingSkillsServer(false);
    }
  };

  const handleAddPresetTool = async (serverType: string, templateId: string) => {
    // Get template by templateId, then check if authType is supported, if not, toast error message.
    const template = toolTemplates.find((tool) => tool.toolTmplId === templateId);
    if (!template) {
      toast.error('Template not found');
      return;
    }
    // Check if template.authType is in ServerAuthType enum definition. If not, toast error message.
    if (!Object.values(ServerAuthType).includes(template.authType)) {
      toast.error('Template auth type not supported');
      return;
    }

    // Generate a temporary ID for the configuration page
    const tempId = generateConsistentId(`${serverType}-${Date.now()}`);

    setDialogOpen(false);
    // Navigate directly to configuration page with templateId parameter
    router.push(
      `/dashboard/tool-configure/${tempId}?mode=setup&type=${serverType}&templateId=${templateId}`,
    );
  };

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-4 mb-2">
        <p className="text-sm text-amber-800 dark:text-amber-300">
          <strong>Deprecated:</strong> Tool Configuration has been replaced by{' '}
          <a href="/dashboard/skills" className="underline font-medium">
            Skills
          </a>
          . This page will be removed in a future release.
        </p>
      </div>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-0">
          <h1 className="text-[30px] font-bold">Tool Configuration</h1>
          <p className="text-base text-muted-foreground">
            Manage MCP tools and their authentication.
          </p>
        </div>
        <div>
          <Dialog
            open={dialogOpen}
            onOpenChange={(open) => {
              setDialogOpen(open);
              if (open) {
                loadToolTemplates();
              }
            }}
          >
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <DialogTrigger asChild>
                    <Button
                      className="bg-playground-gradient hover:opacity-90 text-white"
                      onClick={() => setDialogOpen(true)}
                    >
                      <Plus className="mr-2 h-4 w-4" /> Add Tool
                    </Button>
                  </DialogTrigger>
                </TooltipTrigger>
                <TooltipContent>
                  <p>Add new tool</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
            <DialogContent
              className="sm:max-w-[650px] flex flex-col overflow-hidden"
              style={{
                maxHeight: 'calc(100vh - 160px)',
                height: 'calc(100vh - 160px)',
                display: 'flex',
                flexDirection: 'column',
              }}
            >
              <DialogHeader className="px-1">
                <DialogTitle>Add Tool</DialogTitle>
              </DialogHeader>

              <Tabs defaultValue="templates" className="flex-1 flex flex-col overflow-hidden">
                <div className="px-1">
                  <TabsList className="grid w-full grid-cols-4">
                    <TabsTrigger value="templates">Templates</TabsTrigger>
                    <TabsTrigger value="custom">Custom</TabsTrigger>
                    <TabsTrigger value="rest-api">REST API</TabsTrigger>
                    <TabsTrigger value="skills">Skills</TabsTrigger>
                  </TabsList>
                </div>

                <TabsContent
                  value="templates"
                  className="flex-1 overflow-y-auto mt-4 min-h-0"
                  onFocus={() => {
                    loadToolTemplates();
                  }}
                >
                  <div className="px-1 pt-1">
                    {/* Search Box */}
                    <div className="relative mb-4">
                      <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                      <Input
                        placeholder="Search templates..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="pl-8"
                      />
                    </div>
                  </div>
                  {isLoadingTemplates ? (
                    <div className="flex items-center justify-center p-8 px-1">
                      <Loader2 className="h-6 w-6 animate-spin" />
                      <span className="ml-2 text-sm text-muted-foreground">
                        Loading templates...
                      </span>
                    </div>
                  ) : toolTemplates.length === 0 ? (
                    <div className="text-center py-8 px-1">
                      <p className="text-sm text-muted-foreground">
                        No tool templates available. Please check your connection.
                      </p>
                    </div>
                  ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-3 px-1">
                      {toolTemplates
                        .filter((tool) => {
                          if (!searchQuery) return true;
                          const query = searchQuery.toLowerCase();
                          return (
                            tool.name?.toLowerCase().includes(query) ||
                            tool.description?.toLowerCase().includes(query) ||
                            tool.tags?.some((tag: string) => tag.toLowerCase().includes(query))
                          );
                        })
                        .map((tool, index) => {
                          const toolData = {
                            name: tool.name,
                            description: tool.description,
                            serverType: getServerTypeFromToolType(tool.toolType),
                            tags: tool.tags,
                            authtags: tool.authtags,
                          };

                          return (
                            <Card key={tool.toolId || index} className="flex flex-col p-4">
                              <div className="grid gap-1 flex-1">
                                <div className="flex items-center gap-2 mb-1">
                                  <div className="flex-shrink-0 w-8 h-8 rounded-md bg-gradient-to-br from-blue-500 to-indigo-600 flex items-center justify-center">
                                    <Zap className="h-4 w-4 text-white" />
                                  </div>
                                  <h3 className="font-semibold">{toolData.name}</h3>
                                </div>
                                <p className="text-sm text-muted-foreground">
                                  {toolData.description}
                                </p>
                                {/* <div className="flex gap-2 mt-2">
                                  {toolData.tags?.map((tag: string) => (
                                    <Badge variant="secondary" key={tag}>
                                      {tag}
                                    </Badge>
                                  ))}
                                  {toolData.authtags?.map((tag: string) => (
                                    <Badge variant="outline" key={tag}>
                                      {tag}
                                    </Badge>
                                  ))}
                                </div> */}
                              </div>
                              {/* Action buttons */}
                              <div className="mt-4 flex justify-end">
                                <Button
                                  variant="outline"
                                  size="sm"
                                  className="w-full sm:w-[100px]"
                                  onClick={() =>
                                    handleAddPresetTool(toolData.serverType, tool.toolTmplId)
                                  }
                                >
                                  Configure
                                </Button>
                              </div>
                            </Card>
                          );
                        })}
                    </div>
                  )}
                </TabsContent>

                <TabsContent value="custom" className="flex-1 overflow-y-auto mt-4 min-h-0">
                  <div className="space-y-4 px-1 pt-1">
                    {/* Connection Type Selector */}
                    <div className="space-y-2">
                      <Label className="text-sm font-medium">Connection Type</Label>
                      <div className="flex gap-2">
                        <Button
                          type="button"
                          variant={customTransportType === 'url' ? 'default' : 'outline'}
                          size="sm"
                          onClick={() => setCustomTransportType('url')}
                          className="flex-1"
                        >
                          Remote URL
                        </Button>
                        <Button
                          type="button"
                          variant={customTransportType === 'stdio' ? 'default' : 'outline'}
                          size="sm"
                          onClick={() => setCustomTransportType('stdio')}
                          className="flex-1"
                        >
                          Stdio Command
                        </Button>
                      </div>
                    </div>

                    {customTransportType === 'url' && (
                      <>
                        {/* MCP URL */}
                        <div className="space-y-2">
                          <Label htmlFor="custom-mcp-url">
                            MCP URL <span className="text-red-500 dark:text-red-400">*</span>
                          </Label>
                          <Input
                            id="custom-mcp-url"
                            placeholder="e.g., http://localhost:3000 or https://mcp.example.com"
                            value={customMcpUrl}
                            onChange={(e) => setCustomMcpUrl(e.target.value)}
                          />
                          <p className="text-xs text-muted-foreground">
                            Enter the remote MCP server URL
                          </p>
                        </div>

                        {/* Headers */}
                        <div className="space-y-2">
                          <div className="flex items-center justify-between">
                            <Label>Headers (Optional)</Label>
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
                            {customHeaders.map((header, index) => (
                              <div key={header.id} className="flex gap-2 items-start">
                                <Input
                                  aria-label={`Header ${index + 1} key`}
                                  placeholder="Key"
                                  value={header.key}
                                  onChange={(e) => handleHeaderKeyChange(index, e.target.value)}
                                  className="flex-1"
                                />
                                <Input
                                  aria-label={`Header ${index + 1} value`}
                                  placeholder="Value"
                                  value={header.value}
                                  onChange={(e) => handleHeaderValueChange(index, e.target.value)}
                                  className="flex-1"
                                />
                                {customHeaders.length > 1 && (
                                  <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => handleRemoveHeader(index)}
                                    aria-label={`Remove header ${index + 1}`}
                                    className="h-10 w-10 p-0 hover:bg-red-50 dark:hover:bg-red-950/20 hover:text-red-600 dark:text-red-400 dark:hover:text-red-400"
                                  >
                                    <X className="h-4 w-4" />
                                  </Button>
                                )}
                              </div>
                            ))}
                          </div>
                          <p className="text-xs text-muted-foreground">
                            Add custom headers for the MCP server connection
                          </p>
                        </div>
                      </>
                    )}

                    {customTransportType === 'stdio' && (
                      <>
                        {/* Command */}
                        <div className="space-y-2">
                          <Label htmlFor="custom-stdio-command">
                            Command <span className="text-red-500 dark:text-red-400">*</span>
                          </Label>
                          <Input
                            id="custom-stdio-command"
                            placeholder="e.g., npx, uvx, docker"
                            value={customStdioCommand}
                            onChange={(e) => setCustomStdioCommand(e.target.value)}
                            className="font-mono text-sm"
                          />
                          <p className="text-xs text-muted-foreground">
                            The executable command to start the MCP server process
                          </p>
                        </div>

                        {/* Arguments */}
                        <div className="space-y-2">
                          <Label htmlFor="custom-stdio-args">Arguments</Label>
                          <Textarea
                            id="custom-stdio-args"
                            value={customStdioArgs}
                            onChange={(e) => setCustomStdioArgs(e.target.value)}
                            placeholder="One argument per line"
                            className="font-mono text-sm min-h-[100px]"
                          />
                          <p className="text-xs text-muted-foreground">
                            Command-line arguments passed to the server process. Enter each argument
                            on a separate line.
                          </p>
                        </div>

                        {/* Environment Variables */}
                        <div className="space-y-2">
                          <div className="flex items-center justify-between">
                            <Label>Environment Variables (Optional)</Label>
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              onClick={() =>
                                setCustomStdioEnvVars((prev) => [
                                  ...prev,
                                  createEditableKeyValueRow(),
                                ])
                              }
                              className="h-8"
                            >
                              <Plus className="h-3 w-3 mr-1" />
                              Add Variable
                            </Button>
                          </div>
                          <div className="space-y-2">
                            {customStdioEnvVars.map((envVar, index) => (
                              <div key={envVar.id} className="flex gap-2 items-start">
                                <Input
                                  aria-label={`Environment variable ${index + 1} key`}
                                  placeholder="Key"
                                  value={envVar.key}
                                  onChange={(e) => {
                                    const newKey = e.target.value;
                                    setCustomStdioEnvVars((prev) =>
                                      prev.map((row, i) =>
                                        i === index ? { ...row, key: newKey } : row,
                                      ),
                                    );
                                    setCustomStdioHiddenEnvRowIds((prev) => {
                                      const next = new Set(prev);
                                      if (isSensitiveEnvKey(newKey)) next.add(envVar.id);
                                      else next.delete(envVar.id);
                                      return next;
                                    });
                                  }}
                                  className="flex-1"
                                />
                                <Input
                                  aria-label={`Environment variable ${index + 1} value`}
                                  type={
                                    isSensitiveEnvKey(envVar.key) &&
                                    customStdioHiddenEnvRowIds.has(envVar.id)
                                      ? 'password'
                                      : 'text'
                                  }
                                  placeholder="Value"
                                  value={envVar.value}
                                  onChange={(e) => {
                                    setCustomStdioEnvVars((prev) =>
                                      prev.map((row, i) =>
                                        i === index ? { ...row, value: e.target.value } : row,
                                      ),
                                    );
                                  }}
                                  className="flex-1"
                                />
                                {isSensitiveEnvKey(envVar.key) && (
                                  <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => {
                                      setCustomStdioHiddenEnvRowIds((prev) => {
                                        const next = new Set(prev);
                                        if (next.has(envVar.id)) next.delete(envVar.id);
                                        else next.add(envVar.id);
                                        return next;
                                      });
                                    }}
                                    aria-label={
                                      customStdioHiddenEnvRowIds.has(envVar.id)
                                        ? `Show value for environment variable ${index + 1}`
                                        : `Hide value for environment variable ${index + 1}`
                                    }
                                    className="h-10 w-10 p-0"
                                  >
                                    {customStdioHiddenEnvRowIds.has(envVar.id) ? (
                                      <Eye className="h-4 w-4" />
                                    ) : (
                                      <EyeOff className="h-4 w-4" />
                                    )}
                                  </Button>
                                )}
                                {customStdioEnvVars.length > 1 && (
                                  <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => {
                                      setCustomStdioEnvVars((prev) =>
                                        prev.filter((_, i) => i !== index),
                                      );
                                      setCustomStdioHiddenEnvRowIds((prev) => {
                                        const next = new Set(prev);
                                        next.delete(envVar.id);
                                        return next;
                                      });
                                    }}
                                    aria-label={`Remove environment variable ${index + 1}`}
                                    className="h-10 w-10 p-0 hover:bg-red-50 dark:hover:bg-red-950/20 hover:text-red-600 dark:text-red-400 dark:hover:text-red-400"
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

                        {/* Working Directory */}
                        <div className="space-y-2">
                          <Label htmlFor="custom-stdio-cwd">Working Directory</Label>
                          <Input
                            id="custom-stdio-cwd"
                            placeholder="e.g., /path/to/project (optional)"
                            value={customStdioCwd}
                            onChange={(e) => setCustomStdioCwd(e.target.value)}
                            className="font-mono text-sm"
                          />
                          <p className="text-xs text-muted-foreground">
                            Optional working directory used when spawning the MCP server process
                          </p>
                        </div>
                      </>
                    )}
                    <AllowUserInputConfiguration
                      checked={customAllowUserInput}
                      onCheckedChange={setCustomAllowUserInput}
                    />

                    <PublicAccessConfiguration
                      checked={customPublicAccess}
                      onCheckedChange={setCustomPublicAccess}
                    />

                    <LazyStartConfiguration
                      checked={customLazyStartEnabled}
                      onCheckedChange={setCustomLazyStartEnabled}
                      supportsIdleSleep={customTransportType === 'stdio'}
                    />

                    <div className="flex justify-end gap-2 pt-4">
                      <Button
                        variant="outline"
                        onClick={() => {
                          setDialogOpen(false);
                          // Clear form when canceling
                          setCustomMcpUrl('');
                          setCustomHeaders([createEditableKeyValueRow()]);
                          setCustomAllowUserInput(0);
                          setCustomPublicAccess(false);
                          setCustomStdioCommand('');
                          setCustomStdioCwd('');
                          setCustomStdioArgs('');
                          setCustomStdioEnvVars([createEditableKeyValueRow()]);
                          setCustomStdioHiddenEnvRowIds(new Set());
                          setCustomTransportType('url');
                        }}
                      >
                        Cancel
                      </Button>
                      <Button onClick={handleAddCustomTool}>Add Custom Tool</Button>
                    </div>
                  </div>
                </TabsContent>

                <TabsContent value="rest-api" className="flex-1 overflow-y-auto mt-4">
                  <div className="space-y-4 pb-0 px-1 pt-1">
                    {/* Configuration Input */}
                    <div className="space-y-2">
                      <Label htmlFor="rest-api-config">
                        REST API Configuration{' '}
                        <span className="text-red-500 dark:text-red-400">*</span>
                      </Label>
                      <Textarea
                        id="rest-api-config"
                        className="min-h-[300px] font-mono text-sm resize-y"
                        placeholder={`Paste your REST API configuration in JSON or YAML format, or OpenAPI 3.x specification...

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
                        value={restApiConfig}
                        onChange={(e) => {
                          setRestApiConfig(e.target.value);
                          // Clear error when user starts typing
                          if (restApiConfigError) {
                            setRestApiConfigError(null);
                          }
                        }}
                      />
                      <div className="flex flex-col gap-2">
                        <p className="text-xs text-muted-foreground">
                          Supports: JSON format, YAML format, or OpenAPI 3.x specification
                        </p>
                        {restApiConfigError && (
                          <div
                            role="alert"
                            className="bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded-lg p-3"
                          >
                            <div className="flex items-start gap-2">
                              <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
                              <p className="text-sm text-red-800 dark:text-red-300">
                                {restApiConfigError}
                              </p>
                            </div>
                          </div>
                        )}
                      </div>
                    </div>

                    {/* Information Box */}
                    <div className="bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                      <div className="flex items-start gap-2">
                        <FileText className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                        <div className="text-sm text-blue-800 dark:text-blue-300">
                          <p className="font-medium mb-1">Important Notes:</p>
                          <ul className="list-disc list-inside space-y-0.5 text-xs">
                            <li>Required fields: name, description, baseUrl, tools</li>
                            <li>
                              Replace environment variables like ${'{'}API_KEY{'}'} with actual
                              values
                            </li>
                            <li>OpenAPI specifications will be automatically converted</li>
                            <li>Configuration will be validated before submission</li>
                          </ul>
                        </div>
                      </div>
                    </div>

                    {hasRestApiInput && <RestApiValidationPanel configText={restApiConfig} />}

                    <AllowUserInputConfiguration
                      checked={restApiAllowUserInput}
                      onCheckedChange={setRestApiAllowUserInput}
                    />

                    <PublicAccessConfiguration
                      checked={restApiPublicAccess}
                      onCheckedChange={setRestApiPublicAccess}
                    />

                    <LazyStartConfiguration
                      checked={restApiLazyStartEnabled}
                      onCheckedChange={setRestApiLazyStartEnabled}
                    />

                    <div className="flex justify-end gap-2 pt-4">
                      <Button
                        variant="outline"
                        onClick={() => {
                          setDialogOpen(false);
                          setRestApiConfig('');
                          setRestApiConfigError(null);
                          setRestApiAllowUserInput(0);
                          setRestApiPublicAccess(false);
                          setCustomMcpUrl('');
                          setCustomHeaders([createEditableKeyValueRow()]);
                          setCustomPublicAccess(false);
                        }}
                      >
                        Cancel
                      </Button>
                      <Button onClick={handleAddRestApiTool}>Add REST API Tool</Button>
                    </div>
                  </div>
                </TabsContent>

                <TabsContent value="skills" className="flex-1 overflow-y-auto mt-4">
                  <div className="space-y-4 pb-0 px-1 pt-1">
                    {/* Server Name */}
                    <div className="space-y-2">
                      <Label htmlFor="skills-server-name">
                        Server Name <span className="text-red-500 dark:text-red-400">*</span>
                      </Label>
                      <Input
                        id="skills-server-name"
                        placeholder="e.g., My Skills Server"
                        value={skillsServerName}
                        onChange={(e) => setSkillsServerName(e.target.value)}
                      />
                    </div>

                    {/* ZIP Upload Area */}
                    <div className="space-y-2">
                      <Label>Add Skills</Label>
                      <input
                        ref={skillsFileInputRef}
                        type="file"
                        accept=".zip"
                        className="hidden"
                        onChange={(e) => {
                          const file = e.target.files?.[0];
                          if (!file) return;
                          if (file.size > SKILLS_MAX_FILE_SIZE) {
                            toast.error(
                              `File size exceeds the maximum limit of ${SKILLS_MAX_FILE_SIZE / 1024 / 1024}MB`,
                            );
                            return;
                          }
                          setSkillsZipFile(file);
                          // Reset input value to allow selecting the same file again
                          e.target.value = '';
                        }}
                      />
                      <button
                        type="button"
                        className={cn(
                          'border border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors',
                          'w-full',
                          skillsDragActive
                            ? 'border-blue-500 bg-blue-50 dark:bg-blue-950/20'
                            : skillsZipFile
                              ? 'border-green-500 bg-green-50 dark:bg-green-950/20'
                              : 'border-muted-foreground/30 hover:border-muted-foreground/50',
                        )}
                        onDragEnter={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          setSkillsDragActive(true);
                        }}
                        onDragLeave={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          setSkillsDragActive(false);
                        }}
                        onDragOver={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                        }}
                        onDrop={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          setSkillsDragActive(false);
                          const file = e.dataTransfer.files?.[0];
                          if (!file) return;
                          if (!file.name.toLowerCase().endsWith('.zip')) {
                            toast.error('Please select a ZIP file');
                            return;
                          }
                          if (file.size > SKILLS_MAX_FILE_SIZE) {
                            toast.error(
                              `File size exceeds the maximum limit of ${SKILLS_MAX_FILE_SIZE / 1024 / 1024}MB`,
                            );
                            return;
                          }
                          setSkillsZipFile(file);
                        }}
                        onClick={() => {
                          skillsFileInputRef.current?.click();
                        }}
                      >
                        {skillsZipFile ? (
                          <div className="flex flex-col items-center gap-2">
                            <FolderArchive className="h-10 w-10 text-green-600 dark:text-green-400" />
                            <p className="font-medium text-green-700 dark:text-green-300">
                              {skillsZipFile.name}
                            </p>
                            <p className="text-sm text-green-600 dark:text-green-400">
                              {(skillsZipFile.size / 1024).toFixed(1)} KB
                            </p>
                          </div>
                        ) : (
                          <div className="flex flex-col items-center gap-2">
                            <Upload className="h-8 w-8 text-muted-foreground" />
                            <p className="font-medium text-muted-foreground">
                              Drag & drop your skills ZIP file
                            </p>
                            <p className="text-sm text-muted-foreground">or click to select</p>
                          </div>
                        )}
                      </button>
                      {skillsZipFile && (
                        <div className="flex justify-end">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation();
                              setSkillsZipFile(null);
                            }}
                            className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 hover:bg-red-50 dark:hover:bg-red-950/20"
                          >
                            <X className="h-4 w-4 mr-1" />
                            Remove
                          </Button>
                        </div>
                      )}
                    </div>

                    {/* Information Box */}
                    <div className="bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                      <div className="flex items-start gap-2">
                        <Info className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                        <div className="text-sm text-blue-800 dark:text-blue-300">
                          <p className="font-medium mb-1">ZIP Structure:</p>
                          <ul className="text-xs space-y-1 list-disc list-inside">
                            <li>
                              Each subdirectory containing a SKILL.md file will be treated as an
                              individual skill
                            </li>
                            <li>
                              You can compress your entire skills folder or select multiple skill
                              directories
                            </li>
                            <li>Skills with the same name will be replaced</li>
                            <li>Maximum ZIP file size: {SKILLS_MAX_FILE_SIZE / 1024 / 1024}MB</li>
                          </ul>
                        </div>
                      </div>
                    </div>

                    <PublicAccessConfiguration
                      checked={skillsPublicAccess}
                      onCheckedChange={setSkillsPublicAccess}
                    />

                    <LazyStartConfiguration
                      checked={skillsLazyStartEnabled}
                      onCheckedChange={setSkillsLazyStartEnabled}
                    />

                    <div className="flex justify-end gap-2 pt-4">
                      <Button
                        variant="outline"
                        onClick={() => {
                          setDialogOpen(false);
                          setSkillsServerName('Skills MCP Server');
                          setSkillsZipFile(null);
                          setSkillsLazyStartEnabled(true);
                          setSkillsPublicAccess(false);
                        }}
                      >
                        Cancel
                      </Button>
                      <Button
                        onClick={handleAddSkillsServer}
                        disabled={!skillsServerName.trim() || isAddingSkillsServer}
                      >
                        {isAddingSkillsServer ? (
                          <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Adding...
                          </>
                        ) : (
                          'Add Skills MCP Server'
                        )}
                      </Button>
                    </div>
                  </div>
                </TabsContent>
              </Tabs>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-red-600 dark:text-red-400">
              <AlertTriangle className="h-5 w-5" />
              Delete Tool
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Are you sure you want to delete <strong>{serverToDelete?.name}</strong>?
            </p>
            <div className="bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded-lg p-3">
              <div className="flex items-start gap-2">
                <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
                <div className="text-sm text-red-800 dark:text-red-300">
                  <p className="font-medium mb-1">This action cannot be undone.</p>
                  <ul className="list-disc list-inside space-y-0.5 text-xs">
                    <li>All tool configurations will be permanently deleted</li>
                    <li>Active connections will be terminated</li>
                    <li>Stored credentials will be removed</li>
                  </ul>
                </div>
              </div>
            </div>
            <div className="flex justify-end gap-3">
              <Button
                variant="outline"
                onClick={() => {
                  setDeleteDialogOpen(false);
                  setServerToDelete(null);
                }}
                disabled={!!serverToDelete && operatingServers.has(serverToDelete.id)}
              >
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={() => serverToDelete && handleDeleteTool(serverToDelete)}
                disabled={!serverToDelete || operatingServers.has(serverToDelete.id)}
              >
                {serverToDelete && operatingServers.has(serverToDelete.id) ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Deleting...
                  </>
                ) : (
                  <>
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete Tool
                  </>
                )}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
      <Card>
        <CardContent className="p-0">
          {/* Tool List */}
          {isLoadingTools ? (
            <div className="flex items-center justify-center p-8">
              <Loader2 className="h-6 w-6 animate-spin" />
              <span className="ml-2 text-sm text-muted-foreground">Loading tools...</span>
            </div>
          ) : servers.length === 0 ? (
            // Empty state when no tools
            <div className="text-center py-16">
              <div className="mx-auto mb-4 w-16 h-16 bg-muted rounded-full flex items-center justify-center">
                <Settings className="w-8 h-8 text-muted-foreground" />
              </div>
              <h3 className="text-lg font-semibold mb-2">No Tools Configured</h3>
              <p className="text-muted-foreground mb-6 max-w-md mx-auto">
                No MCP tools added yet. Add a tool to get started.
              </p>
              <Button onClick={() => setDialogOpen(true)} className="mx-auto">
                <Plus className="mr-2 h-4 w-4" />
                Add Your First Tool
              </Button>
            </div>
          ) : (
            <div className="divide-y">
              {servers.map((server) => {
                const isConfigured = server.status === 'connected' || server.status === 'sleeping';

                return (
                  <div
                    key={server.id}
                    className={cn(
                      'p-4 hover:bg-muted/50 transition-colors',
                      (server.status === 'connect_failed' || server.status === 'not_configured') &&
                        'bg-red-50/50 dark:bg-red-950/20',
                    )}
                  >
                    <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                      {/* Left Section - Tool Info */}
                      <div className="flex items-center gap-3 flex-1">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <h3 className="font-semibold text-base text-foreground truncate max-w-[300px]">
                                    {server.name}
                                  </h3>
                                </TooltipTrigger>
                                <TooltipContent>
                                  <p>{server.name}</p>
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                            {/* Show connection status for non-allowUserInput tools OR custom MCP tools */}
                            {server.allowUserInput !== 1 && (
                              <>
                                {server.status === 'connected' ? (
                                  // Green dot for successful connection
                                  <div className="w-2 h-2 rounded-full bg-green-500 dark:bg-green-400"></div>
                                ) : server.status === 'sleeping' ? (
                                  // Gray dot for sleeping state
                                  <div className="w-2 h-2 rounded-full bg-gray-400 dark:bg-gray-500"></div>
                                ) : (
                                  // Text badge for error/failed/connecting connection
                                  <div
                                    className={`flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${getStatusColor(
                                      server.status,
                                    )}`}
                                  >
                                    {getStatusIcon(server.status)}
                                    <span>{getStatusText(server.status)}</span>
                                  </div>
                                )}
                              </>
                            )}
                          </div>
                        </div>
                      </div>

                      {/* Right Section - Actions */}
                      <div className="flex items-center flex-wrap gap-2">
                        {/* Check if restart button should be shown */}
                        {(() => {
                          const showRestart =
                            server.enabled &&
                            server.allowUserInput !== 1 &&
                            (server.status === 'connect_failed' ||
                              server.status === 'sleeping' ||
                              server.status === 'error');

                          return (
                            <>
                              {/* Show Restart button for enabled tools with connection issues (exclude allowUserInput=1) */}
                              {showRestart && (
                                <Button
                                  variant="outline"
                                  size="sm"
                                  className="px-3 py-1.5 h-auto bg-orange-50 dark:bg-orange-950/20 border-orange-200 dark:border-orange-800 text-orange-700 dark:text-orange-300 hover:bg-orange-100 dark:hover:bg-orange-950/30"
                                  onClick={() => handleRestartTool(server.id)}
                                  disabled={operatingServers.has(server.id)}
                                >
                                  {operatingServers.has(server.id) ? (
                                    <>
                                      <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                                      Restarting...
                                    </>
                                  ) : (
                                    <>
                                      <RefreshCw className="h-4 w-4 mr-1" />
                                      Restart
                                    </>
                                  )}
                                </Button>
                              )}

                              {/* Credentials button - show for all tools */}
                              <Link
                                href={`/dashboard/tool-configure/${server.id}?mode=credentials&isEdit=true`}
                              >
                                <Button
                                  variant="outline"
                                  size="sm"
                                  className="px-3 py-1.5 h-auto font-[400]"
                                >
                                  Settings
                                </Button>
                              </Link>

                              {/* Functions and Resources buttons - only show when restart button is NOT shown */}
                              {!showRestart && server.allowUserInput !== 1 && (
                                <>
                                  <Link
                                    href={`/dashboard/tool-configure/${server.id}?mode=functions&isEdit=true`}
                                  >
                                    <Button
                                      variant="outline"
                                      size="sm"
                                      className="px-3 py-1.5 h-auto font-[400]"
                                    >
                                      Functions - {server.enabledFunctions?.length || 0}/
                                      {server.totalFunctions || 0}
                                    </Button>
                                  </Link>

                                  <Link
                                    href={`/dashboard/tool-configure/${server.id}?mode=permissions&isEdit=true`}
                                  >
                                    <Button
                                      variant="outline"
                                      size="sm"
                                      className="px-3 py-1.5 h-auto font-[400]"
                                    >
                                      Resources -{' '}
                                      {server.dataPermissions?.allowedResources?.length || 0}/
                                      {server.totalResources || 0}
                                    </Button>
                                  </Link>
                                </>
                              )}
                            </>
                          );
                        })()}

                        {/* Enable/Disable Switch */}
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <div className="flex items-center">
                                {operatingServers.has(server.id) ? (
                                  <div className="flex items-center gap-2">
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                  </div>
                                ) : (
                                  <Switch
                                    aria-label={`${server.enabled ? 'Disable' : 'Enable'} tool ${server.name}`}
                                    checked={server.enabled ?? false}
                                    onCheckedChange={(checked) =>
                                      handleToggleServer(server.id, checked)
                                    }
                                    size="sm"
                                  />
                                )}
                              </div>
                            </TooltipTrigger>
                            <TooltipContent>
                              <p>
                                {operatingServers.has(server.id)
                                  ? 'Updating...'
                                  : server.enabled
                                    ? 'Disable tool'
                                    : 'Enable tool'}
                              </p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>

                        {/* <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="text-red-500 hover:text-red-600 dark:text-red-400 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/20"
                                onClick={() => openDeleteDialog(server)}
                                disabled={operatingServers.has(server.id)}
                              >
                                {operatingServers.has(server.id) ? (
                                  <Loader2 className="h-4 w-4 animate-spin" />
                                ) : (
                                  <Trash2 className="h-4 w-4" />
                                )}
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>
                              <p>
                                {operatingServers.has(server.id)
                                  ? 'Deleting...'
                                  : 'Delete Tool'}
                              </p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider> */}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
