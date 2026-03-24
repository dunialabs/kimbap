import { Buffer } from 'buffer';

import type { APIDefinition, ToolDefinition } from './rest-api-utils';
import { setValueAtPath } from './rest-api-utils';

const DEFAULT_TIMEOUT_MS = 15000;
const ALWAYS_MASKED_HEADERS = [
  'authorization',
  'proxy-authorization',
  'x-api-key',
  'api-key',
  'apikey',
  'x-api-token',
  'x-access-token'
];

export interface PreparedRequest {
  url: URL;
  headers: Record<string, string>;
  maskedUrl: string;
  maskedHeaders: Record<string, string>;
  timeoutMs: number;
  body?: string;
}

export interface PreparedToolRequest extends PreparedRequest {
  method: ToolDefinition['method'];
}

export interface ToolRequestBuildResult {
  request?: PreparedToolRequest;
  missingParams: string[];
  invalidParams: string[];
  pathWarnings: string[];
}

export function prepareConnectionRequest(apiConfig: APIDefinition): PreparedRequest {
  const base = buildBaseRequest(apiConfig);
  return {
    url: base.url,
    headers: base.headers,
    timeoutMs: base.timeoutMs,
    maskedUrl: maskUrl(base.url, base.sensitiveQueryParams),
    maskedHeaders: maskHeaders(base.headers, base.sensitiveHeaders),
    body: undefined
  };
}

export function prepareToolRequest(
  apiConfig: APIDefinition,
  tool: ToolDefinition,
  params: Record<string, any> = {}
): ToolRequestBuildResult {
  const missingParams: string[] = [];
  const invalidParams: string[] = [];
  const pathWarnings: string[] = [];

  const pathAssignments: Array<{ token: string; value: string }> = [];
  const queryAssignments: Array<{ key: string; value: string }> = [];
  const headerAssignments: Array<{ key: string; value: string }> = [];
  const bodyAssignments: Array<{ path: string; value: any }> = [];

  for (let i = 0; i < tool.parameters.length; i++) {
    const param = tool.parameters[i];
    const mapping = param.mapping || param.name;
    const provided =
      params[param.name] ??
      params[mapping] ??
      params[param.name.replace(/[^A-Za-z0-9_]/g, '')];

    let value = provided;
    if ((value === undefined || value === null || value === '') && param.default !== undefined) {
      value = param.default;
    }

    if ((value === undefined || value === null || value === '') && param.required) {
      missingParams.push(param.name);
      continue;
    }

    if (value === undefined || value === null || value === '') {
      continue;
    }

    let coerced: any;
    try {
      coerced = coerceParamValue(param, value);
    } catch (error: any) {
      invalidParams.push(`${param.name}: ${error?.message || 'Invalid value'}`);
      continue;
    }

    if (param.location === 'path') {
      pathAssignments.push({ token: mapping, value: String(coerced) });
      continue;
    }

    if (param.location === 'query') {
      queryAssignments.push({ key: mapping, value: convertValueToString(coerced) });
      continue;
    }

    if (param.location === 'header') {
      headerAssignments.push({ key: mapping, value: convertValueToString(coerced) });
      continue;
    }

    bodyAssignments.push({ path: mapping, value: coerced });
  }

  if (missingParams.length > 0 || invalidParams.length > 0) {
    return { missingParams, invalidParams, pathWarnings };
  }

  let resolvedEndpoint = tool.endpoint;
  for (let i = 0; i < pathAssignments.length; i++) {
    const assignment = pathAssignments[i];
    const placeholder = `{${assignment.token}}`;
    if (!resolvedEndpoint.includes(placeholder)) {
      pathWarnings.push(
        `Path parameter "${assignment.token}" was provided but no placeholder was found in the endpoint.`
      );
    }
    resolvedEndpoint = resolvedEndpoint.split(placeholder).join(encodeURIComponent(assignment.value));
  }

  const base = buildBaseRequest(apiConfig, resolvedEndpoint);
  const headers = mergeHeaders(base.headers, tool.headers);

  for (let i = 0; i < headerAssignments.length; i++) {
    const assignment = headerAssignments[i];
    headers[assignment.key] = assignment.value;
  }

  for (let i = 0; i < queryAssignments.length; i++) {
    const assignment = queryAssignments[i];
    base.url.searchParams.set(assignment.key, assignment.value);
  }

  let body: string | undefined;
  if (bodyAssignments.length > 0) {
    let payload: any = {};
    for (let i = 0; i < bodyAssignments.length; i++) {
      const assignment = bodyAssignments[i];
      payload = setValueAtPath(payload, assignment.path, assignment.value);
    }
    body = JSON.stringify(payload);
    ensureJsonContentType(headers);
  }

  const request: PreparedToolRequest = {
    url: base.url,
    headers,
    body,
    timeoutMs: tool.timeout ?? base.timeoutMs,
    maskedUrl: maskUrl(base.url, base.sensitiveQueryParams),
    maskedHeaders: maskHeaders(headers, base.sensitiveHeaders),
    method: tool.method
  };

  return {
    request,
    missingParams,
    invalidParams,
    pathWarnings
  };
}

