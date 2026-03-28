import { prisma } from '@/lib/prisma';

const NAMESPACE_MAX_LENGTH = 64;
const NAMESPACE_PATTERN = /^[a-zA-Z0-9][a-zA-Z0-9._:/-]*$/;
const TAG_MAX_LENGTH = 32;
const TAG_MAX_COUNT = 50;
const TAG_PATTERN = /^[a-zA-Z0-9][a-zA-Z0-9._-]*$/;

export interface TokenMetadataInput {
  namespace?: string;
  tags?: string[];
}

export interface TokenMetadataOutput {
  namespace: string;
  tags: string[];
}

export type TagsMode = 'replace' | 'add' | 'remove' | 'clear';
export type PermissionsMode = 'replace' | 'merge';

export interface ExternalTokenPermission {
  toolId: string;
  functions?: Array<{ funcName: string; enabled: boolean }>;
  resources?: Array<{ uri: string; enabled: boolean }>;
}

export function parseExternalTokenPermissions(
  permissions?: ExternalTokenPermission[]
): Record<string, Record<string, unknown>> {
  if (!permissions || permissions.length === 0) {
    return {};
  }

  const parsed: Record<string, Record<string, unknown>> = {};

  for (const perm of permissions) {
    if (!perm || typeof perm.toolId !== 'string') {
      throw new Error('invalid toolId: expected non-empty string');
    }
    const toolId = perm.toolId.trim();
    if (!toolId) {
      throw new Error('invalid toolId: expected non-empty string');
    }
    if (parsed[toolId]) {
      throw new Error(`duplicate toolId in permissions: "${toolId}"`);
    }

    const tools: Record<string, { enabled: boolean }> = {};
    if (perm.functions) {
      for (const func of perm.functions) {
        if (!func || typeof func.funcName !== 'string') {
          throw new Error('invalid function name: expected non-empty string');
        }
        const funcName = func.funcName.trim();
        if (!funcName) {
          throw new Error('invalid function name: expected non-empty string');
        }
        if (typeof func.enabled !== 'boolean') {
          throw new Error(`invalid enabled value for function "${funcName}": expected boolean`);
        }
        tools[funcName] = { enabled: func.enabled };
      }
    }

    const resources: Record<string, { enabled: boolean }> = {};
    if (perm.resources) {
      for (const res of perm.resources) {
        if (!res || typeof res.uri !== 'string') {
          throw new Error('invalid resource uri: expected non-empty string');
        }
        const uri = res.uri.trim();
        if (!uri) {
          throw new Error('invalid resource uri: expected non-empty string');
        }
        if (typeof res.enabled !== 'boolean') {
          throw new Error(`invalid enabled value for resource "${uri}": expected boolean`);
        }
        resources[uri] = { enabled: res.enabled };
      }
    }

    parsed[toolId] = {
      enabled: true,
      tools,
      resources,
      prompts: {},
    };
  }

  return parsed;
}

export function mergeParsedPermissions(
  existing: Record<string, any>,
  incoming: Record<string, any>
): Record<string, any> {
  const merged: Record<string, any> = {};

  for (const [sid, caps] of Object.entries(existing)) {
    merged[sid] = { ...caps };
  }

  for (const [sid, newCaps] of Object.entries(incoming)) {
    if (!merged[sid]) {
      merged[sid] = newCaps;
      continue;
    }

    const current = merged[sid];
    if ((newCaps as { enabled?: boolean }).enabled !== undefined) {
      current.enabled = (newCaps as { enabled: boolean }).enabled;
    }
    if ((newCaps as { tools?: Record<string, any> }).tools) {
      current.tools = {
        ...(current.tools || {}),
        ...(newCaps as { tools: Record<string, any> }).tools,
      };
    }
    if ((newCaps as { resources?: Record<string, any> }).resources) {
      current.resources = {
        ...(current.resources || {}),
        ...(newCaps as { resources: Record<string, any> }).resources,
      };
    }
    if ((newCaps as { prompts?: Record<string, any> }).prompts) {
      current.prompts = {
        ...(current.prompts || {}),
        ...(newCaps as { prompts: Record<string, any> }).prompts,
      };
    }
  }

  return merged;
}

export function normalizeNamespace(ns: string | undefined | null): string {
  if (!ns || typeof ns !== 'string') return 'default';
  const trimmed = ns.trim().toLowerCase();
  return trimmed || 'default';
}

export function normalizeTag(tag: string): string {
  return tag.trim().toLowerCase();
}

export function normalizeTags(tags: string[] | undefined | null): string[] {
  if (!tags || !Array.isArray(tags)) return [];
  const normalized = tags
    .filter((t): t is string => typeof t === 'string')
    .map(normalizeTag)
    .filter(t => t.length > 0);
  return Array.from(new Set(normalized));
}

