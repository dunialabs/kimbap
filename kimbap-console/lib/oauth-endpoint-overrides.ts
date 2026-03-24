export const OAUTH_AUTHORIZATION_URL_KEY = 'YOUR_OAUTH_AUTHORIZATION_URL'
export const OAUTH_TOKEN_URL_KEY = 'YOUR_OAUTH_TOKEN_URL'
export const OAUTH_BASE_URL_KEY = 'YOUR_OAUTH_BASE_URL'

export const OAUTH_ENDPOINT_OVERRIDE_KEYS = [
  OAUTH_AUTHORIZATION_URL_KEY,
  OAUTH_TOKEN_URL_KEY,
  OAUTH_BASE_URL_KEY
] as const

interface AuthConfEntryLike {
  key?: string
  value?: string
}

interface OAuthEndpointTemplate {
  authorizationUrl: string
  tokenUrl: string
}

export interface OAuthEndpointOverrides {
  authorizationUrl?: string
  tokenUrl?: string
  baseUrl?: string
}

export interface ResolvedOAuthEndpoints {
  authorizationUrl: string
  tokenUrl: string
  usingOverride: boolean
}

const URL_PROTOCOL_PATTERN = /^[a-zA-Z][a-zA-Z\d+\-.]*:\/\//

const sanitizeOverrideValue = (
  value: string | undefined,
  placeholderKey: string
): string | undefined => {
  if (!value) return undefined

  const trimmed = value.trim()
  if (trimmed === '' || trimmed === placeholderKey) {
    return undefined
  }

  return trimmed
}

const normalizeUrl = (value: string, label: string): string => {
  const candidate = value.trim()
  const withProtocol = URL_PROTOCOL_PATTERN.test(candidate)
    ? candidate
    : `https://${candidate}`

  try {
    const parsed = new URL(withProtocol)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      throw new Error(`${label} must use http or https`)
    }
    return parsed.toString()
  } catch {
    throw new Error(`Invalid ${label}`)
  }
}

const normalizeBaseUrl = (value: string): string => {
  const normalized = new URL(normalizeUrl(value, 'OAuth base URL'))
  normalized.search = ''
  normalized.hash = ''

  const base = normalized.toString()
  return base.endsWith('/') ? base.slice(0, -1) : base
}

const composeFromBaseUrl = (
  templateUrl: string,
  normalizedBaseUrl: string,
  label: string
): string => {
  if (!templateUrl || templateUrl.trim() === '') {
    throw new Error(`${label} template is missing`)
  }

  if (templateUrl.includes(OAUTH_BASE_URL_KEY)) {
    const replaced = templateUrl
      .split(OAUTH_BASE_URL_KEY)
      .join(normalizedBaseUrl)
    return normalizeUrl(replaced, label)
  }

  const templateParsed = new URL(normalizeUrl(templateUrl, label))
  const baseParsed = new URL(normalizedBaseUrl)

  templateParsed.protocol = baseParsed.protocol
  templateParsed.username = baseParsed.username
  templateParsed.password = baseParsed.password
  templateParsed.host = baseParsed.host

  if (baseParsed.pathname && baseParsed.pathname !== '/') {
    const basePath = baseParsed.pathname.endsWith('/')
      ? baseParsed.pathname.slice(0, -1)
      : baseParsed.pathname
    const templatePath = templateParsed.pathname.startsWith('/')
      ? templateParsed.pathname
      : `/${templateParsed.pathname}`
    templateParsed.pathname = `${basePath}${templatePath}`.replace(/\/{2,}/g, '/')
  }

  return templateParsed.toString()
}

export const isOAuthEndpointOverrideKey = (key: string): boolean =>
  OAUTH_ENDPOINT_OVERRIDE_KEYS.includes(key as (typeof OAUTH_ENDPOINT_OVERRIDE_KEYS)[number])

export function extractOAuthEndpointOverridesFromAuthConf(
  authConf?: AuthConfEntryLike[]
): OAuthEndpointOverrides {
  const overrides: OAuthEndpointOverrides = {}

  for (const auth of authConf || []) {
    if (!auth.key || !auth.value) continue

    if (auth.key === OAUTH_AUTHORIZATION_URL_KEY) {
      overrides.authorizationUrl = sanitizeOverrideValue(
        auth.value,
        OAUTH_AUTHORIZATION_URL_KEY
      )
    } else if (auth.key === OAUTH_TOKEN_URL_KEY) {
      overrides.tokenUrl = sanitizeOverrideValue(
        auth.value,
        OAUTH_TOKEN_URL_KEY
      )
    } else if (auth.key === OAUTH_BASE_URL_KEY) {
      overrides.baseUrl = sanitizeOverrideValue(auth.value, OAUTH_BASE_URL_KEY)
    }
  }

  return overrides
}

export function resolveOAuthEndpoints(
  template: OAuthEndpointTemplate,
  overrides: OAuthEndpointOverrides
): ResolvedOAuthEndpoints {
  const authorizationUrl = sanitizeOverrideValue(
    overrides.authorizationUrl,
    OAUTH_AUTHORIZATION_URL_KEY
  )
  const tokenUrl = sanitizeOverrideValue(
    overrides.tokenUrl,
    OAUTH_TOKEN_URL_KEY
  )
  const baseUrl = sanitizeOverrideValue(overrides.baseUrl, OAUTH_BASE_URL_KEY)

  if ((authorizationUrl && !tokenUrl) || (!authorizationUrl && tokenUrl)) {
    throw new Error(
      'Custom OAuth endpoints require both authorization URL and token URL'
    )
  }

  if (authorizationUrl && tokenUrl) {
    return {
      authorizationUrl: normalizeUrl(authorizationUrl, 'OAuth authorization URL'),
      tokenUrl: normalizeUrl(tokenUrl, 'OAuth token URL'),
      usingOverride: true
    }
  }

  if (baseUrl) {
    const normalizedBaseUrl = normalizeBaseUrl(baseUrl)
    return {
      authorizationUrl: composeFromBaseUrl(
        template.authorizationUrl,
        normalizedBaseUrl,
        'OAuth authorization URL'
      ),
      tokenUrl: composeFromBaseUrl(
        template.tokenUrl,
        normalizedBaseUrl,
        'OAuth token URL'
      ),
      usingOverride: true
    }
  }

  return {
    authorizationUrl: normalizeUrl(
      template.authorizationUrl,
      'OAuth authorization URL'
    ),
    tokenUrl: normalizeUrl(template.tokenUrl, 'OAuth token URL'),
    usingOverride: false
  }
}
