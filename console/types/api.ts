// API Types based on api.proto

export interface ApiRequest {
  cmdId: number;
}

export interface ApiResponse {
  code: number;
  message: string;
  cmdId: number;
  errorCode: string;
}

export interface KV {
  key: string;
  value: string;
}

export interface Credential {
  name: string;
  description: string;
  dataType: number; // 1-input, 2-radio, 3-checkbox, 4-select...
  key: string;
  value: string;
  options: KV[];
  selected: KV;
}

export interface ToolFunction {
  funcName: string;
  enabled: boolean; // default true
}

export interface ToolResource {
  uri: string;
  enabled: boolean; // default true
}

export interface OAuthConfig {
  authorizationUrl: string;
  tokenUrl: string;
  clientId: string;
  userClientId: string;
  scopes: string;
  responseType: string;
  extraParams?: Record<string, string>;
  applyUrl?: string; // URL to apply for credentials
  requiresCustomApp?: boolean; // true: users must provide their own OAuth app credentials
  pkce?: {
    required?: boolean;
    method?: 'S256' | 'plain';
  };
}

export interface Tool {
  toolTmplId: string;
  toolType: number; // 1-Brave Search, etc.
  name: string;
  description: string;
  tags: string[];
  authtags: string[];
  credentials: Credential[];
  toolFuncs: ToolFunction[];
  toolResources: ToolResource[];
  lastUsed: number;
  enabled: boolean;
  toolId: string;
  runningState: number; // 1-running
  category?: 1 | 2 | 3 | 4 | 5;
  customRemoteConfig?: string;
  stdioConfig?: string;
  restApiConfig?: unknown;
  oAuthConfig?: OAuthConfig; // OAuth configuration for templates
  authType?: number; // Authentication type
  applyUrl?: string; // URL to apply for credentials (non-OAuth tools)
  anonymousAccess?: boolean; // Whether anonymous access is enabled for this tool
  anonymousRateLimit?: number; // Rate limit for anonymous access (requests/min per source IP)
}

export enum ServerAuthType {
  ApiKey = 1, // API Key authentication
  GoogleAuth = 2, // Google OAuth authentication
  NotionAuth = 3, // Notion OAuth authentication
  FigmaAuth = 4, // Figma OAuth authentication
  GoogleCalendarAuth = 5, // Google Calendar OAuth authentication
  GithubAuth = 6, // Github OAuth authentication
  ZendeskAuth = 7, // Zendesk OAuth authentication
  CanvasAuth = 8, // Canvas OAuth authentication
  CanvaAuth = 9, // Canva OAuth authentication
}

export interface AccessToken {
  tokenId: string;
  name: string;
  role: number; // 1-owner, 2-admin, 3-member
  notes: string;
  lastUsed: number;
  createAt: number;
  expireAt: number;
  rateLimit: number;
  toolList: Tool[];
}

// Request/Response types for each protocol

// 10001 - Init new proxy server and create owner token
export interface Request10001 {
  common: ApiRequest;
  params: {
    masterPwd: string;
  };
}

export interface Response10001 {
  common: ApiResponse;
  data: {
    accessToken: string;
    proxyName: string;
    status: number; // 1-running, 2-stopped
    role: number; // 1-owner
  };
}

// 10002 - Get proxy server info
export interface Request10002 {
  common: ApiRequest;
  params: {};
}

export interface Response10002 {
  data: {
    proxyId: number;
    proxyName: string;
    status: number; // 1-running, 2-stopped
    createAt: number;
  };
}

// 10003 - Edit proxy server info
export interface Request10003 {
  common: ApiRequest;
  params: {
    handleType: number; // 1-edit base info, 2-start server, 3-stop server, 4-backup to cloud, 5-backup to local, 6-restore from cloud, 7-restore from local file
    proxyId: number;
    proxyName: string;
  };
}

export interface Response10003 {
  data: {};
}

// 10004 - Get tools template
export interface Request10004 {
  common: ApiRequest;
  params: {};
}

export interface Response10004 {
  data: {
    toolTmplList: Tool[];
  };
}

// 10005 - Operate tool
export interface Request10005 {
  common: ApiRequest;
  params: {
    handleType: number; // 1-add, 2-edit, 3-enable, 4-disable, 5-delete
    proxyId: string;
    toolId: string; // unique id
    toolType: number; // 1-GitHub ...
    authConf: Credential[];
    functions: ToolFunction[];
    resources: ToolResource[];
    masterPwd: string;
    category?: 1 | 2 | 3 | 4 | 5;
    customRemoteConfig?: {
      url: string;
      headers?: Record<string, string>;
    };
    stdioConfig?: {
      command: string;
      args?: string[];
      env?: Record<string, string>;
      cwd?: string;
    };
    restApiConfig?: string;
  };
}

export interface Response10005 {
  data: {
    toolId: string;
  };
}

// 10006 - Get tool list [for Tool Configuration]
export interface Request10006 {
  common: ApiRequest;
  params: {
    proxyId: number;
    handleType: number; // 1-all, 2-enable=true
  };
}

export interface Response10006 {
  data: {
    toolList: Tool[];
  };
}

// 10007 - Get access tokens
export interface Request10007 {
  common: ApiRequest;
  params: {
    proxyId: number;
  };
}

export interface Response10007 {
  data: {
    tokenList: AccessToken[];
  };
}

// 10008 - Operate access token
export interface Request10008 {
  common: ApiRequest;
  params: {
    handleType: number; // 1-add, 2-edit, 3-delete
    userid: string;
    name: string;
    role: number; // 2-admin, 3-member
    expireAt: number;
    rateLimit: number;
    notes: string;
    permissions: Tool[];
    masterPwd: string;
    proxyId: number;
  };
}

export interface Response10008 {
  data: {
    accessToken: string;
  };
}

// 10009 - Get scopes
export interface Request10009 {
  common: ApiRequest;
  params: {
    proxyId: number;
  };
}

export interface Response10009 {
  data: {
    scopes: Tool[];
  };
}

// Utility types for tool configuration
export type ToolOperationType = 1 | 2 | 3 | 4 | 5; // add, edit, enable, disable, delete
export type AccessTokenOperationType = 1 | 2 | 3; // add, edit, delete
export type ServerOperationType = 1 | 2 | 3 | 4 | 5 | 6 | 7; // edit base info, start, stop, backup cloud, backup local, restore cloud, restore local
export type UserRole = 1 | 2 | 3; // owner, admin, member
export type ServerStatus = 1 | 2; // running, stopped
export type ToolRunningState = 1; // running