export function validateNamespace(ns: string): string | null {
  if (ns.length > NAMESPACE_MAX_LENGTH) {
    return `Namespace exceeds max length of ${NAMESPACE_MAX_LENGTH}`;
  }
  if (ns !== 'default' && !NAMESPACE_PATTERN.test(ns)) {
    return 'Namespace contains invalid characters. Must start with alphanumeric and contain only alphanumeric, dots, underscores, colons, slashes, or hyphens';
  }
  return null;
}

export function validateTags(tags: string[]): string | null {
  if (tags.length > TAG_MAX_COUNT) {
    return `Tags exceed max count of ${TAG_MAX_COUNT}`;
  }
  for (const tag of tags) {
    if (tag.length > TAG_MAX_LENGTH) {
      return `Tag "${tag}" exceeds max length of ${TAG_MAX_LENGTH}`;
    }
    if (!TAG_PATTERN.test(tag)) {
      return `Tag "${tag}" contains invalid characters. Must start with alphanumeric and contain only alphanumeric, dots, underscores, or hyphens`;
    }
  }
  return null;
}

export function validateMetadataInput(input: TokenMetadataInput): string | null {
  if (input.namespace !== undefined) {
    const ns = normalizeNamespace(input.namespace);
    const err = validateNamespace(ns);
    if (err) return err;
  }
  if (input.tags !== undefined) {
    const tags = normalizeTags(input.tags);
    const err = validateTags(tags);
    if (err) return err;
  }
  return null;
}

export function applyTagsOperation(
  existingTags: string[],
  submittedTags: string[],
  mode: TagsMode
): string[] {
  let result: string[];
  switch (mode) {
    case 'replace':
      result = Array.from(new Set(submittedTags));
      break;
    case 'add': {
      const merged = existingTags.concat(submittedTags);
      result = Array.from(new Set(merged));
      break;
    }
    case 'remove':
      result = existingTags.filter(t => !submittedTags.includes(t));
      break;
    case 'clear':
      result = [];
      break;
    default:
      result = existingTags;
  }
  if (result.length > TAG_MAX_COUNT) {
    const dropped = result.length - TAG_MAX_COUNT;
    const droppedSample = result.slice(TAG_MAX_COUNT, TAG_MAX_COUNT + 5);
    console.warn(`Tag operation result exceeded max count (${TAG_MAX_COUNT}); truncating ${dropped} tags`, {
      max: TAG_MAX_COUNT,
      current: result.length,
      mode,
      droppedSample,
    });
    result = result.slice(0, TAG_MAX_COUNT);
  }
  return result;
}

function parseTagsJson(raw: string, ctx: { userid: string }): string[] {
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.filter((t: unknown) => typeof t === 'string') : [];
  } catch (error) {
    console.error('Failed to parse token metadata tags', { ...ctx, error });
    return [];
  }
}

export async function getTokenMetadataMap(
  userids: string[]
): Promise<Map<string, TokenMetadataOutput>> {
  if (userids.length === 0) return new Map();

  const rows = await prisma.tokenMetadata.findMany({
    where: { userid: { in: userids } },
  });

  const map = new Map<string, TokenMetadataOutput>();
  for (const row of rows) {
    map.set(row.userid, { namespace: row.namespace, tags: parseTagsJson(row.tags, { userid: row.userid }) });
  }

  for (const userid of userids) {
    if (!map.has(userid)) {
      map.set(userid, { namespace: 'default', tags: [] });
    }
  }

  return map;
}

export async function getTokenMetadata(
  userid: string
): Promise<TokenMetadataOutput> {
  const row = await prisma.tokenMetadata.findUnique({
    where: { userid },
  });
  if (!row) return { namespace: 'default', tags: [] };

  return { namespace: row.namespace, tags: parseTagsJson(row.tags, { userid }) };
}

export async function upsertTokenMetadata(
  userid: string,
  input: TokenMetadataInput
): Promise<void> {
  const ns = normalizeNamespace(input.namespace);
  const tags = normalizeTags(input.tags);

  await prisma.tokenMetadata.upsert({
    where: { userid },
    create: {
      userid,
      namespace: ns,
      tags: JSON.stringify(tags),
    },
    update: {
      ...(input.namespace !== undefined ? { namespace: ns } : {}),
      ...(input.tags !== undefined ? { tags: JSON.stringify(tags) } : {}),
    },
  });
}

export async function deleteTokenMetadata(
  userid: string
): Promise<void> {
  try {
    await prisma.tokenMetadata.delete({
      where: { userid },
    });
  } catch (error: any) {
    if (error?.code === 'P2025') return;
    throw error;
  }
}
