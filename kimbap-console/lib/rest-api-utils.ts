import { z } from 'zod';

/**
 * REST API Configuration Utilities
 * Provides validation, conversion, and parsing utilities for REST API tool configuration
 */

const httpMethodEnum = z.enum(['GET', 'POST', 'PUT', 'DELETE', 'PATCH']);
const parameterTypeEnum = z.enum(['string', 'number', 'boolean', 'object', 'array']);
const parameterLocationEnum = z.enum(['query', 'body', 'path', 'header']);
const authTypeEnum = z.enum(['bearer', 'query_param', 'header', 'basic', 'none']);
const headersRecordSchema = z.record(z.string().min(1), z.string());

export const ResponseTransformSchema = z.object({
  type: z.enum(['json', 'text', 'raw']),
  jsonPath: z.string().optional(),
  template: z.string().optional(),
  errorPath: z.string().optional()
});

export const ParameterDefinitionSchema = z.object({
  name: z.string().min(1, 'Parameter name is required'),
  description: z.string().min(1, 'Parameter description is required'),
  type: parameterTypeEnum,
  required: z.boolean(),
  default: z.any().optional(),
  location: parameterLocationEnum,
  mapping: z.string().optional(),
  enum: z.array(z.any()).optional(),
  pattern: z.string().optional()
});

export const ToolDefinitionSchema = z
  .object({
    name: z.string().min(1, 'Tool name is required'),
    description: z.string().min(1, 'Tool description is required'),
    endpoint: z
      .string()
      .min(1, 'Endpoint is required')
      .refine((value) => value.startsWith('/'), 'Endpoint must start with "/"'),
    method: httpMethodEnum,
    parameters: z.array(ParameterDefinitionSchema).max(50, 'Too many parameters (max 50)'),
    response: ResponseTransformSchema.optional(),
    headers: headersRecordSchema.optional(),
    timeout: z.number().int().positive().max(300000).optional()
  })
  .superRefine((tool, ctx) => {
    const regexp = /\{([^}]+)\}/g;
    const matches = tool.endpoint.match(regexp);
    if (!matches) {
      return;
    }
    const pathParams = matches.map((match) => match.slice(1, -1));
    for (let i = 0; i < pathParams.length; i++) {
      const token = pathParams[i];
      const hasParam = tool.parameters.some((param) => {
        if (param.location !== 'path') return false;
        const name = param.name;
        const mapping = param.mapping || param.name;
        return token === name || token === mapping;
      });
      if (!hasParam) {
        ctx.addIssue({
          code: 'custom',
          path: ['parameters'],
          message: `Endpoint expects path parameter "${token}" but no matching path parameter was defined`
        });
      }
    }
  });

export const AuthConfigSchema = z
  .object({
    type: authTypeEnum,
    param: z.string().optional(),
    header: z.string().optional(),
    value: z.string().optional(),
    username: z.string().optional(),
    password: z.string().optional()
  })
  .superRefine((auth, ctx) => {
    if (auth.type === 'bearer' && !auth.value?.trim()) {
      ctx.addIssue({ code: 'custom', path: ['value'], message: 'Bearer auth requires value' });
    }
    if (auth.type === 'header') {
      if (!auth.header?.trim()) {
        ctx.addIssue({ code: 'custom', path: ['header'], message: 'Header auth requires header name' });
      }
      if (!auth.value?.trim()) {
        ctx.addIssue({ code: 'custom', path: ['value'], message: 'Header auth requires value' });
      }
    }
    if (auth.type === 'query_param') {
      if (!auth.param?.trim()) {
        ctx.addIssue({ code: 'custom', path: ['param'], message: 'Query param auth requires parameter name' });
      }
      if (!auth.value?.trim()) {
        ctx.addIssue({ code: 'custom', path: ['value'], message: 'Query param auth requires value' });
      }
    }
    if (auth.type === 'basic') {
      if (!auth.username?.trim()) {
        ctx.addIssue({ code: 'custom', path: ['username'], message: 'Basic auth requires username' });
      }
      if (!auth.password?.trim()) {
        ctx.addIssue({ code: 'custom', path: ['password'], message: 'Basic auth requires password' });
      }
    }
  });

