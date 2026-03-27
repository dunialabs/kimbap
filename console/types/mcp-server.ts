import { ServerAuthType } from './api';

export type ServerCategory = 1 | 2 | 3 | 4 | 5;

export interface ServerMetadata {
  type: string;
  name: string;
  icon: string;
  color: string;
  bgColor: string;
  description: string;
}

export interface MCPFunction {
  id: string;
  name: string;
  description: string;
  category: string;
}

export interface MCPServerConfig {
  id: string;
  type: string;
  name: string;
  category?: ServerCategory;
  toolTmplId?: string;
  credentials: Record<string, any>;
  enabledFunctions: string[];
  authType: ServerAuthType;
  allowUserInput: number;
  allFunctions?: Array<{
    funcName: string;
    enabled: boolean;
  }>;
  dataPermissions: {
    type: string;
    allowedResources: string[];
  };
  allResources?: Array<{
    uri: string;
    enabled: boolean;
  }>;
  status: 'connected' | 'connect_failed' | 'not_configured' | 'connecting' | 'error' | 'sleeping';
  lastValidated?: string;
  enabled?: boolean;
  lazyStartEnabled?: boolean;
  anonymousAccess?: boolean;
  anonymousRateLimit?: number;
}

export type ServerType =
  | 'github'
  | 'notion'
  | 'figma'
  | 'postgresql'
  | 'googledrive'
  | 'sequential-thinking'
  | 'wcgw'
  | 'slack'
  | 'aws'
  | 'mysql'
  | 'redis'
  | 'stripe'
  | 'linear'
  | 'openai'
  | 'elasticsearch'
  | 'salesforce'
  | 'mongodb'
  | 'brave-search'
  | 'googlecalendar';