export async function executeHttpRequest(
  request: PreparedRequest,
  method: string
): Promise<{ response: Response; durationMs: number }> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), request.timeoutMs || DEFAULT_TIMEOUT_MS);
  const startedAt = Date.now();

  try {
    const response = await fetch(request.url.toString(), {
      method,
      headers: request.headers,
      body: shouldSendBody(method) ? request.body : undefined,
      cache: 'no-store',
      signal: controller.signal
    });
    const durationMs = Date.now() - startedAt;
    return { response, durationMs };
  } catch (error: any) {
    if (error?.name === 'AbortError') {
      throw new Error(`Request timed out after ${request.timeoutMs || DEFAULT_TIMEOUT_MS} ms`);
    }
    throw error;
  } finally {
    clearTimeout(timeout);
  }
}

function shouldSendBody(method: string): boolean {
  const upper = method.toUpperCase();
  return upper !== 'GET' && upper !== 'HEAD';
}

function buildBaseRequest(apiConfig: APIDefinition, endpoint?: string) {
  const url = resolveEndpointUrl(apiConfig.baseUrl, endpoint);
  const headers = mergeHeaders(apiConfig.headers);
  const timeoutMs = apiConfig.timeout ?? DEFAULT_TIMEOUT_MS;

  const authResult = applyAuthToRequest(apiConfig, url, headers);

  return {
    url,
    headers,
    timeoutMs,
    sensitiveHeaders: authResult.sensitiveHeaders,
    sensitiveQueryParams: authResult.sensitiveQueryParams
  };
}

function mergeHeaders(...sources: Array<Record<string, string> | undefined>): Record<string, string> {
  const merged: Record<string, string> = {};
  for (let i = 0; i < sources.length; i++) {
    const source = sources[i];
    if (!source) continue;
    const keys = Object.keys(source);
    for (let j = 0; j < keys.length; j++) {
      const key = keys[j];
      const value = source[key];
      if (value !== undefined && value !== null) {
        merged[key] = value;
      }
    }
  }
  return merged;
}

function ensureJsonContentType(headers: Record<string, string>) {
  const hasContentType = Object.keys(headers).some((key) => key.toLowerCase() === 'content-type');
  if (!hasContentType) {
    headers['Content-Type'] = 'application/json';
  }
}

