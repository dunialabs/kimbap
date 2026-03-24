/**
 * OAuth Authorization Library
 * Unified OAuth flow handling for KIMBAP Console
 */

export interface OAuthConfig {
  authorizationUrl: string
  tokenUrl: string
  clientId: string
  userClientId?: string // Client ID for user mode (allowUserInput === 1)
  scopes: string
  responseType: string
  extraParams?: Record<string, string>
  applyUrl?: string // URL to apply for credentials
  requiresCustomApp?: boolean // true: users must provide their own OAuth app credentials
  pkce?: {
    required?: boolean
    method?: 'S256' | 'plain'
  }
}

export interface OAuthSessionData {
  templateId: string
  serverType: string
  tempId: string
  credentialSource: 'kimbap' | 'custom'
  credentials?: Record<string, string> // User-filled credentials (custom mode only)
  allowUserInput: number
  redirectUri: string // The redirect URI used for OAuth flow
  pkceVerifier?: string
  lazyStartEnabled?: boolean
  publicAccess?: boolean
  anonymousAccess?: boolean
  anonymousRateLimit?: number
}

const OAUTH_SESSION_PREFIX = 'oauth_session_'
const PKCE_VERIFIER_CHARSET =
  'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~'
const PKCE_VERIFIER_LENGTH = 96

export type OAuthPKCEMethod = 'S256' | 'plain'

export interface OAuthPKCEConfig {
  required: boolean
  method: OAuthPKCEMethod
}

const encodeBase64Url = (input: Uint8Array): string => {
  let binary = ''
  for (const byte of input) {
    binary += String.fromCharCode(byte)
  }

  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

const generateCodeVerifier = (): string => {
  const randomBytes = new Uint8Array(PKCE_VERIFIER_LENGTH)
  crypto.getRandomValues(randomBytes)

  let verifier = ''
  for (const randomByte of randomBytes) {
    verifier += PKCE_VERIFIER_CHARSET[randomByte % PKCE_VERIFIER_CHARSET.length]
  }

  return verifier
}

const generateS256CodeChallenge = async (codeVerifier: string): Promise<string> => {
  const data = new TextEncoder().encode(codeVerifier)
  const digest = await crypto.subtle.digest('SHA-256', data)
  return encodeBase64Url(new Uint8Array(digest))
}

export const getOAuthPKCEConfig = (config: OAuthConfig): OAuthPKCEConfig => {
  const required = Boolean(config.pkce?.required)
  if (!required) {
    return { required: false, method: 'S256' }
  }

  const method = config.pkce?.method || 'S256'
  if (method !== 'S256' && method !== 'plain') {
    throw new Error(`Unsupported PKCE method: ${method}`)
  }

  return {
    required: true,
    method
  }
}

export const generateOAuthPKCEParams = async (
  method: OAuthPKCEMethod
): Promise<{ codeVerifier: string; codeChallenge: string }> => {
  const codeVerifier = generateCodeVerifier()

  if (method === 'plain') {
    return {
      codeVerifier,
      codeChallenge: codeVerifier
    }
  }

  return {
    codeVerifier,
    codeChallenge: await generateS256CodeChallenge(codeVerifier)
  }
}

/**
 * Build the authorization URL for OAuth flow
 */
export function buildAuthorizationUrl(
  config: OAuthConfig,
  customClientId: string | undefined,
  redirectUri: string,
  state: string
): string {
  const clientId = customClientId || config.clientId
  const url = new URL(config.authorizationUrl)

  url.searchParams.set('client_id', clientId)
  url.searchParams.set('redirect_uri', redirectUri)
  url.searchParams.set('response_type', config.responseType)
  url.searchParams.set('state', state)

  if (config.scopes && config.scopes !== '') {
    url.searchParams.set('scope', config.scopes)
  }

  // Add extra params if any
  if (config.extraParams) {
    for (const [key, value] of Object.entries(config.extraParams)) {
      url.searchParams.set(key, value)
    }
  }

  return url.toString()
}

/**
 * Parse authorization callback URL parameters
 */
export function parseAuthorizationCallback(url: string): {
  code: string | null
  state: string | null
  error: string | null
} {
  try {
    const urlObj = new URL(url)
    return {
      code: urlObj.searchParams.get('code'),
      state: urlObj.searchParams.get('state'),
      error: urlObj.searchParams.get('error')
    }
  } catch {
    return {
      code: null,
      state: null,
      error: 'Invalid URL'
    }
  }
}

/**
 * Generate random state and store session data in sessionStorage
 * @returns The generated state string
 */
export function saveOAuthSession(data: OAuthSessionData): string {
  const state = crypto.randomUUID()
  if (typeof window !== 'undefined') {
    sessionStorage.setItem(
      `${OAUTH_SESSION_PREFIX}${state}`,
      JSON.stringify(data)
    )
  }
  return state
}

/**
 * Get session data by state
 */
export function getOAuthSession(state: string): OAuthSessionData | null {
  if (typeof window === 'undefined') return null

  const data = sessionStorage.getItem(`${OAUTH_SESSION_PREFIX}${state}`)
  if (!data) return null

  try {
    return JSON.parse(data) as OAuthSessionData
  } catch {
    return null
  }
}

/**
 * Clear session data by state
 */
export function clearOAuthSession(state: string): void {
  if (typeof window !== 'undefined') {
    sessionStorage.removeItem(`${OAUTH_SESSION_PREFIX}${state}`)
  }
}

/**
 * Check if there is a pending OAuth session (for handling browser back button)
 */
export function hasPendingOAuthSession(): boolean {
  if (typeof window === 'undefined') return false

  for (let i = 0; i < sessionStorage.length; i++) {
    const key = sessionStorage.key(i)
    if (key?.startsWith(OAUTH_SESSION_PREFIX)) {
      return true
    }
  }
  return false
}

/**
 * Clear all pending OAuth sessions
 */
export function clearAllOAuthSessions(): void {
  if (typeof window === 'undefined') return

  const keysToRemove: string[] = []
  for (let i = 0; i < sessionStorage.length; i++) {
    const key = sessionStorage.key(i)
    if (key?.startsWith(OAUTH_SESSION_PREFIX)) {
      keysToRemove.push(key)
    }
  }

  keysToRemove.forEach((key) => sessionStorage.removeItem(key))
}
