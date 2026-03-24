/**
 * Kimbap Cloud API Service
 * 封装对 Kimbap Cloud API 的调用
 */

import axios, { AxiosInstance } from 'axios';
import { config } from './config';
import { OAuthConfig } from '@/types/api';
import { readTemplatesFromFile, compareAndUpdateTemplates } from './tool-template-storage';

export interface TunnelCreateParams {
  appId: string;
}

export interface TunnelCredentials {
  AccountTag: string;
  TunnelSecret: string;
  TunnelID: string;
}

export interface TunnelCreateResponse {
  tunnelId: string;
  subdomain: string;
  credentials: TunnelCredentials;
}

export interface TunnelInfo {
  tunnelId: string;
  name: string;
  subdomain: string | null;
  credentials: TunnelCredentials;
  created_at: string;
}

export interface TunnelDeleteResponse {
  success: boolean;
  message: string;
  tunnelId: string;
  deletedDnsRecords: string[];
}

export interface ToolTemplate {
  toolTmplId: string;
  toolType: number;
  name: string;
  description: string;
  tags: string[];
  authtags: string[];
  applyUrl?: string;
  credentials: Array<{
    name: string;
    description: string;
    dataType: number;
    key: string;
  }>;
  mcpJsonConf: any; // Can be any structure - could have command/args, url/headers, or other fields
  authType: number;
  toolDefaultConfig?: string;
  oAuthConfig?: OAuthConfig;
}

export class KimbapCloudApiService {
  private readonly apiClient: AxiosInstance;

  constructor() {
    this.apiClient = axios.create({
      baseURL: config.cloudApi.baseUrl,
      headers: {
        'Content-Type': 'application/json'
      },
      timeout: 15000 // 15 second timeout for tunnel operations
    });
  }

  /**
   * 创建 Cloudflare Tunnel
   * @param params 创建参数
   * @returns Tunnel 创建响应
   */
  async createTunnel(params: TunnelCreateParams): Promise<TunnelCreateResponse> {
    try {
      const response = await this.apiClient.post<TunnelCreateResponse>(
        config.cloudApi.endpoints.tunnelCreate, 
        params
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.error || 'Failed to create tunnel');
    }
  }

  /**
   * 获取 Tunnel 凭据
   * @param tunnelId Tunnel ID
   * @returns Tunnel 信息
   */
  async getTunnelCredentials(tunnelId: string): Promise<TunnelInfo> {
    try {
      const response = await this.apiClient.post<TunnelInfo>(
        config.cloudApi.endpoints.tunnelCredentials,
        { tunnelId }
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.error || 'Failed to get tunnel credentials');
    }
  }

  /**
   * 删除 Tunnel
   * @param tunnelId Tunnel ID
   * @returns 删除响应
   */
  async deleteTunnel(tunnelId: string): Promise<TunnelDeleteResponse> {
    try {
      const response = await this.apiClient.post<TunnelDeleteResponse>(
        config.cloudApi.endpoints.tunnelDelete,
        { tunnelId }
      );
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.error || 'Failed to delete tunnel');
    }
  }

  /**
   * Fetches tool templates from the cloud API with retry logic
   * @returns Array of tool templates with parsed JSON fields
   * @throws Error if the API call fails after retries
   */
  async fetchToolTemplates(): Promise<ToolTemplate[]> {
    const maxRetries = 2;
    let lastError: any;

    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        const response = await this.apiClient.post(
          config.cloudApi.endpoints.toolTemplates,
          {},
          { timeout: 8000 } // 8 second timeout for tool templates
        );
        
        // Success - process the response
        const cloudData = response.data;
        // Ensure we have an array
        const dataArray = Array.isArray(cloudData) ? cloudData : [cloudData];
        // Map and filter only the required fields
        const templates: ToolTemplate[] = dataArray.map((item: any) => ({
          toolTmplId: item.toolId || '',
          toolType: item.toolType || 0,
          name: item.name || '',
          description: item.description || '',
          applyUrl: item.applyUrl || undefined,
          // Parse JSON strings to proper types
          tags: typeof item.tags === 'string' ? JSON.parse(item.tags) : (item.tags || []),
          authtags: typeof item.authtags === 'string' ? JSON.parse(item.authtags) : (item.authtags || []),
          credentials: typeof item.credentials === 'string' ?
            (item.credentials ? JSON.parse(item.credentials) : []) :
            (item.credentials || []),
          mcpJsonConf: typeof item.mcpJsonConf === 'string' ?
            JSON.parse(item.mcpJsonConf) :
            (item.mcpJsonConf || {}), // Return empty object as default, let the consumer handle the structure
          authType: item.authType || 0,
          toolDefaultConfig: item.toolDefaultConfig || '',
          oAuthConfig: item.oAuthConfig
            ? typeof item.oAuthConfig === 'string' ? JSON.parse(item.oAuthConfig) : item.oAuthConfig
            : undefined
        }));

        // 对比并更新本地文件
        const updated = compareAndUpdateTemplates(templates);
        if (updated) {
          console.log('Local tool templates file updated from cloud');
        }

        return templates;
      } catch (error: any) {
        lastError = error;
        
        // Log retry attempts for 502 errors
        if (error.response?.status === 502 && attempt < maxRetries) {
          console.log(`Cloud API gateway error (502), retrying... (attempt ${attempt}/${maxRetries})`);
          // Small delay before retry
          await new Promise(resolve => setTimeout(resolve, 1000));
          continue;
        }
        
        // For other errors or last retry, log and break
        console.error(`Failed to fetch tool templates (attempt ${attempt}/${maxRetries}):`, error);
        break;
      }
    }
    
    // Log specific error types
    if (lastError?.response?.status === 502) {
      console.error('Cloud API gateway error (502) - service may be temporarily unavailable after retries');
    } else if (lastError?.code === 'ECONNABORTED') {
      console.error('Request timeout - cloud API did not respond in time');
    } else if (lastError?.code === 'ECONNRESET' || lastError?.code === 'ECONNREFUSED') {
      console.error('Network connection error - unable to reach cloud API');
    }
    
    // Throw the error to let the caller handle it
    throw lastError;
  }

  /**
   * Fetches tool templates with error handling and local file fallback
   * @returns Array of tool templates, uses local file if cloud API fails
   */
  async fetchToolTemplatesWithFallback(): Promise<ToolTemplate[]> {
    try {
      return await this.fetchToolTemplates();
    } catch (error) {
      console.log('Failed to fetch tool templates from cloud, trying local file');

      // 尝试从本地文件读取
      const localTemplates = readTemplatesFromFile();
      if (localTemplates) {
        console.log('Using local tool templates as fallback');
        return localTemplates;
      }

      console.log('No local templates available, returning empty array');
      return [];
    }
  }
}