export const ApiDefinitionSchema = z
  .object({
    name: z.string().min(1, 'API name is required'),
    description: z.string().min(1, 'API description is required'),
    baseUrl: z.string().min(1, 'Base URL is required'),
    auth: AuthConfigSchema.optional(),
    tools: z.array(ToolDefinitionSchema).min(1, 'At least one tool must be defined').max(50, 'Too many tools (max 50)'),
    headers: headersRecordSchema.optional(),
    timeout: z.number().int().positive().max(300000).optional()
  })
  .superRefine((api, ctx) => {
    const issue = validateHttpsBaseUrl(api.baseUrl);
    if (issue) {
      ctx.addIssue({ code: 'custom', path: ['baseUrl'], message: issue });
    }
  });

export type ResponseTransform = z.infer<typeof ResponseTransformSchema>;
export type ParameterDefinition = z.infer<typeof ParameterDefinitionSchema>;
export type ToolDefinition = z.infer<typeof ToolDefinitionSchema>;
export type AuthConfig = z.infer<typeof AuthConfigSchema>;
export type APIDefinition = z.infer<typeof ApiDefinitionSchema>;

export type RestApiValidationStatus = 'pass' | 'warn' | 'error';

export interface RestApiValidationCheck {
  id: string;
  label: string;
  status: RestApiValidationStatus;
  message: string;
  details?: string[];
}

export type RestApiValidationSummary = 'idle' | 'valid' | 'warning' | 'error';

export interface RestApiValidationReport {
  format: 'json' | 'yaml' | 'unknown';
  openApiDetected: boolean;
  isJsonValid: boolean;
  isSchemaValid: boolean;
  summary: RestApiValidationSummary;
  checks: RestApiValidationCheck[];
  warnings: string[];
  errors: string[];
  environmentPlaceholders: string[];
  parsedConfig?: APIDefinition;
  toolCount?: number;
  rawConfigSize?: number;
  checkedAt: number;
}

/**
 * Detect and parse input string as JSON or YAML format
 */
export type DetectFormatResult =
  | { format: 'json'; data: any }
  | { format: 'yaml'; data: any }
  | { format: 'unknown'; data: null; error?: string };

export async function detectFormat(input: string): Promise<DetectFormatResult> {
  const text = input.trim();
  const errors: string[] = [];

  // First try JSON
  if (text.startsWith('{') || text.startsWith('[')) {
    try {
      const parsed = JSON.parse(text);
      return { format: 'json', data: parsed };
    } catch (e) {
      errors.push(`JSON parse failed: ${e instanceof Error ? e.message : 'Unknown error'}`);
    }
  }

  // Then try YAML
  try {
    // Dynamic import to avoid bundling issues
    const yaml = await import('js-yaml');
    const loaded = yaml.load(text);
    return { format: 'yaml', data: loaded };
  } catch (e) {
    errors.push(
      `YAML parse failed: ${
        e instanceof Error ? e.message : 'Unknown error'
      }`
    );
  }

  return { format: 'unknown', data: null, error: errors.join(' | ') };
}

/**
 * Parse YAML string to JSON object
 * Note: Requires js-yaml library
 */