function applyAuthToRequest(
  apiConfig: APIDefinition,
  url: URL,
  headers: Record<string, string>
) {
  const sensitiveHeaders: Set<string> = new Set();
  const sensitiveQueryParams: Set<string> = new Set();

  const auth = apiConfig.auth;
  if (!auth || auth.type === 'none') {
    return {
      sensitiveHeaders: Array.from(sensitiveHeaders),
      sensitiveQueryParams: Array.from(sensitiveQueryParams)
    };
  }

  switch (auth.type) {
    case 'bearer': {
      if (!auth.value?.trim()) {
        throw new Error('Bearer token is required for authentication.');
      }
      headers['Authorization'] = `Bearer ${auth.value.trim()}`;
      sensitiveHeaders.add('authorization');
      break;
    }
    case 'header': {
      if (!auth.header?.trim()) {
        throw new Error('Authentication header name is required.');
      }
      if (!auth.value?.trim()) {
        throw new Error('Authentication header value is required.');
      }
      headers[auth.header] = auth.value.trim();
      sensitiveHeaders.add(auth.header.toLowerCase());
      break;
    }
    case 'query_param': {
      if (!auth.param?.trim()) {
        throw new Error('Authentication query parameter name is required.');
      }
      if (!auth.value?.trim()) {
        throw new Error('Authentication query parameter value is required.');
      }
      url.searchParams.set(auth.param, auth.value.trim());
      sensitiveQueryParams.add(auth.param);
      break;
    }
    case 'basic': {
      if (!auth.username?.trim() || !auth.password?.trim()) {
        throw new Error('Basic authentication requires both username and password.');
      }
      const encoded = Buffer.from(`${auth.username}:${auth.password}`).toString('base64');
      headers['Authorization'] = `Basic ${encoded}`;
      sensitiveHeaders.add('authorization');
      break;
    }
    default:
      break;
  }

  return {
    sensitiveHeaders: Array.from(sensitiveHeaders),
    sensitiveQueryParams: Array.from(sensitiveQueryParams)
  };
}

function maskHeaders(headers: Record<string, string>, sensitiveKeys: string[]): Record<string, string> {
  const masked: Record<string, string> = {};
  const maskSet = new Set(ALWAYS_MASKED_HEADERS.map((h) => h.toLowerCase()));
  for (let i = 0; i < sensitiveKeys.length; i++) {
    maskSet.add(sensitiveKeys[i].toLowerCase());
  }

  const keys = Object.keys(headers);
  for (let i = 0; i < keys.length; i++) {
    const key = keys[i];
    if (maskSet.has(key.toLowerCase())) {
      masked[key] = '••••••';
    } else {
      masked[key] = headers[key];
    }
  }
  return masked;
}

function maskUrl(url: URL, sensitiveParams: string[]): string {
  const clone = new URL(url.toString());
  for (let i = 0; i < sensitiveParams.length; i++) {
    const param = sensitiveParams[i];
    if (clone.searchParams.has(param)) {
      clone.searchParams.set(param, '••••••');
    }
  }
  return clone.toString();
}

function resolveEndpointUrl(baseUrl: string, endpoint?: string): URL {
  if (!endpoint || !endpoint.trim()) {
    return new URL(baseUrl);
  }
  const trimmed = endpoint.trim();
  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return new URL(trimmed);
  }
  if (!trimmed.startsWith('/')) {
    return new URL(`/${trimmed}`, baseUrl);
  }
  return new URL(trimmed, baseUrl);
}

function coerceParamValue(param: ToolDefinition['parameters'][number], value: any): any {
  switch (param.type) {
    case 'number': {
      const num = typeof value === 'number' ? value : Number(value);
      if (Number.isNaN(num)) {
        throw new Error('Expected a numeric value');
      }
      return num;
    }
    case 'boolean': {
      if (typeof value === 'boolean') return value;
      if (typeof value === 'string') {
        const normalized = value.toLowerCase();
        if (normalized === 'true' || normalized === '1') return true;
        if (normalized === 'false' || normalized === '0') return false;
      }
      throw new Error('Expected a boolean value (true/false)');
    }
    case 'object':
    case 'array': {
      if (typeof value === 'object') {
        return value;
      }
      try {
        return JSON.parse(value);
      } catch {
        throw new Error('Expected JSON-formatted value');
      }
    }
    default:
      return String(value);
  }
}

function convertValueToString(value: any): string {
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  return JSON.stringify(value);
}