export async function parseYamlToJson(yamlString: string): Promise<any> {
  try {
    // Dynamic import to avoid bundling issues
    const yaml = await import('js-yaml');
    return yaml.load(yamlString);
  } catch (error) {
    throw new Error(`Failed to parse YAML: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }
}

/**
 * Validate REST API configuration structure
 */
export function validateRestApiConfig(config: any): { valid: boolean; error?: string } {
  const result = ApiDefinitionSchema.safeParse(config);
  if (result.success) {
    return { valid: true };
  }

  const issue = result.error.issues[0];
  const path = issue?.path?.length ? `${issue.path.join('.')}: ` : '';
  const message = issue?.message || 'Invalid REST API configuration';
  return { valid: false, error: `${path}${message}` };
}

/**
 * Ensure baseUrl is HTTPS with valid hostname
 */
export function validateHttpsBaseUrl(baseUrl: string): string | null {
  try {
    const url = new URL(baseUrl);
    if (url.protocol !== 'https:') {
      return 'Base URL must use HTTPS (https://)';
    }
    if (!url.hostname) {
      return 'Base URL must include a valid hostname';
    }
    return null;
  } catch {
    return 'Base URL must be a valid HTTPS URL';
  }
}

/**
 * Generate a detailed validation report for a raw REST API configuration string
 */
export async function generateRestApiValidationReport(rawInput: string): Promise<RestApiValidationReport> {
  const trimmed = rawInput.trim();
  const rawConfigSize = new TextEncoder().encode(rawInput || '').length;
  const report: RestApiValidationReport = {
    format: 'unknown',
    openApiDetected: false,
    isJsonValid: false,
    isSchemaValid: false,
    summary: 'idle',
    checks: [],
    warnings: [],
    errors: [],
    environmentPlaceholders: [],
    rawConfigSize,
    checkedAt: Date.now()
  };

  if (!trimmed) {
    const message = 'Configuration content is required';
    report.errors.push(message);
    report.checks.push({
      id: 'empty-input',
      label: 'Configuration Input',
      status: 'error',
      message
    });
    report.summary = 'error';
    return report;
  }

  const detected = await detectFormat(trimmed);
  report.format = detected.format;

  if (detected.format === 'unknown' || !detected.data) {
    const message = detected.format === 'unknown'
      ? detected.error || 'Unable to parse configuration as JSON or YAML'
      : 'Unable to parse configuration';
    report.errors.push(message);
    report.checks.push({
      id: 'format',
      label: 'Format Detection',
      status: 'error',
      message
    });
    report.summary = 'error';
    return report;
  }

  report.isJsonValid = true;
  report.checks.push({
    id: 'format',
    label: detected.format === 'yaml' ? 'YAML Format' : 'JSON Format',
    status: 'pass',
    message: detected.format === 'yaml' ? 'Parsed YAML successfully' : 'Parsed JSON successfully'
  });

  if (isOpenApiFormat(detected.data)) {
    report.openApiDetected = true;
    const message = 'Detected OpenAPI specification. Convert it to the REST API configuration format before saving or testing.';
    report.warnings.push(message);
    report.checks.push({
      id: 'openapi-detected',
      label: 'OpenAPI Specification',
      status: 'warn',
      message
    });
    report.summary = 'warning';
    return report;
  }

  const schemaResult = ApiDefinitionSchema.safeParse(detected.data);
  if (!schemaResult.success) {
    const issue = schemaResult.error.issues[0];
    const path = issue?.path?.length ? `${issue.path.join('.')}: ` : '';
    const message = `${path}${issue?.message || 'Invalid REST API configuration'}`;
    report.errors.push(message);
    report.checks.push({
      id: 'schema',
      label: 'Schema Validation',
      status: 'error',
      message
    });
    report.summary = 'error';
    return report;
  }

  const parsedConfig = schemaResult.data;
  report.parsedConfig = parsedConfig;
  report.isSchemaValid = true;
  report.toolCount = parsedConfig.tools.length;
  report.checks.push({
    id: 'schema',
    label: 'Schema Validation',
    status: 'pass',
    message: 'Configuration matches the required schema'
  });

  const baseUrlIssue = validateHttpsBaseUrl(parsedConfig.baseUrl);
  report.checks.push({
    id: 'base-url',
    label: 'Base URL',
    status: baseUrlIssue ? 'error' : 'pass',
    message: baseUrlIssue || 'Base URL uses HTTPS and is valid'
  });
  if (baseUrlIssue) {
    report.errors.push(baseUrlIssue);
    report.summary = 'error';
    return report;
  }

  const maxBytes = 100 * 1024;
  if (rawConfigSize > maxBytes) {
    const message = `Configuration is too large (${Math.ceil(rawConfigSize / 1024)} KB). Maximum allowed size is 100 KB.`;
    report.errors.push(message);
    report.checks.push({
      id: 'config-size',
      label: 'Configuration Size',
      status: 'error',
      message
    });
    report.summary = 'error';
    return report;
  }

  report.checks.push({
    id: 'config-size',
    label: 'Configuration Size',
    status: 'pass',
    message: `Configuration size is ${(rawConfigSize / 1024).toFixed(1)} KB (limit 100 KB)`
  });

  report.checks.push({
    id: 'tool-count',
    label: 'Tool Count',
    status: parsedConfig.tools.length > 0 ? 'pass' : 'error',
    message: `${parsedConfig.tools.length} tool${parsedConfig.tools.length === 1 ? '' : 's'} configured`
  });

  const envPlaceholders = checkEnvironmentVariables(parsedConfig);
  report.environmentPlaceholders = envPlaceholders;
  if (envPlaceholders.length > 0) {
    const message = `Detected environment placeholders: ${envPlaceholders.join(', ')}`;
    report.warnings.push(message);
    report.checks.push({
      id: 'env-placeholders',
      label: 'Credential Placeholders',
      status: 'warn',
      message,
      details: [
        'Replace placeholders like ${API_KEY} with actual credentials before saving.',
        'Alternatively, document the required environment variables for your deployment.'
      ]
    });
  } else {
    report.checks.push({
      id: 'env-placeholders',
      label: 'Credential Placeholders',
      status: 'pass',
      message: 'No environment variable placeholders detected'
    });
  }

  report.summary = deriveValidationSummary(report);
  return report;
}

function deriveValidationSummary(report: RestApiValidationReport): RestApiValidationSummary {
  const hasErrors = report.errors.length > 0 || report.checks.some((check) => check.status === 'error');
  if (hasErrors) {
    return 'error';
  }

  const hasWarnings = report.warnings.length > 0 || report.checks.some((check) => check.status === 'warn');
  if (hasWarnings) {
    return 'warning';
  }

  return 'valid';
}

/**
 * Check for environment variable placeholders in configuration
 */
export function checkEnvironmentVariables(config: any): string[] {
  if (!config || typeof config !== 'object') {
    return [];
  }

  const envVars: Set<string> = new Set();
  const envVarPattern = /\$\{([A-Za-z_][A-Za-z0-9_]*)\}/g;

  const searchObject = (obj: any) => {
    if (typeof obj === 'string') {
      envVarPattern.lastIndex = 0;
      let match: RegExpExecArray | null;
      while ((match = envVarPattern.exec(obj)) !== null) {
        envVars.add(match[1]);
      }
      return;
    }

    if (Array.isArray(obj)) {
      for (let i = 0; i < obj.length; i++) {
        searchObject(obj[i]);
      }
      return;
    }

    if (obj && typeof obj === 'object') {
      const keys = Object.keys(obj);
      for (let i = 0; i < keys.length; i++) {
        const key = keys[i];
        searchObject(obj[key]);
      }
    }
  };

  searchObject(config);
  return Array.from(envVars);
}

type PathToken = string | number;

function tokenizePath(path: string, options?: { allowJsonRoot?: boolean }): PathToken[] {
  if (!path) return [];
  let working = path.trim();

  if (options?.allowJsonRoot && working.startsWith('$')) {
    working = working.slice(1);
  }

  if (working.startsWith('.')) {
    working = working.slice(1);
  }

  const tokens: PathToken[] = [];
  const regex = /([^[.\]]+)|\[(\d+)\]/g;
  let match: RegExpExecArray | null;
  while ((match = regex.exec(working)) !== null) {
    if (match[1]) {
      tokens.push(match[1]);
    } else if (match[2]) {
      tokens.push(Number(match[2]));
    }
  }
  return tokens;
}

export function extractValueByJsonPath(data: any, jsonPath?: string): any {
  if (!jsonPath) return data;
  const tokens = tokenizePath(jsonPath, { allowJsonRoot: true });
  if (tokens.length === 0) {
    return data;
  }

  let current: any = data;
  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i];
    if (current == null) {
      return undefined;
    }
    current = current[token as any];
  }
  return current;
}

export function setValueAtPath(target: any, path: string, value: any): any {
  const tokens = tokenizePath(path);
  if (tokens.length === 0) {
    return value;
  }

  const base =
    target && typeof target === 'object'
      ? target
      : typeof tokens[0] === 'number'
        ? []
        : {};

  let current = base;
  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i];
    const isLast = i === tokens.length - 1;

    if (isLast) {
      if (typeof token === 'number') {
        if (!Array.isArray(current)) {
          current[token] = value;
        } else {
          current[token] = value;
        }
      } else {
        current[token] = value;
      }
      return base;
    }

    if (typeof token === 'number') {
      if (!Array.isArray(current[token])) {
        current[token] = typeof tokens[i + 1] === 'number' ? [] : {};
      }
      current = current[token];
    } else {
      if (typeof current[token] !== 'object' || current[token] === null) {
        current[token] = typeof tokens[i + 1] === 'number' ? [] : {};
      }
      current = current[token];
    }
  }
  return base;
}

export function applyResponseTransform(data: any, transform?: ResponseTransform): any {
  if (!transform) {
    return data;
  }

  switch (transform.type) {
    case 'json':
      return transform.jsonPath ? extractValueByJsonPath(data, transform.jsonPath) : data;
    case 'text':
      if (typeof data === 'string') {
        return data;
      }
      return data !== undefined ? JSON.stringify(data, null, 2) : '';
    case 'raw':
    default:
      return data;
  }
}

/**
 * Detect if configuration is OpenAPI format
 */
export function isOpenApiFormat(config: any): boolean {
  return !!(
    config &&
    typeof config === 'object' &&
    (config.openapi || config.swagger) &&
    config.paths &&
    config.info
  );
}

/**
 * Convert OpenAPI specification to REST API format
 */
export function convertOpenApiToRestApiFormat(openApiSpec: any): APIDefinition {
  // Extract basic info
  const name = openApiSpec.info?.title || 'API';
  const description =
    openApiSpec.info?.description || openApiSpec.info?.title || 'Converted from OpenAPI specification';
  const baseUrl = openApiSpec.servers?.[0]?.url || '';

  if (!baseUrl) {
    throw new Error('OpenAPI spec must have at least one server URL');
  }

  // Convert auth
  const auth = convertOpenApiAuth(openApiSpec) ?? { type: 'none' };

  // Convert paths to tools
  const tools: ToolDefinition[] = [];

  if (openApiSpec.paths) {
    const methods = ['get', 'post', 'put', 'delete', 'patch'];
    for (const path in openApiSpec.paths) {
      if (!Object.prototype.hasOwnProperty.call(openApiSpec.paths, path)) continue;
      const pathItem = openApiSpec.paths[path];
      for (let i = 0; i < methods.length; i++) {
        const method = methods[i];
        if (pathItem && pathItem[method]) {
          const operation = pathItem[method];
          const tool = convertOpenApiOperation(path, method, operation, pathItem, openApiSpec);
          tools.push(tool);
        }
      }
    }
  }

  if (tools.length === 0) {
    throw new Error('OpenAPI spec must have at least one operation');
  }

  return {
    name,
    description,
    baseUrl,
    auth,
    tools
  };
}

/**
 * Convert OpenAPI security schemes to auth config
 */
function convertOpenApiAuth(openApiSpec: any): AuthConfig | undefined {
  const securitySchemes = openApiSpec?.components?.securitySchemes;
  if (!securitySchemes || typeof securitySchemes !== 'object') {
    return undefined;
  }

  // Prefer global security selection when present
  let schemeName: string | undefined;
  const security = openApiSpec?.security;
  if (Array.isArray(security) && security.length > 0) {
    const firstReq = security[0];
    if (firstReq && typeof firstReq === 'object' && !Array.isArray(firstReq)) {
      schemeName = Object.keys(firstReq)[0];
    }
  }

  if (!schemeName) {
    schemeName = Object.keys(securitySchemes)[0];
  }

  if (!schemeName) {
    return undefined;
  }

  const scheme = securitySchemes[schemeName];
  if (!scheme || typeof scheme !== 'object') {
    return undefined;
  }

  const valuePlaceholder = `\${${schemeName}}`;
  const usernamePlaceholder = `\${${schemeName}_USERNAME}`;
  const passwordPlaceholder = `\${${schemeName}_PASSWORD}`;

  // Convert based on scheme type
  if (scheme.type === 'apiKey') {
    if (scheme.in === 'header') {
      return {
        type: 'header',
        header: scheme.name,
        value: valuePlaceholder
      };
    }
    if (scheme.in === 'query') {
      return {
        type: 'query_param',
        param: scheme.name,
        value: valuePlaceholder
      };
    }
  }

  if (scheme.type === 'http') {
    if (scheme.scheme === 'bearer') {
      return {
        type: 'bearer',
        value: valuePlaceholder
      };
    }
    if (scheme.scheme === 'basic') {
      return {
        type: 'basic',
        username: usernamePlaceholder,
        password: passwordPlaceholder
      };
    }
  }

  // Not supported in current AuthConfig mapping
  if (scheme.type === 'oauth2' || scheme.type === 'openIdConnect') {
    throw new Error(
      `OpenAPI security scheme "${schemeName}" is "${scheme.type}" and is not supported for automatic conversion. ` +
        `Please configure auth manually.`
    );
  }

  return { type: 'none' };
}

type OpenApiSchemaLike = any;

function getByRef(openApiSpec: any, ref: string): any | undefined {
  if (!ref || typeof ref !== 'string') return undefined;
  if (ref.indexOf('#/') !== 0) return undefined;
  const parts = ref.slice(2).split('/');
  let cur: any = openApiSpec;
  for (let i = 0; i < parts.length; i++) {
    const key = parts[i];
    if (!cur || typeof cur !== 'object' || !(key in cur)) return undefined;
    cur = cur[key];
  }
  return cur;
}

type NormalizedSchema = {
  type?: string;
  description?: string;
  properties?: Record<string, OpenApiSchemaLike>;
  required?: string[];
  enum?: any[];
  default?: any;
  format?: string;
  items?: OpenApiSchemaLike;
  unionSource?: 'oneOf' | 'anyOf';
};

function normalizeSchema(
  openApiSpec: any,
  schema: OpenApiSchemaLike,
  ctx: { seenRefs: Record<string, true> }
): NormalizedSchema {
  if (!schema || typeof schema !== 'object') return {};

  // $ref
  if (schema.$ref && typeof schema.$ref === 'string') {
    const ref = schema.$ref;
    if (ctx.seenRefs[ref]) {
      return { type: 'object', description: 'Cyclic $ref detected' };
    }
    ctx.seenRefs[ref] = true;
    const resolved = getByRef(openApiSpec, ref);
    // Allow siblings with $ref; local keys override
    const merged =
      resolved && typeof resolved === 'object' ? { ...resolved, ...schema } : { ...schema };
    delete (merged as any).$ref;
    return normalizeSchema(openApiSpec, merged, ctx);
  }

  // allOf: merge properties + required (union)
  if (Array.isArray(schema.allOf) && schema.allOf.length > 0) {
    const merged: NormalizedSchema = { type: 'object', properties: {}, required: [] };
    for (let i = 0; i < schema.allOf.length; i++) {
      const part = normalizeSchema(openApiSpec, schema.allOf[i], ctx);
      if (part.description && !merged.description) merged.description = part.description;
      if (part.properties) {
        merged.properties = { ...(merged.properties || {}), ...part.properties };
      }
      if (Array.isArray(part.required)) {
        for (let j = 0; j < part.required.length; j++) {
          const r = part.required[j];
          if ((merged.required as string[]).indexOf(r) === -1) (merged.required as string[]).push(r);
        }
      }
    }
    return merged;
  }

  // oneOf/anyOf: union properties, but required will be forced false during flatten
  const unionKey = Array.isArray(schema.oneOf)
    ? 'oneOf'
    : Array.isArray(schema.anyOf)
      ? 'anyOf'
      : undefined;
  if (unionKey) {
    const arr = schema[unionKey] as any[];
    const merged: NormalizedSchema = {
      type: 'object',
      properties: {},
      required: [],
      unionSource: unionKey as 'oneOf' | 'anyOf'
    };
    for (let i = 0; i < arr.length; i++) {
      const part = normalizeSchema(openApiSpec, arr[i], ctx);
      if (part.description && !merged.description) merged.description = part.description;
      if (part.properties) {
        merged.properties = { ...(merged.properties || {}), ...part.properties };
      }
    }
    return merged;
  }

  const t = schema.type || (schema.properties ? 'object' : undefined);
  return {
    type: t,
    description: schema.description,
    properties: schema.properties,
    required: Array.isArray(schema.required) ? schema.required : undefined,
    enum: schema.enum,
    default: schema.default,
    format: schema.format,
    items: schema.items
  };
}

function mapOpenApiTypeToParamType(openApiType: any): ParameterDefinition['type'] {
  if (openApiType === 'integer' || openApiType === 'number') return 'number';
  if (openApiType === 'boolean') return 'boolean';
  if (openApiType === 'array') return 'array';
  if (openApiType === 'object') return 'object';
  return 'string';
}

function flattenBodySchemaToParameters(args: {
  openApiSpec: any;
  schema: NormalizedSchema;
  ctx: { seenRefs: Record<string, true> };
  basePath: string;
  parentRequired: boolean;
  forceOptional: boolean;
  unionSource?: 'oneOf' | 'anyOf';
}): ParameterDefinition[] {
  const { openApiSpec, schema, ctx, basePath, parentRequired, forceOptional, unionSource } = args;

  const effectiveUnionSource = unionSource || schema.unionSource;
  const effectiveForceOptional = forceOptional || !!schema.unionSource;

  // object with properties => recurse
  if (schema.type === 'object' && schema.properties && typeof schema.properties === 'object') {
    const params: ParameterDefinition[] = [];
    for (const key in schema.properties) {
      if (!Object.prototype.hasOwnProperty.call(schema.properties, key)) continue;
      const childRaw = schema.properties[key];
      const child = normalizeSchema(openApiSpec, childRaw, ctx);
      const path = basePath ? `${basePath}.${key}` : key;

      const isRequiredHere =
        parentRequired &&
        !effectiveForceOptional &&
        Array.isArray(schema.required) &&
        schema.required.indexOf(key) !== -1;

      if (child.type === 'object' && child.properties) {
        params.push(
          ...flattenBodySchemaToParameters({
            openApiSpec,
            schema: child,
            ctx,
            basePath: path,
            parentRequired: !!isRequiredHere,
            forceOptional: effectiveForceOptional,
            unionSource: effectiveUnionSource
          })
        );
        continue;
      }

      const descBase = child.description || '';
      const desc = effectiveUnionSource
        ? `${descBase}${descBase ? ' ' : ''}(from ${effectiveUnionSource}; required=false)`
        : descBase;

      params.push({
        name: path,
        description: desc,
        type: mapOpenApiTypeToParamType(child.type),
        required: effectiveForceOptional ? false : !!isRequiredHere,
        location: 'body',
        mapping: path,
        enum: child.enum,
        default: child.default
      });
    }
    return params;
  }

  // fallback: single payload
  const descBase = schema.description || 'Request body payload';
  const desc = effectiveUnionSource
    ? `${descBase}${descBase ? ' ' : ''}(from ${effectiveUnionSource}; required=false)`
    : descBase;

  return [
    {
      name: basePath || 'payload',
      description: desc,
      type: mapOpenApiTypeToParamType(schema.type),
      required: effectiveForceOptional ? false : !!parentRequired,
      location: 'body',
      mapping: basePath || 'payload',
      enum: schema.enum,
      default: schema.default
    }
  ];
}

/**
 * Convert OpenAPI operation to tool definition
 */
function convertOpenApiOperation(
  path: string,
  method: string,
  operation: any,
  pathItem: any,
  openApiSpec: any
): ToolDefinition {
  const name = operation.operationId || `${method}_${path.replace(/\//g, '_')}`;
  const description = operation.summary || operation.description || `${method.toUpperCase()} ${path}`;

  // Convert parameters
  const parameters: ParameterDefinition[] = [];

  // Merge path-level + operation-level parameters (dedupe by location+name)
  const mergedParams: any[] = [];
  if (pathItem && Array.isArray(pathItem.parameters)) {
    for (let i = 0; i < pathItem.parameters.length; i++) mergedParams.push(pathItem.parameters[i]);
  }
  if (operation && Array.isArray(operation.parameters)) {
    for (let i = 0; i < operation.parameters.length; i++) mergedParams.push(operation.parameters[i]);
  }

  const seenParamKeys: Record<string, true> = {};
  for (let i = 0; i < mergedParams.length; i++) {
    const converted = convertOpenApiParameter(mergedParams[i], openApiSpec);
    if (!converted) continue;
    const key = `${converted.location}:${converted.name}`;
    if (seenParamKeys[key]) continue;
    seenParamKeys[key] = true;
    parameters.push(converted);
  }

  // Handle requestBody
  if (operation.requestBody) {
    const bodyParams = convertOpenApiRequestBody(operation.requestBody, openApiSpec);
    for (let i = 0; i < bodyParams.length; i++) {
      parameters.push(bodyParams[i]);
    }
  }

  return {
    name,
    description,
    endpoint: path,
    method: method.toUpperCase() as any,
    parameters
  };
}

/**
 * Convert OpenAPI parameter to parameter definition
 */
function convertOpenApiParameter(param: any, openApiSpec: any): ParameterDefinition | null {
  if (!param || typeof param !== 'object') return null;

  // Resolve parameter $ref if present
  if (param.$ref && typeof param.$ref === 'string') {
    const resolved = getByRef(openApiSpec, param.$ref);
    if (resolved && typeof resolved === 'object') {
      param = { ...resolved, ...param };
      delete param.$ref;
    }
  }

  if (!param.name || !param.in) return null;
  if (param.in !== 'query' && param.in !== 'path' && param.in !== 'header') return null;

  const schema = param.schema || {};
  const type = mapOpenApiTypeToParamType(schema.type);

  return {
    name: param.name,
    description: param.description || '',
    type,
    required: !!param.required,
    location: param.in,
    mapping: param.name,
    enum: schema.enum,
    default: schema.default,
    pattern: schema.pattern
  };
}

/**
 * Convert OpenAPI requestBody to parameter definition
 */
function convertOpenApiRequestBody(requestBody: any, openApiSpec: any): ParameterDefinition[] {
  if (!requestBody || typeof requestBody !== 'object') return [];
  if (!requestBody.content || typeof requestBody.content !== 'object') return [];

  let contentType = 'application/json';
  let content = requestBody.content[contentType];
  if (!content) {
    const keys = Object.keys(requestBody.content);
    contentType = keys[0];
    content = requestBody.content[contentType];
  }

  const schemaRaw = content && content.schema;
  if (!schemaRaw) return [];

  const ctx = { seenRefs: {} as Record<string, true> };
  const normalized = normalizeSchema(openApiSpec, schemaRaw, ctx);
  const params = flattenBodySchemaToParameters({
    openApiSpec,
    schema: normalized,
    ctx,
    basePath: '',
    parentRequired: !!requestBody.required,
    forceOptional: false
  });

  for (let i = 0; i < params.length; i++) {
    if (!params[i].description) {
      params[i].description = requestBody.description || `Request body (${contentType})`;
    }
  }

  return params;
}
